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
	ErrAccountNotFound          = errors.New("account not found")
	ErrInvalidAmount            = errors.New("invalid amount")
	ErrInsufficientFunds        = errors.New("insufficient funds")
	ErrInvalidDescription       = errors.New("invalid description")
	ErrAccountBlocked           = errors.New("account is blocked")
	ErrAccountClosed            = errors.New("account is closed")
	ErrAccountAlreadyClosed     = errors.New("account already closed")
	ErrAccountBalanceMustBeZero = errors.New("account balance must be zero")
	ErrAccountHasActiveCredit   = errors.New("account has active credit")
)

type accountStore interface {
	Create(ctx context.Context, userID int64, accountNumber string) (*models.Account, error)
	FindByUserID(ctx context.Context, userID int64) ([]models.Account, error)
	FindByIDAndUserID(ctx context.Context, accountID, userID int64) (*models.Account, error)
	Deposit(ctx context.Context, userID, accountID int64, amount, description string) (*models.Account, int64, error)
	Withdraw(ctx context.Context, userID, accountID int64, amount, description string) (*models.Account, int64, error)
	WithdrawWithMFA(ctx context.Context, userID, accountID int64, amount, description string, mfaCodeID int64) (*models.Account, int64, error)
	Close(ctx context.Context, userID int64, accountID int64) (*models.Account, error)
}

type withdrawMFAVerifier interface {
	VerifyWithdrawCode(ctx context.Context, userID int64, accountID int64, request dto.WithdrawRequest) (*MFAVerification, error)
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
		if errors.Is(err, repositories.ErrAccountClosed) {
			return nil, ErrAccountClosed
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
	verification, err := s.mfaService.VerifyWithdrawCode(ctx, userID, accountID, request)
	if err != nil {
		return nil, err
	}

	account, _, err := s.accountRepository.WithdrawWithMFA(ctx, userID, accountID, amount, "account withdraw", verification.CodeID)
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

	return toAccountResponse(account), nil
}

func (s *AccountService) CloseAccount(ctx context.Context, userID int64, accountID int64) (*dto.CloseAccountResponse, error) {
	if accountID <= 0 {
		return nil, ErrAccountNotFound
	}

	// Accounts are soft-closed instead of deleted so transactions, credits and audit records remain intact.
	account, err := s.accountRepository.Close(ctx, userID, accountID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}
		if errors.Is(err, repositories.ErrAccountBlocked) {
			return nil, ErrAccountBlocked
		}
		if errors.Is(err, repositories.ErrAccountAlreadyClosed) {
			return nil, ErrAccountAlreadyClosed
		}
		if errors.Is(err, repositories.ErrAccountBalanceMustBeZero) {
			return nil, ErrAccountBalanceMustBeZero
		}
		if errors.Is(err, repositories.ErrAccountHasActiveCredit) {
			return nil, ErrAccountHasActiveCredit
		}

		return nil, err
	}

	return toCloseAccountResponse(account), nil
}

func toAccountResponse(account *models.Account) *dto.AccountResponse {
	response := &dto.AccountResponse{
		ID:            account.ID,
		AccountNumber: account.AccountNumber,
		Balance:       account.Balance,
		Currency:      account.Currency,
		CreatedAt:     account.CreatedAt.Format(time.RFC3339),
		IsBlocked:     account.IsBlocked,
		Status:        account.Status,
	}
	if account.ClosedAt.Valid {
		response.ClosedAt = account.ClosedAt.Time.Format(time.RFC3339)
	}

	return response
}

func toCloseAccountResponse(account *models.Account) *dto.CloseAccountResponse {
	closedAt := ""
	if account.ClosedAt.Valid {
		closedAt = account.ClosedAt.Time.Format(time.RFC3339)
	}

	return &dto.CloseAccountResponse{
		ID:            account.ID,
		AccountNumber: account.AccountNumber,
		Status:        account.Status,
		ClosedAt:      closedAt,
		Message:       "account closed",
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
