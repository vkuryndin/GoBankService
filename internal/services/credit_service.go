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

type CreditPolicyError struct {
	Err     error
	Details *dto.CreditCheckResponse
}

func (e *CreditPolicyError) Error() string {
	return e.Err.Error()
}

func (e *CreditPolicyError) Unwrap() error {
	return e.Err
}

func (e *CreditPolicyError) PublicDetails() any {
	return e.Details
}

type creditStore interface {
	CreateWithScheduleAndIssue(ctx context.Context, userID int64, accountID int64, principalAmount string, interestRate string, termMonths int, monthlyPayment string, schedule []repositories.PaymentScheduleInput, mfaCodeID int64, validatePolicy repositories.CreditPolicyValidator) (*models.Credit, error)
	FindByUserID(ctx context.Context, userID int64) ([]models.Credit, error)
	FindByIDAndUserID(ctx context.Context, creditID, userID int64) (*models.Credit, error)
	FindScheduleByCreditIDAndUserID(ctx context.Context, creditID, userID int64) ([]models.PaymentSchedule, error)
	GetCreditRiskSummary(ctx context.Context, userID int64) (*repositories.CreditRiskSummary, error)
}

type creditAccountStore interface {
	FindByIDAndUserID(ctx context.Context, accountID, userID int64) (*models.Account, error)
}

type bankRateProvider interface {
	GetBankRateValue(ctx context.Context) (float64, error)
}

type creditMFAVerifier interface {
	VerifyCreditCreateCode(ctx context.Context, userID int64, request dto.CreateCreditRequest) (*MFAVerification, error)
}

type CreditService struct {
	creditRepository  creditStore
	accountRepository creditAccountStore
	rateService       bankRateProvider
	mfaService        creditMFAVerifier
	policy            CreditPolicy
}

func NewCreditService(
	creditRepository creditStore,
	accountRepository creditAccountStore,
	rateService bankRateProvider,
	mfaService creditMFAVerifier,
	policy CreditPolicy,
) *CreditService {
	return &CreditService{
		creditRepository:  creditRepository,
		accountRepository: accountRepository,
		rateService:       rateService,
		mfaService:        mfaService,
		policy:            policy,
	}
}

func (s *CreditService) CreateCredit(ctx context.Context, userID int64, request dto.CreateCreditRequest) (*dto.CreditResponse, error) {
	checkRequest := dto.CheckCreditRequest{
		AccountID:       request.AccountID,
		PrincipalAmount: request.PrincipalAmount,
		TermMonths:      request.TermMonths,
	}

	decision, err := s.CheckCredit(ctx, userID, checkRequest)
	if err != nil {
		return nil, err
	}

	if !decision.Eligible {
		return nil, &CreditPolicyError{
			Err:     creditPolicyReasonError(decision.Reason),
			Details: decision,
		}
	}

	// MFA is verified after the first credit policy check so a valid one-time code is not consumed
	// when the bank would reject the credit anyway. The code is consumed inside the final DB transaction.
	verification, err := s.mfaService.VerifyCreditCreateCode(ctx, userID, request)
	if err != nil {
		return nil, err
	}

	schedule := buildPaymentSchedule(request.TermMonths, decision.MonthlyPayment)
	validatePolicy := func(summary *repositories.CreditRiskSummary) error {
		freshDecision, err := s.buildCreditPolicyDecisionFromSummary(
			summary,
			request.AccountID,
			decision.PrincipalAmount,
			request.TermMonths,
			decision.InterestRate,
			decision.MonthlyPayment,
		)
		if err != nil {
			return err
		}

		if !freshDecision.Eligible {
			return &CreditPolicyError{
				Err:     creditPolicyReasonError(freshDecision.Reason),
				Details: freshDecision,
			}
		}

		return nil
	}

	credit, err := s.creditRepository.CreateWithScheduleAndIssue(
		ctx,
		userID,
		request.AccountID,
		decision.PrincipalAmount,
		decision.InterestRate,
		request.TermMonths,
		decision.MonthlyPayment,
		schedule,
		verification.CodeID,
		validatePolicy,
	)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}
		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}

		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			return nil, ErrInvalidMFACode
		}

		if errors.Is(err, repositories.ErrAccountClosed) {
			return nil, ErrAccountClosed
		}

		return nil, err
	}

	return toCreditResponse(credit), nil
}

