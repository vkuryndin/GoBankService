package services

import (
	"context"
	"crypto/hmac"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
	"bank-service/internal/security"
)

var (
	ErrCardNotFound        = errors.New("card not found")
	ErrInvalidCardData     = errors.New("invalid card data")
	ErrCardCreateRetries   = errors.New("card create retries exceeded")
	ErrCVVAttemptsBlocked  = errors.New("cvv attempts blocked")
	ErrCardClosed          = errors.New("card is closed")
	ErrCardExpired         = errors.New("card is expired")
	ErrCardAlreadyClosed   = errors.New("card already closed")
	ErrInvalidCardTransfer = errors.New("invalid card transfer")
)

type cardStore interface {
	Create(ctx context.Context, userID int64, accountID int64, number string, expiry string, cvvHash string, numberHMAC string, pgpKey string) (*models.CardDetails, error)
	FindByUserID(ctx context.Context, userID int64, pgpKey string) ([]models.CardDetails, error)
	FindByIDAndUserID(ctx context.Context, cardID, userID int64, pgpKey string) (*models.CardDetails, error)
	FindActiveByID(ctx context.Context, cardID int64, pgpKey string) (*models.CardDetails, error)
	FindActiveByNumberHMAC(ctx context.Context, numberHMAC string, pgpKey string) (*models.CardDetails, error)
	FindActiveAccountIDByID(ctx context.Context, cardID int64) (int64, error)
	Close(ctx context.Context, userID int64, cardID int64) (*models.CardDetails, error)
}

type cardPaymentAccountStore interface {
	CardPayment(ctx context.Context, userID, accountID, cardID int64, amount, description string) (*models.Account, int64, error)
	CardPaymentWithMFA(ctx context.Context, userID, accountID, cardID int64, amount, description string, mfaCodeID int64) (*models.Account, int64, error)
	Transfer(ctx context.Context, userID, fromAccountID, toAccountID int64, amount, description string) (int64, error)
	TransferWithMFA(ctx context.Context, userID, fromAccountID, toAccountID int64, amount, description string, mfaCodeID int64) (int64, error)
	TransferByCardWithMFA(ctx context.Context, userID, fromAccountID, toAccountID, fromCardID, toCardID int64, amount, description string, mfaCodeID int64) (int64, error)
}

type cardProcessor interface {
	GenerateCVVAndHash() (string, string, error)
	VerifyCVV(card *models.CardDetails, cvv string) error
}

type cardPaymentMFAVerifier interface {
	VerifyCardPaymentCode(ctx context.Context, userID int64, cardID int64, request dto.CardPaymentRequest) (*MFAVerification, error)
	VerifyCardTransferCode(ctx context.Context, userID int64, fromCardID int64, request dto.CardTransferRequest) (*MFAVerification, error)
	VerifyCardRevealCode(ctx context.Context, userID int64, cardID int64, request dto.CardRevealRequest) (*MFAVerification, error)
	ConsumeVerifiedCode(ctx context.Context, verification *MFAVerification) error
}

type CardService struct {
	cardRepository        cardStore
	accountRepository     cardPaymentAccountStore
	cardProcessingService cardProcessor
	mfaService            cardPaymentMFAVerifier
	pgpKey                string
	hmacSecret            string
	cvvAttemptLimiter     *attemptLimiter
}

func NewCardService(
	cardRepository cardStore,
	accountRepository cardPaymentAccountStore,
	cardProcessingService cardProcessor,
	mfaService cardPaymentMFAVerifier,
	pgpKey string,
	hmacSecret string,
	maxFailedCVVAttempts int,
	cvvLockout time.Duration,
) *CardService {
	return &CardService{
		cardRepository:        cardRepository,
		accountRepository:     accountRepository,
		cardProcessingService: cardProcessingService,
		mfaService:            mfaService,
		pgpKey:                pgpKey,
		hmacSecret:            hmacSecret,
		cvvAttemptLimiter:     newAttemptLimiter(maxFailedCVVAttempts, cvvLockout),
	}
}

