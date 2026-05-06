package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
	"bank-service/internal/security"
)

var (
	ErrCardNotFound      = errors.New("card not found")
	ErrInvalidCardData   = errors.New("invalid card data")
	ErrCardCreateRetries = errors.New("card create retries exceeded")
)

type CardService struct {
	cardRepository        *repositories.CardRepository
	accountRepository     *repositories.AccountRepository
	cardProcessingService *CardProcessingService
	mfaService            *MFAService
	pgpKey                string
	hmacSecret            string
}

func NewCardService(
	cardRepository *repositories.CardRepository,
	accountRepository *repositories.AccountRepository,
	cardProcessingService *CardProcessingService,
	mfaService *MFAService,
	pgpKey string,
	hmacSecret string,
) *CardService {
	return &CardService{
		cardRepository:        cardRepository,
		accountRepository:     accountRepository,
		cardProcessingService: cardProcessingService,
		mfaService:            mfaService,
		pgpKey:                pgpKey,
		hmacSecret:            hmacSecret,
	}
}

func (s *CardService) CreateCard(ctx context.Context, userID int64, request dto.CreateCardRequest) (*dto.CardResponse, error) {
	if request.AccountID <= 0 {
		return nil, ErrInvalidCardData
	}

	const maxAttempts = 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
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

	return toFullCardResponse(card), nil
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

	if len(description) > 500 {
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

		return nil, err
	}

	if err := s.verifyCardHMAC(card); err != nil {
		return nil, err
	}

	if err := s.cardProcessingService.VerifyCVV(card, request.CVV); err != nil {
		return nil, err
	}

	if err := s.mfaService.VerifyCardPaymentCode(ctx, userID, cardID, request); err != nil {
		return nil, err
	}

	_, transactionID, err := s.accountRepository.CardPayment(
		ctx,
		userID,
		card.AccountID,
		amount,
		description,
	)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}

		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
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

	card, err := s.cardRepository.Create(
		ctx,
		userID,
		accountID,
		number,
		expiry,
		cvvHash,
		numberHMAC,
		s.pgpKey,
	)
	if err != nil {
		return nil, err
	}

	card.CVV = cvv

	return card, nil
}

func (s *CardService) verifyCardHMAC(card *models.CardDetails) error {
	expectedHMAC := security.ComputeHMAC(card.Number, s.hmacSecret)
	if expectedHMAC != card.NumberHMAC {
		return fmt.Errorf("card hmac verification failed")
	}

	return nil
}

func toMaskedCardResponse(card *models.CardDetails) *dto.CardResponse {
	return &dto.CardResponse{
		ID:           card.ID,
		AccountID:    card.AccountID,
		MaskedNumber: security.MaskCardNumber(card.Number),
		NumberHMAC:   card.NumberHMAC,
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
		NumberHMAC:   card.NumberHMAC,
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
		NumberHMAC:   card.NumberHMAC,
		CreatedAt:    card.CreatedAt.Format(time.RFC3339),
	}
}
