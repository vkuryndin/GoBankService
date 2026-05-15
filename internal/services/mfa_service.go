package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
	"bank-service/internal/security"
)

const (
	MFAPurposeTransfer         = "transfer"
	MFAPurposeCardPayment      = "card_payment"
	MFAPurposeCardTransfer     = "card_transfer"
	MFAPurposeCardReveal       = "card_reveal"
	MFAPurposeCreditCreate     = "credit_create"
	MFAPurposeCreditPrepayment = "credit_prepayment"
	MFAPurposeWithdraw         = "withdraw"
)

var (
	ErrInvalidMFAPurpose     = errors.New("invalid mfa purpose")
	ErrMFACodeRequired       = errors.New("mfa code required")
	ErrInvalidMFACode        = errors.New("invalid mfa code")
	ErrInvalidMFAOperation   = errors.New("invalid mfa operation")
	ErrMFARequestTooFrequent = errors.New("mfa request too frequent")
)

type MFAVerification struct {
	CodeID        int64
	Purpose       string
	OperationHash string
}

type mfaCodeStore interface {
	SaveCode(ctx context.Context, userID int64, purpose string, operationHash string, codeHash string, expiresAt time.Time) error
	FindActiveCode(ctx context.Context, userID int64, purpose string, operationHash string) (*repositories.MFACode, error)
	MarkUsed(ctx context.Context, codeID int64) error
}

type mfaAccountStore interface {
	ValidateTransferAccounts(ctx context.Context, userID int64, fromAccountID int64, toAccountID int64) error
	FindByIDAndUserID(ctx context.Context, accountID, userID int64) (*models.Account, error)
}

type mfaCardStore interface {
	FindAccountIDByIDAndUserID(ctx context.Context, cardID, userID int64) (int64, error)
	FindActiveAccountIDByID(ctx context.Context, cardID int64) (int64, error)
	FindByIDAndUserID(ctx context.Context, cardID, userID int64, pgpKey string) (*models.CardDetails, error)
	FindActiveByID(ctx context.Context, cardID int64, pgpKey string) (*models.CardDetails, error)
	FindActiveByNumberHMAC(ctx context.Context, numberHMAC string, pgpKey string) (*models.CardDetails, error)
}

type mfaNotificationSender interface {
	SendMFAEmail(ctx context.Context, userID int64, purpose string, code string) error
}

type MFAService struct {
	mfaRepository       mfaCodeStore
	accountRepository   mfaAccountStore
	cardRepository      mfaCardStore
	notificationService mfaNotificationSender
	attemptLimiter      *attemptLimiter
	auditRecorder       audit.Recorder
	pgpKey              string
	hmacSecret          string
	requestCooldown     *cooldownLimiter
}

func NewMFAService(
	mfaRepository mfaCodeStore,
	accountRepository mfaAccountStore,
	cardRepository mfaCardStore,
	notificationService mfaNotificationSender,
	maxFailedAttempts int,
	lockout time.Duration,
	requestCooldown time.Duration,
	auditRecorder audit.Recorder,
	pgpKey string,
	hmacSecret string,
) *MFAService {
	return &MFAService{
		mfaRepository:       mfaRepository,
		accountRepository:   accountRepository,
		cardRepository:      cardRepository,
		notificationService: notificationService,
		attemptLimiter:      newAttemptLimiter(maxFailedAttempts, lockout),
		auditRecorder:       auditRecorder,
		pgpKey:              pgpKey,
		hmacSecret:          hmacSecret,
		requestCooldown:     newCooldownLimiter(requestCooldown),
	}
}

