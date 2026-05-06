package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrInvalidDescription  = errors.New("invalid description")
	ErrAccountBlocked      = errors.New("account is blocked")
	amountValidationRegexp = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)
)

type AccountService struct {
	accountRepository *repositories.AccountRepository
}

func NewAccountService(accountRepository *repositories.AccountRepository) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, userID int64) (*dto.AccountResponse, error) {
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

func normalizeAmount(amount string) (string, error) {
	amount = strings.TrimSpace(amount)

	if !amountValidationRegexp.MatchString(amount) {
		return "", ErrInvalidAmount
	}

	value := new(big.Rat)
	if _, ok := value.SetString(amount); !ok {
		return "", ErrInvalidAmount
	}

	if value.Sign() <= 0 {
		return "", ErrInvalidAmount
	}

	return amount, nil
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