func (s *CardService) CreateCard(ctx context.Context, userID int64, request dto.CreateCardRequest) (*dto.CardResponse, error) {
	if request.AccountID <= 0 {
		return nil, ErrInvalidCardData
	}

	for attempt := 0; attempt < maxCardCreationAttempts; attempt++ {
		card, err := s.createCardOnce(ctx, userID, request.AccountID)
		if err != nil {
			if errors.Is(err, repositories.ErrCardAlreadyExists) {
				continue
			}

			if errors.Is(err, repositories.ErrAccountNotFound) {
				return nil, ErrAccountNotFound
			}

			return nil, err
		}

		return toIssuedCardResponse(card), nil
	}

	return nil, ErrCardCreateRetries
}

func (s *CardService) GetUserCards(ctx context.Context, userID int64) ([]dto.CardResponse, error) {
	cards, err := s.cardRepository.FindByUserID(ctx, userID, s.pgpKey)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.CardResponse, 0, len(cards))
	for _, card := range cards {
		responses = append(responses, *toMaskedCardResponse(&card))
	}

	return responses, nil
}

func (s *CardService) GetCard(ctx context.Context, userID, cardID int64) (*dto.CardResponse, error) {
	card, err := s.cardRepository.FindByIDAndUserID(ctx, cardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}

		return nil, err
	}

	if err := s.verifyCardHMAC(card); err != nil {
		return nil, err
	}

	return toMaskedCardResponse(card), nil
}

func (s *CardService) RevealCard(
	ctx context.Context,
	userID int64,
	cardID int64,
	request dto.CardRevealRequest,
) (*dto.CardResponse, error) {
	if cardID <= 0 {
		return nil, ErrInvalidCardData
	}

	card, err := s.cardRepository.FindByIDAndUserID(ctx, cardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}

		return nil, err
	}

	if card.Status == models.CardStatusClosed {
		return nil, ErrCardClosed
	}

	if err := ensureCardNotExpired(card); err != nil {
		return nil, err
	}

	if err := s.verifyCardHMAC(card); err != nil {
		return nil, err
	}

	verification, err := s.mfaService.VerifyCardRevealCode(ctx, userID, cardID, request)
	if err != nil {
		return nil, err
	}

	if err := s.mfaService.ConsumeVerifiedCode(ctx, verification); err != nil {
		return nil, err
	}

	return toFullCardResponse(card), nil
}

func (s *CardService) CloseCard(ctx context.Context, userID int64, cardID int64) (*dto.CloseCardResponse, error) {
	if cardID <= 0 {
		return nil, ErrInvalidCardData
	}

	// Cards are soft-closed instead of deleted to preserve transaction history and auditability.
	card, err := s.cardRepository.Close(ctx, userID, cardID)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}
		if errors.Is(err, repositories.ErrCardAlreadyClosed) {
			return nil, ErrCardAlreadyClosed
		}

		return nil, err
	}

	return toCloseCardResponse(card), nil
}

func (s *CardService) PayByCard(
	ctx context.Context,
	userID int64,
	cardID int64,
	request dto.CardPaymentRequest,
) (*dto.CardPaymentResponse, error) {
	amount, err := normalizeAmount(request.Amount)
	if err != nil {
		return nil, err
	}

	description := strings.TrimSpace(request.Description)
	if description == "" {
		description = "card payment"
	}

	if len(description) > maxDescriptionLength {
		return nil, ErrInvalidDescription
	}

	card, err := s.cardRepository.FindByIDAndUserID(ctx, cardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}
		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}

		if errors.Is(err, repositories.ErrAccountClosed) {
			return nil, ErrAccountClosed
		}

		return nil, err
	}

	if card.Status == models.CardStatusClosed {
		return nil, ErrCardClosed
	}

	if err := ensureCardNotExpired(card); err != nil {
		return nil, err
	}

	if err := s.verifyCardHMAC(card); err != nil {
		return nil, err
	}

	cvvAttemptKey := fmt.Sprintf("cvv:%d:%d", userID, cardID)
	if s.cvvAttemptLimiter.isLocked(cvvAttemptKey) {
		return nil, ErrCVVAttemptsBlocked
	}

	if err := s.cardProcessingService.VerifyCVV(card, request.CVV); err != nil {
		s.cvvAttemptLimiter.recordFailure(cvvAttemptKey)
		if s.cvvAttemptLimiter.isLocked(cvvAttemptKey) {
			return nil, ErrCVVAttemptsBlocked
		}

		return nil, err
	}

	s.cvvAttemptLimiter.reset(cvvAttemptKey)

	verification, err := s.mfaService.VerifyCardPaymentCode(ctx, userID, cardID, request)
	if err != nil {
		return nil, err
	}

	_, transactionID, err := s.accountRepository.CardPaymentWithMFA(ctx, userID, card.AccountID, cardID, amount, description, verification.CodeID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}

		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			return nil, ErrInvalidMFACode
		}

		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}

		if errors.Is(err, repositories.ErrAccountClosed) {
			return nil, ErrAccountClosed
		}

		return nil, err
	}

	return &dto.CardPaymentResponse{
		TransactionID: transactionID,
		CardID:        cardID,
		AccountID:     card.AccountID,
		Amount:        amount,
		Status:        "completed",
	}, nil
}