func (s *MFAService) RequestCode(ctx context.Context, userID int64, request dto.MFARequest) error {
	purpose := normalizePurpose(request.Purpose)
	if !isAllowedPurpose(purpose) {
		return ErrInvalidMFAPurpose
	}

	// The operation hash binds an MFA code to exact operation parameters.
	// A code requested for one amount, account, card or credit cannot be reused for another operation.
	operationHash, err := s.buildOperationHash(ctx, userID, purpose, request)
	if err != nil {
		return err
	}

	requestKey := fmt.Sprintf("mfa_request:%d:%s:%s", userID, purpose, operationHash)
	if !s.requestCooldown.allow(requestKey) {
		return ErrMFARequestTooFrequent
	}

	code, err := generateMFACode()
	if err != nil {
		return fmt.Errorf("generate mfa code: %w", err)
	}

	codeHash, err := security.HashPassword(code)
	if err != nil {
		return fmt.Errorf("hash mfa code: %w", err)
	}

	// Five minutes keeps the code short-lived, but still leaves enough time to copy it from email during manual API usage.
	expiresAt := time.Now().Add(mfaCodeLifetime)

	if err := s.mfaRepository.SaveCode(ctx, userID, purpose, operationHash, codeHash, expiresAt); err != nil {
		return err
	}

	if err := s.notificationService.SendMFAEmail(ctx, userID, purpose, code); err != nil {
		return err
	}

	return nil
}

func (s *MFAService) VerifyTransferCode(ctx context.Context, userID int64, request dto.TransferRequest) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:       MFAPurposeTransfer,
		FromAccountID: request.FromAccountID,
		ToAccountID:   request.ToAccountID,
		Amount:        request.Amount,
	}

	return s.verifyCode(ctx, userID, MFAPurposeTransfer, mfaRequest, code)
}

func (s *MFAService) VerifyCardPaymentCode(
	ctx context.Context,
	userID int64,
	cardID int64,
	request dto.CardPaymentRequest,
) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose: MFAPurposeCardPayment,
		CardID:  cardID,
		Amount:  request.Amount,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCardPayment, mfaRequest, code)
}

func (s *MFAService) VerifyCardTransferCode(
	ctx context.Context,
	userID int64,
	fromCardID int64,
	request dto.CardTransferRequest,
) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:      MFAPurposeCardTransfer,
		CardID:       fromCardID,
		ToCardID:     request.RecipientCardID(),
		ToCardNumber: request.RecipientCardNumber(),
		Amount:       request.Amount,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCardTransfer, mfaRequest, code)
}

func (s *MFAService) VerifyCardRevealCode(
	ctx context.Context,
	userID int64,
	cardID int64,
	request dto.CardRevealRequest,
) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose: MFAPurposeCardReveal,
		CardID:  cardID,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCardReveal, mfaRequest, code)
}

func (s *MFAService) VerifyCreditCreateCode(
	ctx context.Context,
	userID int64,
	request dto.CreateCreditRequest,
) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:         MFAPurposeCreditCreate,
		AccountID:       request.AccountID,
		PrincipalAmount: request.PrincipalAmount,
		TermMonths:      request.TermMonths,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCreditCreate, mfaRequest, code)
}

func (s *MFAService) VerifyCreditPrepaymentCode(
	ctx context.Context,
	userID int64,
	creditID int64,
	request dto.CreditPrepaymentRequest,
) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:        MFAPurposeCreditPrepayment,
		CreditID:       creditID,
		Amount:         request.Amount,
		PrepaymentMode: request.Mode,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCreditPrepayment, mfaRequest, code)
}

func (s *MFAService) VerifyWithdrawCode(
	ctx context.Context,
	userID int64,
	accountID int64,
	request dto.WithdrawRequest,
) (*MFAVerification, error) {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return nil, ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return nil, ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:   MFAPurposeWithdraw,
		AccountID: accountID,
		Amount:    request.Amount,
	}

	return s.verifyCode(ctx, userID, MFAPurposeWithdraw, mfaRequest, code)
}

func (s *MFAService) ConsumeVerifiedCode(ctx context.Context, verification *MFAVerification) error {
	if verification == nil || verification.CodeID <= 0 {
		return ErrInvalidMFACode
	}

	if err := s.mfaRepository.MarkUsed(ctx, verification.CodeID); err != nil {
		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			s.recordMFAVerification(ctx, 0, verification.Purpose, audit.StatusFailed, "not_found_or_used")
			return ErrInvalidMFACode
		}

		return err
	}

	return nil
}

