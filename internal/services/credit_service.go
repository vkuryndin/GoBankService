package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
)

var (
	ErrCreditNotFound                    = errors.New("credit not found")
	ErrInvalidCreditData                 = errors.New("invalid credit data")
	ErrActiveOverdueCreditExists         = errors.New("active overdue credit exists")
	ErrActiveCreditLimitExceeded         = errors.New("active credit limit exceeded")
	ErrCreditPrincipalLimitExceeded      = errors.New("credit principal limit exceeded")
	ErrCreditTotalPrincipalLimitExceeded = errors.New("credit total principal limit exceeded")
	ErrCreditDebtLoadTooHigh             = errors.New("credit debt load too high")
)

type CreditPolicy struct {
	Enabled                 bool
	MaxActiveCredits        int
	MaxPrincipalAmount      string
	MaxTotalPrincipalAmount string
	MaxDebtLoadPercent      int
}

type creditStore interface {
	CreateWithScheduleAndIssue(ctx context.Context, userID int64, accountID int64, principalAmount string, interestRate string, termMonths int, monthlyPayment string, schedule []repositories.PaymentScheduleInput) (*models.Credit, error)
	FindByUserID(ctx context.Context, userID int64) ([]models.Credit, error)
	FindByIDAndUserID(ctx context.Context, creditID, userID int64) (*models.Credit, error)
	FindScheduleByCreditIDAndUserID(ctx context.Context, creditID, userID int64) ([]models.PaymentSchedule, error)
	GetCreditRiskSummary(ctx context.Context, userID int64) (*repositories.CreditRiskSummary, error)
}

type bankRateProvider interface {
	GetBankRateValue(ctx context.Context) (float64, error)
}

type creditMFAVerifier interface {
	VerifyCreditCreateCode(ctx context.Context, userID int64, request dto.CreateCreditRequest) error
}

type CreditService struct {
	creditRepository creditStore
	rateService      bankRateProvider
	mfaService       creditMFAVerifier
	policy           CreditPolicy
}

func NewCreditService(
	creditRepository creditStore,
	rateService bankRateProvider,
	mfaService creditMFAVerifier,
	policy CreditPolicy,
) *CreditService {
	return &CreditService{
		creditRepository: creditRepository,
		rateService:      rateService,
		mfaService:       mfaService,
		policy:           policy,
	}
}

func (s *CreditService) CreateCredit(ctx context.Context, userID int64, request dto.CreateCreditRequest) (*dto.CreditResponse, error) {
	if request.AccountID <= 0 {
		return nil, ErrInvalidCreditData
	}

	if request.TermMonths <= 0 || request.TermMonths > maxCreditTermMonths {
		return nil, ErrInvalidCreditData
	}

	principalAmount, err := normalizeAmount(request.PrincipalAmount)
	if err != nil {
		return nil, err
	}

	principalValue, err := strconv.ParseFloat(principalAmount, 64)
	if err != nil {
		return nil, ErrInvalidAmount
	}

	annualRate, err := s.rateService.GetBankRateValue(ctx)
	if err != nil {
		return nil, fmt.Errorf("get bank rate: %w", err)
	}

	monthlyPaymentValue := calculateAnnuityPayment(principalValue, annualRate, request.TermMonths)
	monthlyPayment := formatMoney(monthlyPaymentValue)
	interestRate := formatPercent(annualRate)

	if err := s.checkCreditPolicy(ctx, userID, principalAmount, monthlyPayment); err != nil {
		return nil, err
	}

	// MFA is verified after the credit policy check so a valid one-time code is not consumed
	// when the bank would reject the credit anyway.
	if err := s.mfaService.VerifyCreditCreateCode(ctx, userID, request); err != nil {
		return nil, err
	}

	schedule := buildPaymentSchedule(request.TermMonths, monthlyPayment)

	credit, err := s.creditRepository.CreateWithScheduleAndIssue(
		ctx,
		userID,
		request.AccountID,
		principalAmount,
		interestRate,
		request.TermMonths,
		monthlyPayment,
		schedule,
	)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}
		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}

		return nil, err
	}

	return toCreditResponse(credit), nil
}