func (s *CardService) TransferByCard(
	ctx context.Context,
	userID int64,
	fromCardID int64,
	request dto.CardTransferRequest,
) (*dto.CardTransferResponse, error) {
	amount, err := normalizeAmount(request.Amount)
	if err != nil {
		return nil, err
	}

	description := strings.TrimSpace(request.Description)
	if description == "" {
		description = "card to card transfer"
	}

	if len(description) > maxDescriptionLength {
		return nil, ErrInvalidDescription
	}

	fromCard, err := s.cardRepository.FindByIDAndUserID(ctx, fromCardID, userID, s.pgpKey)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}

		return nil, err
	}

	if fromCard.Status == models.CardStatusClosed {
		return nil, ErrCardClosed
	}

	if err := ensureCardNotExpired(fromCard); err != nil {
		return nil, err
	}

	if err := s.verifyCardHMAC(fromCard); err != nil {
		return nil, err
	}

	cvvAttemptKey := fmt.Sprintf("cvv:%d:%d", userID, fromCardID)
	if s.cvvAttemptLimiter.isLocked(cvvAttemptKey) {
		return nil, ErrCVVAttemptsBlocked
	}

	if err := s.cardProcessingService.VerifyCVV(fromCard, request.CVV); err != nil {
		s.cvvAttemptLimiter.recordFailure(cvvAttemptKey)
		if s.cvvAttemptLimiter.isLocked(cvvAttemptKey) {
			return nil, ErrCVVAttemptsBlocked
		}

		return nil, err
	}

	s.cvvAttemptLimiter.reset(cvvAttemptKey)

	toCard, err := s.findTransferRecipientCard(ctx, request)
	if err != nil {
		return nil, err
	}

	if toCard.ID == fromCardID || fromCard.AccountID == toCard.AccountID {
		return nil, ErrInvalidCardTransfer
	}

	if err := ensureCardNotExpired(toCard); err != nil {
		return nil, err
	}

	verification, err := s.mfaService.VerifyCardTransferCode(ctx, userID, fromCardID, request)
	if err != nil {
		return nil, err
	}

	transactionID, err := s.accountRepository.TransferByCardWithMFA(ctx, userID, fromCard.AccountID, toCard.AccountID, fromCardID, toCard.ID, amount, description, verification.CodeID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}

		if errors.Is(err, repositories.ErrMFACodeNotFound) {
			return nil, ErrInvalidMFACode
		}

		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}

		if errors.Is(err, repositories.ErrAccountClosed) {
			return nil, ErrAccountClosed
		}

		return nil, err
	}

	return &dto.CardTransferResponse{
		TransactionID: transactionID,
		FromCardID:    fromCardID,
		ToCardID:      toCard.ID,
		FromAccountID: fromCard.AccountID,
		ToAccountID:   toCard.AccountID,
		Amount:        amount,
		Status:        "completed",
	}, nil
}