func (s *CreditService) CheckCredit(
	ctx context.Context,
	userID int64,
	request dto.CheckCreditRequest,
) (*dto.CreditCheckResponse, error) {
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

	account, err := s.accountRepository.FindByIDAndUserID(ctx, request.AccountID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}
	if account.IsBlocked {
		return nil, ErrAccountBlocked
	}

	if account.IsClosed() {
		return nil, ErrAccountClosed
	}

	annualRate, err := s.rateService.GetBankRateValue(ctx)
	if err != nil {
		return nil, fmt.Errorf("get bank rate: %w", err)
	}

	monthlyPaymentValue := calculateAnnuityPayment(principalValue, annualRate, request.TermMonths)
	monthlyPayment := formatMoney(monthlyPaymentValue)

	decision, err := s.buildCreditPolicyDecision(
		ctx,
		userID,
		request.AccountID,
		principalAmount,
		request.TermMonths,
		formatPercent(annualRate),
		monthlyPayment,
	)
	if err != nil {
		return nil, err
	}

	return decision, nil
}

func (s *CreditService) buildCreditPolicyDecision(
	ctx context.Context,
	userID int64,
	accountID int64,
	principalAmount string,
	termMonths int,
	interestRate string,
	monthlyPayment string,
) (*dto.CreditCheckResponse, error) {
	summary, err := s.creditRepository.GetCreditRiskSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.buildCreditPolicyDecisionFromSummary(summary, accountID, principalAmount, termMonths, interestRate, monthlyPayment)
}