func (s *MFAService) verifyCode(
	ctx context.Context,
	userID int64,
	purpose string,
	request dto.MFARequest,
	code string,
) (*MFAVerification, error) {
	operationHash, err := s.buildOperationHash(ctx, userID, purpose, request)
	if err != nil {
		return nil, err
	}

	activeCode, err := s.mfaRepository.FindActiveCode(ctx, userID, purpose, operationHash)
	if err != nil {
		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			s.recordMFAVerification(ctx, userID, purpose, audit.StatusFailed, "not_found_or_used")
			return nil, ErrInvalidMFACode
		}

		return nil, err
	}

	attemptKey := fmt.Sprintf("mfa:%d:%s:%s", userID, purpose, operationHash)
	if s.attemptLimiter.isLocked(attemptKey) {
		s.recordMFAVerification(ctx, userID, purpose, audit.StatusBlocked, "too_many_attempts")
		return nil, ErrInvalidMFACode
	}

	if !security.CheckPassword(code, activeCode.CodeHash) {
		s.attemptLimiter.recordFailure(attemptKey)
		s.recordMFAVerification(ctx, userID, purpose, audit.StatusFailed, "invalid_code")
		return nil, ErrInvalidMFACode
	}

	s.attemptLimiter.reset(attemptKey)
	s.recordMFAVerification(ctx, userID, purpose, audit.StatusSuccess, "verified")

	return &MFAVerification{
		CodeID:        activeCode.ID,
		Purpose:       purpose,
		OperationHash: operationHash,
	}, nil
}

func (s *MFAService) recordMFAVerification(ctx context.Context, userID int64, purpose string, status string, reason string) {
	if s.auditRecorder == nil {
		return
	}

	var auditUserID *int64
	if userID > 0 {
		auditUserID = audit.Int64Ptr(userID)
	}

	s.auditRecorder.Record(ctx, audit.Event{
		UserID:       auditUserID,
		Action:       "mfa.verify." + status,
		ResourceType: "mfa_code",
		Status:       status,
		Details: map[string]any{
			"purpose": purpose,
			"reason":  reason,
		},
	})
}

func (s *MFAService) buildOperationHash(
	ctx context.Context,
	userID int64,
	purpose string,
	request dto.MFARequest,
) (string, error) {
	switch purpose {
	case MFAPurposeTransfer:
		return s.buildTransferOperationHash(ctx, userID, request)

	case MFAPurposeCardPayment:
		return s.buildCardPaymentOperationHash(ctx, userID, request)

	case MFAPurposeCardTransfer:
		return s.buildCardTransferOperationHash(ctx, userID, request)

	case MFAPurposeCardReveal:
		return s.buildCardRevealOperationHash(ctx, userID, request)

	case MFAPurposeCreditCreate:
		return s.buildCreditCreateOperationHash(ctx, userID, request)

	case MFAPurposeCreditPrepayment:
		return s.buildCreditPrepaymentOperationHash(ctx, userID, request)

	case MFAPurposeWithdraw:
		return s.buildWithdrawOperationHash(ctx, userID, request)

	default:
		return "", ErrInvalidMFAPurpose
	}
}

func (s *MFAService) buildTransferOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.FromAccountID <= 0 || request.ToAccountID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	if request.FromAccountID == request.ToAccountID {
		return "", ErrInvalidMFAOperation
	}

	amount, err := canonicalMoneyAmount(request.Amount)
	if err != nil {
		return "", err
	}

	if err := s.accountRepository.ValidateTransferAccounts(
		ctx,
		userID,
		request.FromAccountID,
		request.ToAccountID,
	); err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return "", ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrAccountBlocked) {
			return "", ErrAccountBlocked
		}

		if errors.Is(err, repositories.ErrAccountClosed) {
			return "", ErrAccountClosed
		}

		return "", err
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|from=%d|to=%d|amount=%s",
		userID,
		MFAPurposeTransfer,
		request.FromAccountID,
		request.ToAccountID,
		amount,
	)

	return hashOperation(raw), nil
}

