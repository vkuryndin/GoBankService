package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
)

var (
	ErrCreditNotFound    = errors.New("credit not found")
	ErrInvalidCreditData = errors.New("invalid credit data")
)

type creditStore interface {
	CreateWithScheduleAndIssue(ctx context.Context, userID int64, accountID int64, principalAmount string, interestRate string, termMonths int, monthlyPayment string, schedule []repositories.PaymentScheduleInput) (*models.Credit, error)
	FindByUserID(ctx context.Context, userID int64) ([]models.Credit, error)
	FindByIDAndUserID(ctx context.Context, creditID, userID int64) (*models.Credit, error)
	FindScheduleByCreditIDAndUserID(ctx context.Context, creditID, userID int64) ([]models.PaymentSchedule, error)
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
}

func NewCreditService(
	creditRepository creditStore,
	rateService bankRateProvider,
	mfaService creditMFAVerifier,
) *CreditService {
	return &CreditService{
		creditRepository: creditRepository,
		rateService:      rateService,
		mfaService:       mfaService,
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

	if err := s.mfaService.VerifyCreditCreateCode(ctx, userID, request); err != nil {
		return nil, err
	}

	annualRate, err := s.rateService.GetBankRateValue(ctx)
	if err != nil {
		return nil, fmt.Errorf("get bank rate: %w", err)
	}

	monthlyPaymentValue := calculateAnnuityPayment(principalValue, annualRate, request.TermMonths)
	monthlyPayment := formatMoney(monthlyPaymentValue)
	interestRate := formatPercent(annualRate)

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

func formatMoney(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
