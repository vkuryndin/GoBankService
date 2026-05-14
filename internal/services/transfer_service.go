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

	if request.FromAccountID <= 0 || request.ToAccountID <= 0 {
		return nil, ErrInvalidTransfer
	}

	if request.FromAccountID == request.ToAccountID {
		return nil, ErrInvalidTransfer
	}

	description := strings.TrimSpace(request.Description)
	if len(description) > maxDescriptionLength {
		return nil, ErrInvalidDescription
	}

	verification, err := s.mfaService.VerifyTransferCode(ctx, userID, request)
	if err != nil {
		return nil, err
	}

	transactionID, err := s.accountRepository.TransferWithMFA(
		ctx,
		userID,
		request.FromAccountID,
		request.ToAccountID,
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
		ToAccountID:   request.ToAccountID,
		Amount:        amount,
		Status:        "completed",
	}, nil
}
