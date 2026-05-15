package services

import (
	"context"
	"errors"
	"strings"

	"bank-service/internal/dto"
	"bank-service/internal/repositories"
)

var ErrInvalidTransfer = errors.New("invalid transfer")

type transferAccountStore interface {
	FindIDByAccountNumber(ctx context.Context, accountNumber string) (int64, error)
	Transfer(ctx context.Context, userID, fromAccountID, toAccountID int64, amount, description string) (int64, error)
	TransferWithMFA(ctx context.Context, userID, fromAccountID, toAccountID int64, amount, description string, mfaCodeID int64) (int64, error)
}

type transferMFAVerifier interface {
	VerifyTransferCode(ctx context.Context, userID int64, request dto.TransferRequest) (*MFAVerification, error)
}

type TransferService struct {
	accountRepository transferAccountStore
	mfaService        transferMFAVerifier
}

func normalizeAccountNumber(value string) string {
	return strings.Join(strings.Fields(value), "")
}

func (s *TransferService) resolveTransferRecipientAccountID(ctx context.Context, request dto.TransferRequest) (int64, error) {
	if request.ToAccountID > 0 {
		return request.ToAccountID, nil
	}

	accountNumber := normalizeAccountNumber(request.RecipientAccountNumber())
	if accountNumber == "" {
		return 0, ErrInvalidTransfer
	}

	accountID, err := s.accountRepository.FindIDByAccountNumber(ctx, accountNumber)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return 0, ErrAccountNotFound
		}

		return 0, err
	}

	return accountID, nil
}

func NewTransferService(
	accountRepository transferAccountStore,
	mfaService transferMFAVerifier,
) *TransferService {
	return &TransferService{
		accountRepository: accountRepository,
		mfaService:        mfaService,
	}
}

func (s *TransferService) Transfer(ctx context.Context, userID int64, request dto.TransferRequest) (*dto.TransferResponse, error) {
	amount, err := normalizeAmount(request.Amount)
	if err != nil {
		return nil, err
	}

	toAccountID, err := s.resolveTransferRecipientAccountID(ctx, request)
	if err != nil {
		return nil, err
	}

	if request.FromAccountID <= 0 || toAccountID <= 0 {
		return nil, ErrInvalidTransfer
	}

	if request.FromAccountID == toAccountID {
		return nil, ErrInvalidTransfer
	}

	description := strings.TrimSpace(request.Description)
	if len(description) > maxDescriptionLength {
		return nil, ErrInvalidDescription
	}

	canonicalRequest := request
	canonicalRequest.ToAccountID = toAccountID

	verification, err := s.mfaService.VerifyTransferCode(ctx, userID, canonicalRequest)
	if err != nil {
		return nil, err
	}

	transactionID, err := s.accountRepository.TransferWithMFA(
		ctx,
		userID,
		request.FromAccountID,
		toAccountID,
		amount,
		description,
		verification.CodeID,
	)
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

	return &dto.TransferResponse{
		TransactionID: transactionID,
		FromAccountID: request.FromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Status:        "completed",
	}, nil
}