func (s *CreditService) buildCreditPolicyDecisionFromSummary(
	summary *repositories.CreditRiskSummary,
	accountID int64,
	principalAmount string,
	termMonths int,
	interestRate string,
	monthlyPayment string,
) (*dto.CreditCheckResponse, error) {
	principal, err := moneyRat(principalAmount)
	if err != nil {
		return nil, ErrInvalidAmount
	}
	requestedMonthlyPayment, err := moneyRat(monthlyPayment)
	if err != nil {
		return nil, ErrInvalidAmount
	}

	monthlyIncome, err := moneyRat(summary.MonthlyIncome)
	if err != nil {
		return nil, err
	}
	existingMonthlyPayment, err := moneyRat(summary.TotalMonthlyPayment)
	if err != nil {
		return nil, err
	}
	totalPrincipal, err := moneyRat(summary.TotalPrincipalAmount)
	if err != nil {
		return nil, err
	}

	totalMonthlyPayments := new(big.Rat).Add(existingMonthlyPayment, requestedMonthlyPayment)
	maxAllowedMonthlyPayments := big.NewRat(0, 1)
	debtLoadPercent := "0.00"
	if monthlyIncome.Sign() > 0 && s.policy.MaxDebtLoadPercent > 0 {
		maxAllowedMonthlyPayments = new(big.Rat).Mul(monthlyIncome, big.NewRat(int64(s.policy.MaxDebtLoadPercent), 100))
		debtLoadPercent = formatPercentRat(totalMonthlyPayments, monthlyIncome)
	}

	decision := &dto.CreditCheckResponse{
		Eligible:                  true,
		PolicyEnabled:             s.policy.Enabled,
		AccountID:                 accountID,
		PrincipalAmount:           principalAmount,
		MaxPrincipalAmount:        s.policy.MaxPrincipalAmount,
		TermMonths:                termMonths,
		InterestRate:              interestRate,
		MonthlyPayment:            monthlyPayment,
		ActiveCreditsCount:        summary.ActiveCreditsCount,
		MaxActiveCredits:          s.policy.MaxActiveCredits,
		HasOverdueCredit:          summary.OverdueCreditsCount > 0,
		TotalPrincipalAmount:      summary.TotalPrincipalAmount,
		MaxTotalPrincipalAmount:   s.policy.MaxTotalPrincipalAmount,
		MonthlyIncome:             summary.MonthlyIncome,
		CurrentMonthlyPayments:    summary.TotalMonthlyPayment,
		RequestedMonthlyPayment:   monthlyPayment,
		TotalMonthlyPayments:      formatMoneyRat(totalMonthlyPayments),
		MaxAllowedMonthlyPayments: formatMoneyRat(maxAllowedMonthlyPayments),
		DebtLoadPercent:           debtLoadPercent,
		MaxDebtLoadPercent:        s.policy.MaxDebtLoadPercent,
	}

	if !s.policy.Enabled {
		return decision, nil
	}

	if s.policy.MaxPrincipalAmount != "" {
		maxPrincipal, err := moneyRat(s.policy.MaxPrincipalAmount)
		if err != nil {
			return nil, ErrInvalidCreditData
		}
		if principal.Cmp(maxPrincipal) > 0 {
			addCreditDecisionReason(decision, ErrCreditPrincipalLimitExceeded)
		}
	}

	if summary.OverdueCreditsCount > 0 {
		addCreditDecisionReason(decision, ErrActiveOverdueCreditExists)
	}

	if s.policy.MaxActiveCredits > 0 && summary.ActiveCreditsCount >= s.policy.MaxActiveCredits {
		addCreditDecisionReason(decision, ErrActiveCreditLimitExceeded)
	}

	if s.policy.MaxTotalPrincipalAmount != "" {
		maxTotalPrincipal, err := moneyRat(s.policy.MaxTotalPrincipalAmount)
		if err != nil {
			return nil, ErrInvalidCreditData
		}
		if new(big.Rat).Add(totalPrincipal, principal).Cmp(maxTotalPrincipal) > 0 {
			addCreditDecisionReason(decision, ErrCreditTotalPrincipalLimitExceeded)
		}
	}

	// Debt load is checked only when the user has actual income in the last 30 days.
	// This keeps local demos usable while still rejecting risky extra borrowing for active customers.
	if s.policy.MaxDebtLoadPercent > 0 && monthlyIncome.Sign() > 0 {
		if totalMonthlyPayments.Cmp(maxAllowedMonthlyPayments) > 0 {
			addCreditDecisionReason(decision, ErrCreditDebtLoadTooHigh)
		}
	}

	return decision, nil
}

func addCreditDecisionReason(decision *dto.CreditCheckResponse, reason error) {
	decision.Eligible = false
	message := reason.Error()
	if decision.Reason == "" {
		decision.Reason = message
	}
	decision.Reasons = append(decision.Reasons, message)
}

func creditPolicyReasonError(reason string) error {
	switch reason {
	case ErrActiveOverdueCreditExists.Error():
		return ErrActiveOverdueCreditExists
	case ErrActiveCreditLimitExceeded.Error():
		return ErrActiveCreditLimitExceeded
	case ErrCreditPrincipalLimitExceeded.Error():
		return ErrCreditPrincipalLimitExceeded
	case ErrCreditTotalPrincipalLimitExceeded.Error():
		return ErrCreditTotalPrincipalLimitExceeded
	case ErrCreditDebtLoadTooHigh.Error():
		return ErrCreditDebtLoadTooHigh
	default:
		return ErrInvalidCreditData
	}
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

func formatMoneyRat(value *big.Rat) string {
	asFloat, _ := value.Float64()
	return formatMoney(asFloat)
}

func formatPercentRat(numerator *big.Rat, denominator *big.Rat) string {
	if denominator.Sign() == 0 {
		return "0.00"
	}

	percent := new(big.Rat).Quo(numerator, denominator)
	percent.Mul(percent, big.NewRat(100, 1))
	asFloat, _ := percent.Float64()
	return formatPercent(asFloat)
}

func formatMoney(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
