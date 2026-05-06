package services

import (
	"context"
	"errors"

	"bank-service/internal/dto"
	"bank-service/internal/repositories"
)

var ErrInvalidPredictionDays = errors.New("invalid prediction days")

type analyticsStore interface {
	GetMonthlyAnalytics(ctx context.Context, userID int64) (*repositories.MonthlyAnalytics, error)
	PredictBalance(ctx context.Context, userID int64, accountID int64, days int) (*repositories.BalancePrediction, error)
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