func (s *CreditService) checkCreditPolicy(ctx context.Context, userID int64, principalAmount string, monthlyPayment string) error {
	if !s.policy.Enabled {
		return nil
	}

	principal, err := moneyRat(principalAmount)
	if err != nil {
		return ErrInvalidAmount
	}
	newMonthlyPayment, err := moneyRat(monthlyPayment)
	if err != nil {
		return ErrInvalidAmount
	}

	if s.policy.MaxPrincipalAmount != "" {
		maxPrincipal, err := moneyRat(s.policy.MaxPrincipalAmount)
		if err != nil {
			return ErrInvalidCreditData
		}
		if principal.Cmp(maxPrincipal) > 0 {
			return ErrCreditPrincipalLimitExceeded
		}
	}

	summary, err := s.creditRepository.GetCreditRiskSummary(ctx, userID)
	if err != nil {
		return err
	}

	if summary.OverdueCreditsCount > 0 {
		return ErrActiveOverdueCreditExists
	}

	if s.policy.MaxActiveCredits > 0 && summary.ActiveCreditsCount >= s.policy.MaxActiveCredits {
		return ErrActiveCreditLimitExceeded
	}

	if s.policy.MaxTotalPrincipalAmount != "" {
		totalPrincipal, err := moneyRat(summary.TotalPrincipalAmount)
		if err != nil {
			return err
		}
		maxTotalPrincipal, err := moneyRat(s.policy.MaxTotalPrincipalAmount)
		if err != nil {
			return ErrInvalidCreditData
		}
		if new(big.Rat).Add(totalPrincipal, principal).Cmp(maxTotalPrincipal) > 0 {
			return ErrCreditTotalPrincipalLimitExceeded
		}
	}

	// Debt load is checked only when the user has actual income in the last 30 days.
	// This keeps local demos usable while still rejecting risky extra borrowing for active customers.
	if s.policy.MaxDebtLoadPercent > 0 {
		monthlyIncome, err := moneyRat(summary.MonthlyIncome)
		if err != nil {
			return err
		}
		if monthlyIncome.Sign() > 0 {
			existingMonthlyPayment, err := moneyRat(summary.TotalMonthlyPayment)
			if err != nil {
				return err
			}

			allowedDebtLoad := new(big.Rat).Mul(monthlyIncome, big.NewRat(int64(s.policy.MaxDebtLoadPercent), 100))
			newDebtLoad := new(big.Rat).Add(existingMonthlyPayment, newMonthlyPayment)
			if newDebtLoad.Cmp(allowedDebtLoad) > 0 {
				return ErrCreditDebtLoadTooHigh
			}
		}
	}

	return nil
}

func (s *CreditService) GetUserCredits(ctx context.Context, userID int64) ([]dto.CreditResponse, error) {
	credits, err := s.creditRepository.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.CreditResponse, 0, len(credits))
	for _, credit := range credits {
		responses = append(responses, *toCreditResponse(&credit))
	}

	return responses, nil
}

func (s *CreditService) GetCredit(ctx context.Context, userID, creditID int64) (*dto.CreditResponse, error) {
	credit, err := s.creditRepository.FindByIDAndUserID(ctx, creditID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrCreditNotFound) {
			return nil, ErrCreditNotFound
		}

		return nil, err
	}

	return toCreditResponse(credit), nil
}

func (s *CreditService) GetCreditSchedule(
	ctx context.Context,
	userID int64,
	creditID int64,
) ([]dto.PaymentScheduleResponse, error) {
	schedule, err := s.creditRepository.FindScheduleByCreditIDAndUserID(ctx, creditID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrCreditNotFound) {
			return nil, ErrCreditNotFound
		}

		return nil, err
	}

	responses := make([]dto.PaymentScheduleResponse, 0, len(schedule))
	for _, payment := range schedule {
		responses = append(responses, *toPaymentScheduleResponse(&payment))
	}

	return responses, nil
}

func calculateAnnuityPayment(principal float64, annualRate float64, termMonths int) float64 {
	monthlyRate := annualRate / 100 / 12

	if monthlyRate == 0 {
		return principal / float64(termMonths)
	}

	pow := math.Pow(1+monthlyRate, float64(termMonths))

	return principal * monthlyRate * pow / (pow - 1)
}

func buildPaymentSchedule(termMonths int, monthlyPayment string) []repositories.PaymentScheduleInput {
	schedule := make([]repositories.PaymentScheduleInput, 0, termMonths)
	now := time.Now()

	for i := 1; i <= termMonths; i++ {
		schedule = append(schedule, repositories.PaymentScheduleInput{
			PaymentDate: now.AddDate(0, i, 0),
			Amount:      monthlyPayment,
		})
	}

	return schedule
}

func toCreditResponse(credit *models.Credit) *dto.CreditResponse {
	return &dto.CreditResponse{
		ID:              credit.ID,
		AccountID:       credit.AccountID,
		PrincipalAmount: credit.PrincipalAmount,
		InterestRate:    credit.InterestRate,
		TermMonths:      credit.TermMonths,
		MonthlyPayment:  credit.MonthlyPayment,
		Status:          credit.Status,
		CreatedAt:       credit.CreatedAt.Format(time.RFC3339),
	}
}

func toPaymentScheduleResponse(payment *models.PaymentSchedule) *dto.PaymentScheduleResponse {
	response := &dto.PaymentScheduleResponse{
		ID:            payment.ID,
		CreditID:      payment.CreditID,
		PaymentDate:   payment.PaymentDate.Format("2006-01-02"),
		Amount:        payment.Amount,
		PenaltyAmount: payment.PenaltyAmount,
		Status:        payment.Status,
	}

	if payment.PaidAt.Valid {
		response.PaidAt = payment.PaidAt.Time.Format(time.RFC3339)
	}

	return response
}

func moneyRat(amount string) (*big.Rat, error) {
	value, ok := new(big.Rat).SetString(amount)
	if !ok {
		return nil, ErrInvalidAmount
	}
	return value, nil
}

func formatMoney(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
