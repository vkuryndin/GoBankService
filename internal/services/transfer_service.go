package services

import (
	"context"
	"errors"
	"strings"

	"bank-service/internal/dto"
	"bank-service/internal/repositories"
)

var ErrInvalidTransfer = errors.New("invalid transfer")

type TransferService struct {
	accountRepository *repositories.AccountRepository
	mfaService        *MFAService
}

func NewTransferService(
	accountRepository *repositories.AccountRepository,
	mfaService *MFAService,
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

	if err := s.mfaService.VerifyTransferCode(ctx, userID, request); err != nil {
		return nil, err
	}

	transactionID, err := s.accountRepository.Transfer(
		ctx,
		userID,
		request.FromAccountID,
		request.ToAccountID,
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

	return &dto.TransferResponse{
		TransactionID: transactionID,
		FromAccountID: request.FromAccountID,
		ToAccountID:   request.ToAccountID,
		Amount:        amount,
		Status:        "completed",
	}, nil
}
