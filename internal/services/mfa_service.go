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
	MFAPurposeTransfer     = "transfer"
	MFAPurposeCardPayment  = "card_payment"
	MFAPurposeCardTransfer = "card_transfer"
	MFAPurposeCreditCreate = "credit_create"
	MFAPurposeWithdraw     = "withdraw"
)

var (
	ErrInvalidMFAPurpose     = errors.New("invalid mfa purpose")
	ErrMFACodeRequired       = errors.New("mfa code required")
	ErrInvalidMFACode        = errors.New("invalid mfa code")
	ErrInvalidMFAOperation   = errors.New("invalid mfa operation")
	ErrMFARequestTooFrequent = errors.New("mfa request too frequent")
)

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
) *MFAService {
	return &MFAService{
		mfaRepository:       mfaRepository,
		accountRepository:   accountRepository,
		cardRepository:      cardRepository,
		notificationService: notificationService,
		attemptLimiter:      newAttemptLimiter(maxFailedAttempts, lockout),
		auditRecorder:       auditRecorder,
		pgpKey:              pgpKey,
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

func (s *MFAService) VerifyTransferCode(ctx context.Context, userID int64, request dto.TransferRequest) error {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return ErrInvalidMFACode
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
) error {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return ErrInvalidMFACode
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
) error {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:  MFAPurposeCardTransfer,
		CardID:   fromCardID,
		ToCardID: request.ToCardID,
		Amount:   request.Amount,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCardTransfer, mfaRequest, code)
}

func (s *MFAService) VerifyCreditCreateCode(
	ctx context.Context,
	userID int64,
	request dto.CreateCreditRequest,
) error {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:         MFAPurposeCreditCreate,
		AccountID:       request.AccountID,
		PrincipalAmount: request.PrincipalAmount,
		TermMonths:      request.TermMonths,
	}

	return s.verifyCode(ctx, userID, MFAPurposeCreditCreate, mfaRequest, code)
}

func (s *MFAService) VerifyWithdrawCode(
	ctx context.Context,
	userID int64,
	accountID int64,
	request dto.WithdrawRequest,
) error {
	code := strings.TrimSpace(request.MFACode)
	if code == "" {
		return ErrMFACodeRequired
	}

	if !isValidMFACodeFormat(code) {
		return ErrInvalidMFACode
	}

	mfaRequest := dto.MFARequest{
		Purpose:   MFAPurposeWithdraw,
		AccountID: accountID,
		Amount:    request.Amount,
	}

	return s.verifyCode(ctx, userID, MFAPurposeWithdraw, mfaRequest, code)
}

func (s *MFAService) verifyCode(
	ctx context.Context,
	userID int64,
	purpose string,
	request dto.MFARequest,
	code string,
) error {
	operationHash, err := s.buildOperationHash(ctx, userID, purpose, request)
	if err != nil {
		return err
	}

	activeCode, err := s.mfaRepository.FindActiveCode(ctx, userID, purpose, operationHash)
	if err != nil {
		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			s.recordMFAVerification(ctx, userID, purpose, audit.StatusFailed, "not_found_or_used")
			return ErrInvalidMFACode
		}

		return err
	}

	attemptKey := fmt.Sprintf("mfa:%d:%s:%s", userID, purpose, operationHash)
	if s.attemptLimiter.isLocked(attemptKey) {
		s.recordMFAVerification(ctx, userID, purpose, audit.StatusBlocked, "too_many_attempts")
		return ErrInvalidMFACode
	}

	if !security.CheckPassword(code, activeCode.CodeHash) {
		s.attemptLimiter.recordFailure(attemptKey)
		s.recordMFAVerification(ctx, userID, purpose, audit.StatusFailed, "invalid_code")
		return ErrInvalidMFACode
	}

	if err := s.mfaRepository.MarkUsed(ctx, activeCode.ID); err != nil {
		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			s.recordMFAVerification(ctx, userID, purpose, audit.StatusFailed, "not_found_or_used")
			return ErrInvalidMFACode
		}

		return err
	}

	s.attemptLimiter.reset(attemptKey)
	s.recordMFAVerification(ctx, userID, purpose, audit.StatusSuccess, "verified")

	return nil
}

func (s *MFAService) recordMFAVerification(ctx context.Context, userID int64, purpose string, status string, reason string) {
	if s.auditRecorder == nil {
		return
	}

	s.auditRecorder.Record(ctx, audit.Event{
		UserID:       audit.Int64Ptr(userID),
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

	case MFAPurposeCreditCreate:
		return s.buildCreditCreateOperationHash(ctx, userID, request)

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
	if request.CardID <= 0 || request.ToCardID <= 0 {
		return "", ErrInvalidMFAOperation
	}

	if request.CardID == request.ToCardID {
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

	toCard, err := s.cardRepository.FindActiveByID(ctx, request.ToCardID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return "", ErrCardNotFound
		}

		return "", err
	}

	if err := ensureCardNotExpired(toCard); err != nil {
		return "", err
	}

	if fromCard.AccountID == toCard.AccountID {
		return "", ErrInvalidMFAOperation
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
		request.ToCardID,
		fromCard.AccountID,
		toCard.AccountID,
		amount,
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
		purpose == MFAPurposeCreditCreate ||
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