func (s *MFAService) buildCardPaymentOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.CardID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	amount, err := canonicalMoneyAmount(request.Amount)
	if err != nil {
		return "", err
	}

	card, err := s.cardRepository.FindByIDAndUserID(ctx, request.CardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return "", ErrCardNotFound
		}

		return "", err
	}

	if card.Status == models.CardStatusClosed {
		return "", ErrCardClosed
	}

	if err := ensureCardNotExpired(card); err != nil {
		return "", err
	}

	account, err := s.accountRepository.FindByIDAndUserID(ctx, card.AccountID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return "", ErrAccountNotFound
		}

		return "", err
	}

	if account.IsBlocked {
		return "", ErrAccountBlocked
	}

	if account.IsClosed() {
		return "", ErrAccountClosed
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|card_id=%d|amount=%s",
		userID,
		MFAPurposeCardPayment,
		request.CardID,
		amount,
	)

	return hashOperation(raw), nil
}

func (s *MFAService) buildCardTransferOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.CardID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	amount, err := canonicalMoneyAmount(request.Amount)
	if err != nil {
		return "", err
	}

	fromCard, err := s.cardRepository.FindByIDAndUserID(ctx, request.CardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return "", ErrCardNotFound
		}

		return "", err
	}

	if fromCard.Status == models.CardStatusClosed {
		return "", ErrCardClosed
	}

	if err := ensureCardNotExpired(fromCard); err != nil {
		return "", err
	}

	toCard, err := s.findCardTransferRecipient(ctx, request)
	if err != nil {
		return "", err
	}

	if request.CardID == toCard.ID || fromCard.AccountID == toCard.AccountID {
		return "", ErrInvalidMFAOperation
	}

	if err := ensureCardNotExpired(toCard); err != nil {
		return "", err
	}

	if err := s.accountRepository.ValidateTransferAccounts(ctx, userID, fromCard.AccountID, toCard.AccountID); err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return "", ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrAccountBlocked) {
			return "", ErrAccountBlocked
		}

		if errors.Is(err, repositories.ErrAccountClosed) {
			return "", ErrAccountClosed
		}

		return "", err
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|from_card_id=%d|to_card_id=%d|from_account_id=%d|to_account_id=%d|amount=%s",
		userID,
		MFAPurposeCardTransfer,
		request.CardID,
		toCard.ID,
		fromCard.AccountID,
		toCard.AccountID,
		amount,
	)

	return hashOperation(raw), nil
}

func (s *MFAService) findCardTransferRecipient(ctx context.Context, request dto.MFARequest) (*models.CardDetails, error) {
	toCardNumber := strings.TrimSpace(request.RecipientCardNumber())
	if toCardNumber != "" {
		normalizedNumber := security.NormalizeCardNumber(toCardNumber)
		if !security.IsValidCardNumber(normalizedNumber) {
			return nil, ErrInvalidMFAOperation
		}

		toCard, err := s.cardRepository.FindActiveByNumberHMAC(ctx, security.ComputeHMAC(normalizedNumber, s.hmacSecret), s.pgpKey)
		if err != nil {
			if errors.Is(err, repositories.ErrCardNotFound) {
				return nil, ErrCardNotFound
			}

			return nil, err
		}

		return toCard, nil
	}

	toCardID := request.RecipientCardID()
	if toCardID <= 0 {
		return nil, ErrInvalidMFAOperation
	}

	toCard, err := s.cardRepository.FindActiveByID(ctx, toCardID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}

		return nil, err
	}

	return toCard, nil
}

