package services

import (
	"context"
	"database/sql"
	"errors"

	"bank-service/internal/dto"
	"bank-service/internal/repositories"
)

var (
	ErrInvalidPredictionDays  = errors.New("invalid prediction days")
	ErrInvalidStatisticsLimit = errors.New("invalid statistics limit")
)

type analyticsStore interface {
	GetMonthlyAnalytics(ctx context.Context, userID int64) (*repositories.MonthlyAnalytics, error)
	PredictBalance(ctx context.Context, userID int64, accountID int64, days int) (*repositories.BalancePrediction, error)
	GetAccountOperationStatistics(ctx context.Context, userID int64, accountID int64, limit int) (*repositories.OperationStatistics, error)
	GetCardOperationStatistics(ctx context.Context, userID int64, cardID int64, limit int) (*repositories.OperationStatistics, error)
}

type AnalyticsService struct {
	analyticsRepository analyticsStore
}

func NewAnalyticsService(analyticsRepository analyticsStore) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepository: analyticsRepository,
	}
}

func (s *AnalyticsService) GetAnalytics(ctx context.Context, userID int64) (*dto.AnalyticsResponse, error) {
	analytics, err := s.analyticsRepository.GetMonthlyAnalytics(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.AnalyticsResponse{
		IncomeThisMonth:  analytics.Income,
		ExpenseThisMonth: analytics.Expense,
		CreditLoad:       analytics.CreditLoad,
	}, nil
}

func (s *AnalyticsService) PredictBalance(
	ctx context.Context,
	userID int64,
	accountID int64,
	days int,
) (*dto.PredictBalanceResponse, error) {
	if days <= 0 || days > maxPredictionDays {
		return nil, ErrInvalidPredictionDays
	}

	prediction, err := s.analyticsRepository.PredictBalance(ctx, userID, accountID, days)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}

	return &dto.PredictBalanceResponse{
		AccountID:               prediction.AccountID,
		Days:                    prediction.Days,
		CurrentBalance:          prediction.CurrentBalance,
		ExpectedIncome:          prediction.ExpectedIncome,
		ExpectedExpense:         prediction.ExpectedExpense,
		ScheduledCreditPayments: prediction.ScheduledCreditPayments,
		PredictedBalance:        prediction.PredictedBalance,
		AverageDailyIncome:      prediction.AverageDailyIncome,
		AverageDailyExpense:     prediction.AverageDailyExpense,
		StatisticsPeriodDays:    prediction.StatisticsPeriodDays,
	}, nil
}

func (s *AnalyticsService) GetAccountOperationStatistics(
	ctx context.Context,
	userID int64,
	accountID int64,
	limit int,
) (*dto.OperationStatisticsResponse, error) {
	if limit <= 0 || limit > maxOperationStatisticsLimit {
		return nil, ErrInvalidStatisticsLimit
	}

	stats, err := s.analyticsRepository.GetAccountOperationStatistics(ctx, userID, accountID, limit)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}

		return nil, err
	}

	return toOperationStatisticsResponse(stats), nil
}

func (s *AnalyticsService) GetCardOperationStatistics(
	ctx context.Context,
	userID int64,
	cardID int64,
	limit int,
) (*dto.OperationStatisticsResponse, error) {
	if limit <= 0 || limit > maxOperationStatisticsLimit {
		return nil, ErrInvalidStatisticsLimit
	}

	stats, err := s.analyticsRepository.GetCardOperationStatistics(ctx, userID, cardID, limit)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return nil, ErrCardNotFound
		}

		return nil, err
	}

	return toOperationStatisticsResponse(stats), nil
}

func toOperationStatisticsResponse(stats *repositories.OperationStatistics) *dto.OperationStatisticsResponse {
	response := &dto.OperationStatisticsResponse{
		EntityType:     stats.EntityType,
		EntityID:       stats.EntityID,
		Currency:       stats.Currency,
		OperationCount: stats.OperationCount,
		IncomeCount:    stats.IncomeCount,
		ExpenseCount:   stats.ExpenseCount,
		TotalIncome:    stats.TotalIncome,
		TotalExpense:   stats.TotalExpense,
		NetAmount:      stats.NetAmount,
		ByType:         make([]dto.OperationTypeResponse, 0, len(stats.ByType)),
		ByStatus:       make([]dto.OperationStatusResponse, 0, len(stats.ByStatus)),
		Operations:     make([]dto.OperationHistoryResponse, 0, len(stats.Operations)),
	}

	for _, item := range stats.ByType {
		response.ByType = append(response.ByType, dto.OperationTypeResponse{
			Type:         item.Type,
			Count:        item.Count,
			TotalIncome:  item.TotalIncome,
			TotalExpense: item.TotalExpense,
			NetAmount:    item.NetAmount,
		})
	}

	for _, item := range stats.ByStatus {
		response.ByStatus = append(response.ByStatus, dto.OperationStatusResponse{
			Status:      item.Status,
			Count:       item.Count,
			TotalAmount: item.TotalAmount,
		})
	}

	for _, item := range stats.Operations {
		response.Operations = append(response.Operations, dto.OperationHistoryResponse{
			ID:            item.ID,
			Direction:     item.Direction,
			Type:          item.Type,
			Status:        item.Status,
			Amount:        item.Amount,
			Currency:      item.Currency,
			Description:   item.Description,
			FromAccountID: nullableInt64Ptr(item.FromAccountID),
			ToAccountID:   nullableInt64Ptr(item.ToAccountID),
			FromCardID:    nullableInt64Ptr(item.FromCardID),
			ToCardID:      nullableInt64Ptr(item.ToCardID),
			CreatedAt:     item.CreatedAt,
		})
	}

	return response
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	return &value.Int64
}