func (s *CardService) findTransferRecipientCard(ctx context.Context, request dto.CardTransferRequest) (*models.CardDetails, error) {
	toCardNumber := strings.TrimSpace(request.RecipientCardNumber())
	if toCardNumber != "" {
		normalizedNumber := security.NormalizeCardNumber(toCardNumber)
		if !security.IsValidCardNumber(normalizedNumber) {
			return nil, ErrInvalidCardTransfer
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
		return nil, ErrInvalidCardTransfer
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

func (s *CardService) createCardOnce(ctx context.Context, userID, accountID int64) (*models.CardDetails, error) {
	number, err := security.GenerateCardNumber()
	if err != nil {
		return nil, fmt.Errorf("generate card number: %w", err)
	}

	expiry := security.GenerateExpiry()

	cvv, cvvHash, err := s.cardProcessingService.GenerateCVVAndHash()
	if err != nil {
		return nil, err
	}

	numberHMAC := security.ComputeHMAC(number, s.hmacSecret)

	card, err := s.cardRepository.Create(ctx, userID, accountID, number, expiry, cvvHash, numberHMAC, s.pgpKey)
	if err != nil {
		return nil, err
	}

	card.CVV = cvv

	return card, nil
}

func (s *CardService) verifyCardHMAC(card *models.CardDetails) error {
	// PGP protects card confidentiality; HMAC gives us an integrity check without storing the plain card number.
	expectedHMAC := security.ComputeHMAC(card.Number, s.hmacSecret)
	if !hmac.Equal([]byte(expectedHMAC), []byte(card.NumberHMAC)) {
		return fmt.Errorf("card hmac verification failed")
	}

	return nil
}

func toMaskedCardResponse(card *models.CardDetails) *dto.CardResponse {
	return &dto.CardResponse{
		ID:           card.ID,
		AccountID:    card.AccountID,
		MaskedNumber: security.MaskCardNumber(card.Number),
		Expiry:       card.Expiry,
		Status:       normalizeCardStatus(card.Status),
		ClosedAt:     formatNullableTime(card.ClosedAt),
		CreatedAt:    card.CreatedAt.Format(time.RFC3339),
	}
}

func toFullCardResponse(card *models.CardDetails) *dto.CardResponse {
	return &dto.CardResponse{
		ID:           card.ID,
		AccountID:    card.AccountID,
		Number:       card.Number,
		MaskedNumber: security.MaskCardNumber(card.Number),
		Expiry:       card.Expiry,
		Status:       normalizeCardStatus(card.Status),
		ClosedAt:     formatNullableTime(card.ClosedAt),
		CreatedAt:    card.CreatedAt.Format(time.RFC3339),
	}
}

func toIssuedCardResponse(card *models.CardDetails) *dto.CardResponse {
	return &dto.CardResponse{
		ID:           card.ID,
		AccountID:    card.AccountID,
		Number:       card.Number,
		MaskedNumber: security.MaskCardNumber(card.Number),
		Expiry:       card.Expiry,
		CVV:          card.CVV,
		Status:       normalizeCardStatus(card.Status),
		ClosedAt:     formatNullableTime(card.ClosedAt),
		CreatedAt:    card.CreatedAt.Format(time.RFC3339),
	}
}

func toCloseCardResponse(card *models.CardDetails) *dto.CloseCardResponse {
	return &dto.CloseCardResponse{
		ID:        card.ID,
		AccountID: card.AccountID,
		Status:    normalizeCardStatus(card.Status),
		ClosedAt:  formatNullableTime(card.ClosedAt),
	}
}

func ensureCardNotExpired(card *models.CardDetails) error {
	expiresAt, err := parseCardExpiry(card.Expiry)
	if err != nil {
		return ErrInvalidCardData
	}

	if !time.Now().UTC().Before(expiresAt) {
		return ErrCardExpired
	}

	return nil
}

func parseCardExpiry(expiry string) (time.Time, error) {
	expiry = strings.TrimSpace(expiry)
	if expiry == "" {
		return time.Time{}, ErrInvalidCardData
	}

	if parsed, err := time.Parse("2006-01", expiry); err == nil {
		return time.Date(parsed.Year(), parsed.Month()+1, 1, 0, 0, 0, 0, time.UTC), nil
	}

	if parsed, err := time.Parse("2006-01-02", expiry); err == nil {
		return time.Date(parsed.Year(), parsed.Month()+1, 1, 0, 0, 0, 0, time.UTC), nil
	}

	separator := "/"
	if strings.Contains(expiry, "-") {
		separator = "-"
	}

	parts := strings.Split(expiry, separator)
	if len(parts) != 2 {
		return time.Time{}, ErrInvalidCardData
	}

	month, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || month < 1 || month > 12 {
		return time.Time{}, ErrInvalidCardData
	}

	year, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}, ErrInvalidCardData
	}

	if year < 100 {
		year += 2000
	}

	if year < 2000 {
		return time.Time{}, ErrInvalidCardData
	}

	return time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC), nil
}

func normalizeCardStatus(status string) string {
	if status == "" {
		return models.CardStatusActive
	}
	return status
}

func formatNullableTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}