func (s *MFAService) buildCardRevealOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.CardID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	card, err := s.cardRepository.FindByIDAndUserID(ctx, request.CardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return "", ErrCardNotFound
		}

		return "", err
	}

	if card.Status == models.CardStatusClosed {
		return "", ErrCardClosed
	}

	if err := ensureCardNotExpired(card); err != nil {
		return "", err
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|card_id=%d",
		userID,
		MFAPurposeCardReveal,
		request.CardID,
	)

	return hashOperation(raw), nil
}

func (s *MFAService) buildCreditCreateOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.AccountID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	if request.TermMonths <= 0 || request.TermMonths > maxCreditTermMonths {
		return "", ErrInvalidMFAOperation
	}

	principalAmount, err := canonicalMoneyAmount(request.PrincipalAmount)
	if err != nil {
		return "", err
	}

	account, err := s.accountRepository.FindByIDAndUserID(ctx, request.AccountID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return "", ErrAccountNotFound
		}

		return "", err
	}

	if account.IsBlocked {
		return "", ErrAccountBlocked
	}

	if account.IsClosed() {
		return "", ErrAccountClosed
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|account_id=%d|principal_amount=%s|term_months=%d",
		userID,
		MFAPurposeCreditCreate,
		request.AccountID,
		principalAmount,
		request.TermMonths,
	)

	return hashOperation(raw), nil
}

func (s *MFAService) buildCreditPrepaymentOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.CreditID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	amount, err := canonicalMoneyAmount(request.Amount)
	if err != nil {
		return "", err
	}

	mode := strings.TrimSpace(strings.ToLower(request.CreditPrepaymentMode()))
	if mode != repositories.CreditPrepaymentModeReducePayment &&
		mode != repositories.CreditPrepaymentModeReduceTerm &&
		mode != repositories.CreditPrepaymentModeFullClose {
		return "", ErrInvalidMFAOperation
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|credit_id=%d|amount=%s|mode=%s",
		userID,
		MFAPurposeCreditPrepayment,
		request.CreditID,
		amount,
		mode,
	)

	return hashOperation(raw), nil
}

func (s *MFAService) buildWithdrawOperationHash(
	ctx context.Context,
	userID int64,
	request dto.MFARequest,
) (string, error) {
	if request.AccountID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	amount, err := canonicalMoneyAmount(request.Amount)
	if err != nil {
		return "", err
	}

	account, err := s.accountRepository.FindByIDAndUserID(ctx, request.AccountID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return "", ErrAccountNotFound
		}

		return "", err
	}

	if account.IsBlocked {
		return "", ErrAccountBlocked
	}

	if account.IsClosed() {
		return "", ErrAccountClosed
	}

	raw := fmt.Sprintf(
		"user_id=%d|purpose=%s|account_id=%d|amount=%s",
		userID,
		MFAPurposeWithdraw,
		request.AccountID,
		amount,
	)

	return hashOperation(raw), nil
}

func hashOperation(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func canonicalMoneyAmount(amount string) (string, error) {
	normalized, err := normalizeAmount(amount)
	if err != nil {
		return "", err
	}

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return "", ErrInvalidAmount
	}

	return fmt.Sprintf("%.2f", value), nil
}

func normalizePurpose(purpose string) string {
	return strings.TrimSpace(strings.ToLower(purpose))
}

func isAllowedPurpose(purpose string) bool {
	return purpose == MFAPurposeTransfer ||
		purpose == MFAPurposeCardPayment ||
		purpose == MFAPurposeCardTransfer ||
		purpose == MFAPurposeCardReveal ||
		purpose == MFAPurposeCreditCreate ||
		purpose == MFAPurposeCreditPrepayment ||
		purpose == MFAPurposeWithdraw
}

func generateMFACode() (string, error) {
	result := ""

	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}

		result += n.String()
	}

	return result, nil
}

func isValidMFACodeFormat(code string) bool {
	if len(code) != 6 {
		return false
	}

	for _, symbol := range code {
		if symbol < '0' || symbol > '9' {
			return false
		}
	}

	return true
}
