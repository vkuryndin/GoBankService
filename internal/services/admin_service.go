package services

import (
	"context"
	"errors"

	"bank-service/internal/dto"
	"bank-service/internal/repositories"
)

var (
	ErrAccountAlreadyBlocked   = errors.New("account already blocked")
	ErrAccountAlreadyUnblocked = errors.New("account already unblocked")
)

type adminStore interface {
	FindUsers(ctx context.Context) ([]repositories.AdminUser, error)
	FindActiveSessions(ctx context.Context) ([]repositories.AdminActiveSession, error)
	SetAccountBlocked(ctx context.Context, accountID int64, blocked bool) (*repositories.AdminAccountStatus, error)
}

type AdminService struct {
	adminRepository adminStore
}

func NewAdminService(adminRepository adminStore) *AdminService {
	return &AdminService{
		adminRepository: adminRepository,
	}
}

func (s *AdminService) GetUsers(ctx context.Context) ([]dto.AdminUserResponse, error) {
	users, err := s.adminRepository.FindUsers(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]dto.AdminUserResponse, 0, len(users))
	for _, user := range users {
		response = append(response, dto.AdminUserResponse{
			ID:                   user.ID,
			Email:                user.Email,
			Username:             user.Username,
			IsAdmin:              user.IsAdmin,
			AccountsCount:        user.AccountsCount,
			BlockedAccountsCount: user.BlockedAccountsCount,
			CreatedAt:            user.CreatedAt.Format(timeFormat),
		})
	}

	return response, nil
}

func (s *AdminService) GetLoggedInUsers(ctx context.Context) ([]dto.AdminActiveSessionResponse, error) {
	sessions, err := s.adminRepository.FindActiveSessions(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]dto.AdminActiveSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		response = append(response, dto.AdminActiveSessionResponse{
			SessionID: session.SessionID,
			UserID:    session.UserID,
			Email:     session.Email,
			Username:  session.Username,
			CreatedAt: session.CreatedAt.Format(timeFormat),
			ExpiresAt: session.ExpiresAt.Format(timeFormat),
		})
	}

	return response, nil
}

func (s *AdminService) BlockAccount(ctx context.Context, accountID int64) (*dto.AdminAccountStatusResponse, error) {
	return s.setAccountBlocked(ctx, accountID, true)
}

func (s *AdminService) UnblockAccount(ctx context.Context, accountID int64) (*dto.AdminAccountStatusResponse, error) {
	return s.setAccountBlocked(ctx, accountID, false)
}

func (s *AdminService) setAccountBlocked(
	ctx context.Context,
	accountID int64,
	blocked bool,
) (*dto.AdminAccountStatusResponse, error) {
	account, err := s.adminRepository.SetAccountBlocked(ctx, accountID, blocked)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		if errors.Is(err, repositories.ErrAccountStatusNotChanged) {
			if blocked {
				return nil, ErrAccountAlreadyBlocked
			}

			return nil, ErrAccountAlreadyUnblocked
		}

		return nil, err
	}

	message := "account unblocked"
	if blocked {
		message = "account blocked"
	}

	return &dto.AdminAccountStatusResponse{
		ID:            account.ID,
		UserID:        account.UserID,
		AccountNumber: account.AccountNumber,
		IsBlocked:     account.IsBlocked,
		Message:       message,
	}, nil
}

const timeFormat = "2006-01-02T15:04:05Z07:00"
