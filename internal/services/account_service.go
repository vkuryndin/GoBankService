package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
)

var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrInvalidDescription = errors.New("invalid description")
	ErrAccountBlocked     = errors.New("account is blocked")
)

type accountStore interface {
	Create(ctx context.Context, userID int64, accountNumber string) (*models.Account, error)
	FindByUserID(ctx context.Context, userID int64) ([]models.Account, error)
	FindByIDAndUserID(ctx context.Context, accountID, userID int64) (*models.Account, error)
	Deposit(ctx context.Context, userID, accountID int64, amount, description string) (*models.Account, int64, error)
	Withdraw(ctx context.Context, userID, accountID int64, amount, description string) (*models.Account, int64, error)
}

type withdrawMFAVerifier interface {
	VerifyWithdrawCode(ctx context.Context, userID int64, accountID int64, request dto.WithdrawRequest) error
}

type AccountService struct {
	accountRepository accountStore
	mfaService        withdrawMFAVerifier
}

func NewAccountService(accountRepository accountStore, mfaService withdrawMFAVerifier) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
		mfaService:        mfaService,
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, userID int64) (*dto.AccountResponse, error) {
	// Account numbers are generated server-side so clients cannot choose predictable or conflicting identifiers.
	accountNumber, err := generateAccountNumber()
	if err != nil {
		return nil, fmt.Errorf("generate account number: %w", err)
	}

	account, err := s.accountRepository.Create(ctx, userID, accountNumber)
	if err != nil {
		return nil, err
	}

	return toAccountResponse(account), nil
}

func (s *AccountService) GetUserAccounts(ctx context.Context, userID int64) ([]dto.AccountResponse, error) {
	accounts, err := s.accountRepository.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.AccountResponse, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, *toAccountResponse(&account))
	}

	return responses, nil
}

func (s *AccountService) GetAccount(ctx context.Context, userID, accountID int64) (*dto.AccountResponse, error) {
	account, err := s.accountRepository.FindByIDAndUserID(ctx, accountID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}

	return toAccountResponse(account), nil
}

func (s *AccountService) Deposit(ctx context.Context, userID, accountID int64, request dto.DepositRequest) (*dto.AccountResponse, error) {
	amount, err := normalizeAmount(request.Amount)
	if err != nil {
		return nil, err
	}

	account, _, err := s.accountRepository.Deposit(ctx, userID, accountID, amount, "account deposit")
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}

		return nil, err
	}

	return toAccountResponse(account), nil
}

func (s *AccountService) Withdraw(ctx context.Context, userID, accountID int64, request dto.WithdrawRequest) (*dto.AccountResponse, error) {
	amount, err := normalizeAmount(request.Amount)
	if err != nil {
		return nil, err
	}

	// Withdraw directly decreases the user's balance, so it is protected with operation-bound MFA.
	if err := s.mfaService.VerifyWithdrawCode(ctx, userID, accountID, request); err != nil {
		return nil, err
	}

	account, _, err := s.accountRepository.Withdraw(ctx, userID, accountID, amount, "account withdraw")
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

	return toAccountResponse(account), nil
}

func toAccountResponse(account *models.Account) *dto.AccountResponse {
	return &dto.AccountResponse{
		ID:            account.ID,
		AccountNumber: account.AccountNumber,
		Balance:       account.Balance,
		Currency:      account.Currency,
		CreatedAt:     account.CreatedAt.Format(time.RFC3339),
		IsBlocked:     account.IsBlocked,
	}
}

func generateAccountNumber() (string, error) {
	const prefix = "40817810"
	const randomDigitsCount = 12

	result := prefix

	for i := 0; i < randomDigitsCount; i++ {
		digit, err := randomDigit()
		if err != nil {
			return "", err
		}

		result += digit
	}

	return result, nil
}

func randomDigit() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return "", err
	}

	return n.String(), nil
}
