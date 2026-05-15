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
	GetSystemStatistics(ctx context.Context) (*repositories.AdminSystemStatistics, error)
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

func (s *AdminService) GetSystemStatistics(ctx context.Context) (*dto.AdminSystemStatisticsResponse, error) {
	statistics, err := s.adminRepository.GetSystemStatistics(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.AdminSystemStatisticsResponse{
		GeneratedAt: statistics.GeneratedAt.Format(timeFormat),
		Users: dto.AdminUsersStatisticsResponse{
			Total:          statistics.Users.Total,
			Admins:         statistics.Users.Admins,
			RegularUsers:   statistics.Users.RegularUsers,
			NewLast24h:     statistics.Users.NewLast24h,
			ActiveSessions: statistics.Users.ActiveSessions,
		},
		Accounts: dto.AdminAccountsStatisticsResponse{
			Total:        statistics.Accounts.Total,
			Active:       statistics.Accounts.Active,
			Closed:       statistics.Accounts.Closed,
			Blocked:      statistics.Accounts.Blocked,
			TotalBalance: statistics.Accounts.TotalBalance,
			Currency:     statistics.Accounts.Currency,
		},
		Cards: dto.AdminCardsStatisticsResponse{
			Total:  statistics.Cards.Total,
			Active: statistics.Cards.Active,
			Closed: statistics.Cards.Closed,
		},
		Credits: dto.AdminCreditsStatisticsResponse{
			Total:                 statistics.Credits.Total,
			Active:                statistics.Credits.Active,
			Closed:                statistics.Credits.Closed,
			Overdue:               statistics.Credits.Overdue,
			ActivePrincipalAmount: statistics.Credits.ActivePrincipalAmount,
			ActiveMonthlyPayment:  statistics.Credits.ActiveMonthlyPayment,
			Currency:              statistics.Credits.Currency,
		},
		Transactions: dto.AdminTransactionsStatisticsResponse{
			Total:              statistics.Transactions.Total,
			Completed:          statistics.Transactions.Completed,
			Failed:             statistics.Transactions.Failed,
			Last24h:            statistics.Transactions.Last24h,
			CompletedAmount:    statistics.Transactions.CompletedAmount,
			CompletedThisMonth: statistics.Transactions.CompletedThisMonth,
			Currency:           statistics.Transactions.Currency,
			ByType:             toAdminTransactionTypeResponses(statistics.Transactions.ByType),
			Recent:             toAdminRecentTransactionResponses(statistics.Transactions.Recent),
		},
		Audit: dto.AdminAuditStatisticsResponse{
			Total:   statistics.Audit.Total,
			Success: statistics.Audit.Success,
			Failed:  statistics.Audit.Failed,
			Blocked: statistics.Audit.Blocked,
			Recent:  toAdminRecentAuditResponses(statistics.Audit.Recent),
		},
	}, nil
}

func toAdminTransactionTypeResponses(items []repositories.AdminTransactionTypeStatistics) []dto.AdminTransactionTypeResponse {
	response := make([]dto.AdminTransactionTypeResponse, 0, len(items))
	for _, item := range items {
		response = append(response, dto.AdminTransactionTypeResponse{
			Type:        item.Type,
			Count:       item.Count,
			TotalAmount: item.TotalAmount,
		})
	}
	return response
}

func toAdminRecentTransactionResponses(items []repositories.AdminRecentTransaction) []dto.AdminRecentTransactionResponse {
	response := make([]dto.AdminRecentTransactionResponse, 0, len(items))
	for _, item := range items {
		transaction := dto.AdminRecentTransactionResponse{
			ID:        item.ID,
			UserID:    item.UserID,
			Type:      item.Type,
			Status:    item.Status,
			Amount:    item.Amount,
			Currency:  item.Currency,
			CreatedAt: item.CreatedAt.Format(timeFormat),
		}
		if item.Description.Valid {
			transaction.Description = item.Description.String
		}
		response = append(response, transaction)
	}
	return response
}

func toAdminRecentAuditResponses(items []repositories.AdminRecentAudit) []dto.AdminRecentAuditResponse {
	response := make([]dto.AdminRecentAuditResponse, 0, len(items))
	for _, item := range items {
		auditEvent := dto.AdminRecentAuditResponse{
			ID:        item.ID,
			Action:    item.Action,
			Status:    item.Status,
			CreatedAt: item.CreatedAt.Format(timeFormat),
		}
		if item.UserID.Valid {
			userID := item.UserID.Int64
			auditEvent.UserID = &userID
		}
		if item.ResourceType.Valid {
			auditEvent.ResourceType = item.ResourceType.String
		}
		if item.ResourceID.Valid {
			resourceID := item.ResourceID.Int64
			auditEvent.ResourceID = &resourceID
		}
		response = append(response, auditEvent)
	}
	return response
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
