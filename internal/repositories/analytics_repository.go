package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type MonthlyAnalytics struct {
	Income     string
	Expense    string
	CreditLoad string
}

type BalancePrediction struct {
	AccountID               int64
	Days                    int
	CurrentBalance          string
	ExpectedIncome          string
	ExpectedExpense         string
	ScheduledCreditPayments string
	PredictedBalance        string
	AverageDailyIncome      string
	AverageDailyExpense     string
	StatisticsPeriodDays    int
}

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{
		db: db,
	}
}

func (r *AnalyticsRepository) GetMonthlyAnalytics(ctx context.Context, userID int64) (*MonthlyAnalytics, error) {
	income, err := r.getMonthlyIncome(ctx, userID)
	if err != nil {
		return nil, err
	}

	expense, err := r.getMonthlyExpense(ctx, userID)
	if err != nil {
		return nil, err
	}

	creditLoad, err := r.getCreditLoad(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &MonthlyAnalytics{
		Income:     income,
		Expense:    expense,
		CreditLoad: creditLoad,
	}, nil
}

func (r *AnalyticsRepository) PredictBalance(
	ctx context.Context,
	userID int64,
	accountID int64,
	days int,
) (*BalancePrediction, error) {
	const statisticsPeriodDays = 30

	query := `
		WITH account_data AS (
			SELECT id, balance
			FROM accounts
			WHERE id = $1 AND user_id = $2
		),
		income_data AS (
			SELECT COALESCE(SUM(t.amount), 0) AS income
			FROM transactions t
			LEFT JOIN accounts to_acc ON to_acc.id = t.to_account_id
			LEFT JOIN accounts from_acc ON from_acc.id = t.from_account_id
			WHERE t.status = 'completed'
			  AND t.created_at >= NOW() - INTERVAL '30 days'
			  AND (
				(t.type IN ('deposit', 'credit_issue') AND t.to_account_id = $1 AND t.user_id = $2)
				OR (
					t.type = 'transfer'
					AND to_acc.id = $1
					AND (from_acc.user_id IS NULL OR from_acc.user_id <> $2)
				)
			  )
		),
		expense_data AS (
			SELECT COALESCE(SUM(t.amount), 0) AS expense
			FROM transactions t
			LEFT JOIN accounts to_acc ON to_acc.id = t.to_account_id
			LEFT JOIN accounts from_acc ON from_acc.id = t.from_account_id
			WHERE t.status = 'completed'
			  AND t.created_at >= NOW() - INTERVAL '30 days'
			  AND (
				(t.type IN ('withdraw', 'card_payment', 'credit_payment', 'penalty') AND t.from_account_id = $1)
				OR (
					t.type = 'transfer'
					AND from_acc.id = $1
					AND (to_acc.user_id IS NULL OR to_acc.user_id <> $2)
				)
			  )
		),
		scheduled_payments AS (
			SELECT COALESCE(SUM(ps.amount + ps.penalty_amount), 0) AS scheduled_amount
			FROM payment_schedules ps
			INNER JOIN credits c ON c.id = ps.credit_id
			WHERE c.account_id = $1
			  AND c.user_id = $2
			  AND ps.status IN ('pending', 'overdue')
			  AND ps.payment_date <= CURRENT_DATE + ($3::int * INTERVAL '1 day')
		)
		SELECT
			ad.balance::text,
			ROUND((id.income / $4::numeric) * $3::numeric, 2)::text AS expected_income,
			ROUND((ed.expense / $4::numeric) * $3::numeric, 2)::text AS expected_expense,
			sp.scheduled_amount::text,
			ROUND((
				ad.balance
				+ ((id.income / $4::numeric) * $3::numeric)
				- ((ed.expense / $4::numeric) * $3::numeric)
				- sp.scheduled_amount
			), 2)::text AS predicted_balance,
			ROUND(id.income / $4::numeric, 2)::text AS average_daily_income,
			ROUND(ed.expense / $4::numeric, 2)::text AS average_daily_expense
		FROM account_data ad
		CROSS JOIN income_data id
		CROSS JOIN expense_data ed
		CROSS JOIN scheduled_payments sp
	`

	prediction := &BalancePrediction{
		AccountID:            accountID,
		Days:                 days,
		StatisticsPeriodDays: statisticsPeriodDays,
	}

	err := r.db.QueryRowContext(ctx, query, accountID, userID, days, statisticsPeriodDays).Scan(
		&prediction.CurrentBalance,
		&prediction.ExpectedIncome,
		&prediction.ExpectedExpense,
		&prediction.ScheduledCreditPayments,
		&prediction.PredictedBalance,
		&prediction.AverageDailyIncome,
		&prediction.AverageDailyExpense,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}

		return nil, fmt.Errorf("predict balance: %w", err)
	}

	return prediction, nil
}

func (r *AnalyticsRepository) getMonthlyIncome(ctx context.Context, userID int64) (string, error) {
	query := `
		SELECT COALESCE(SUM(t.amount), 0)::text
		FROM transactions t
		LEFT JOIN accounts to_acc ON to_acc.id = t.to_account_id
		LEFT JOIN accounts from_acc ON from_acc.id = t.from_account_id
		WHERE t.status = 'completed'
		  AND t.created_at >= date_trunc('month', NOW())
		  AND (
			(t.type IN ('deposit', 'credit_issue') AND t.user_id = $1)
			OR (
				t.type = 'transfer'
				AND to_acc.user_id = $1
				AND (from_acc.user_id IS NULL OR from_acc.user_id <> $1)
			)
		  )
	`

	var income string
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&income); err != nil {
		return "", fmt.Errorf("get monthly income: %w", err)
	}

	return income, nil
}

func (r *AnalyticsRepository) getMonthlyExpense(ctx context.Context, userID int64) (string, error) {
	query := `
		SELECT COALESCE(SUM(t.amount), 0)::text
		FROM transactions t
		LEFT JOIN accounts to_acc ON to_acc.id = t.to_account_id
		LEFT JOIN accounts from_acc ON from_acc.id = t.from_account_id
		WHERE t.status = 'completed'
		  AND t.created_at >= date_trunc('month', NOW())
		  AND (
			(t.type IN ('withdraw', 'card_payment', 'credit_payment', 'penalty') AND t.user_id = $1)
			OR (
				t.type = 'transfer'
				AND from_acc.user_id = $1
				AND (to_acc.user_id IS NULL OR to_acc.user_id <> $1)
			)
		  )
	`

	var expense string
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&expense); err != nil {
		return "", fmt.Errorf("get monthly expense: %w", err)
	}

	return expense, nil
}

func (r *AnalyticsRepository) getCreditLoad(ctx context.Context, userID int64) (string, error) {
	query := `
		SELECT COALESCE(SUM(monthly_payment), 0)::text
		FROM credits
		WHERE user_id = $1
		  AND status IN ('active', 'overdue')
	`

	var creditLoad string
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&creditLoad); err != nil {
		return "", fmt.Errorf("get credit load: %w", err)
	}

	return creditLoad, nil
}
