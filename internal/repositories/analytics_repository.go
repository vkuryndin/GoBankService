package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type MonthlyAnalytics struct {
	Income             string
	Expense            string
	CreditLoad         string
	ActiveCreditsCount int64
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

	activeCreditsCount, err := r.getActiveCreditsCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &MonthlyAnalytics{
		Income:             income,
		Expense:            expense,
		CreditLoad:         creditLoad,
		ActiveCreditsCount: activeCreditsCount,
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

func (r *AnalyticsRepository) getActiveCreditsCount(ctx context.Context, userID int64) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM credits
		WHERE user_id = $1
		  AND status = 'active'
	`

	var count int64
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get active credits count: %w", err)
	}

	return count, nil
}

type OperationStatistics struct {
	EntityType     string
	EntityID       int64
	Currency       string
	OperationCount int64
	IncomeCount    int64
	ExpenseCount   int64
	TotalIncome    string
	TotalExpense   string
	NetAmount      string
	Operations     []OperationEntry
	ByType         []OperationTypeStatistics
	ByStatus       []OperationStatusStatistics
}

type OperationEntry struct {
	ID            int64
	Direction     string
	Type          string
	Status        string
	Amount        string
	Currency      string
	Description   string
	FromAccountID sql.NullInt64
	ToAccountID   sql.NullInt64
	FromCardID    sql.NullInt64
	ToCardID      sql.NullInt64
	CreatedAt     string
}

type OperationTypeStatistics struct {
	Type         string
	Count        int64
	TotalIncome  string
	TotalExpense string
	NetAmount    string
}

type OperationStatusStatistics struct {
	Status      string
	Count       int64
	TotalAmount string
}

func (r *AnalyticsRepository) GetAccountOperationStatistics(
	ctx context.Context,
	userID int64,
	accountID int64,
	limit int,
) (*OperationStatistics, error) {
	if err := r.ensureAccountBelongsToUser(ctx, userID, accountID); err != nil {
		return nil, err
	}

	return r.getOperationStatistics(ctx, accountOperationStatisticsQuery(), accountID, "account", accountID, limit)
}

func (r *AnalyticsRepository) GetCardOperationStatistics(
	ctx context.Context,
	userID int64,
	cardID int64,
	limit int,
) (*OperationStatistics, error) {
	if err := r.ensureCardBelongsToUser(ctx, userID, cardID); err != nil {
		return nil, err
	}

	return r.getOperationStatistics(ctx, cardOperationStatisticsQuery(), cardID, "card", cardID, limit)
}

func (r *AnalyticsRepository) ensureAccountBelongsToUser(ctx context.Context, userID int64, accountID int64) error {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM accounts
			WHERE id = $1 AND user_id = $2
		)
	`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, accountID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("check account ownership for statistics: %w", err)
	}

	if !exists {
		return ErrAccountNotFound
	}

	return nil
}

func (r *AnalyticsRepository) ensureCardBelongsToUser(ctx context.Context, userID int64, cardID int64) error {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM cards
			WHERE id = $1 AND user_id = $2
		)
	`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, cardID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("check card ownership for statistics: %w", err)
	}

	if !exists {
		return ErrCardNotFound
	}

	return nil
}

type operationStatisticsQueries struct {
	summary  string
	byType   string
	byStatus string
	entries  string
}

func (r *AnalyticsRepository) getOperationStatistics(
	ctx context.Context,
	queries operationStatisticsQueries,
	entityID int64,
	entityType string,
	responseEntityID int64,
	limit int,
) (*OperationStatistics, error) {
	stats := &OperationStatistics{
		EntityType: entityType,
		EntityID:   responseEntityID,
		Currency:   "RUB",
	}

	if err := r.db.QueryRowContext(ctx, queries.summary, entityID).Scan(
		&stats.OperationCount,
		&stats.IncomeCount,
		&stats.ExpenseCount,
		&stats.TotalIncome,
		&stats.TotalExpense,
		&stats.NetAmount,
	); err != nil {
		return nil, fmt.Errorf("get operation statistics summary: %w", err)
	}

	byType, err := r.getOperationTypeStatistics(ctx, queries.byType, entityID)
	if err != nil {
		return nil, err
	}
	stats.ByType = byType

	byStatus, err := r.getOperationStatusStatistics(ctx, queries.byStatus, entityID)
	if err != nil {
		return nil, err
	}
	stats.ByStatus = byStatus

	operations, err := r.getOperationEntries(ctx, queries.entries, entityID, limit)
	if err != nil {
		return nil, err
	}
	stats.Operations = operations

	return stats, nil
}

func (r *AnalyticsRepository) getOperationTypeStatistics(
	ctx context.Context,
	query string,
	entityID int64,
) ([]OperationTypeStatistics, error) {
	rows, err := r.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("get operation type statistics: %w", err)
	}
	defer closeRows(rows)

	items := make([]OperationTypeStatistics, 0)
	for rows.Next() {
		var item OperationTypeStatistics
		if err := rows.Scan(
			&item.Type,
			&item.Count,
			&item.TotalIncome,
			&item.TotalExpense,
			&item.NetAmount,
		); err != nil {
			return nil, fmt.Errorf("scan operation type statistics: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation type statistics: %w", err)
	}

	return items, nil
}

func (r *AnalyticsRepository) getOperationStatusStatistics(
	ctx context.Context,
	query string,
	entityID int64,
) ([]OperationStatusStatistics, error) {
	rows, err := r.db.QueryContext(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("get operation status statistics: %w", err)
	}
	defer closeRows(rows)

	items := make([]OperationStatusStatistics, 0)
	for rows.Next() {
		var item OperationStatusStatistics
		if err := rows.Scan(&item.Status, &item.Count, &item.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan operation status statistics: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation status statistics: %w", err)
	}

	return items, nil
}

func (r *AnalyticsRepository) getOperationEntries(
	ctx context.Context,
	query string,
	entityID int64,
	limit int,
) ([]OperationEntry, error) {
	rows, err := r.db.QueryContext(ctx, query, entityID, limit)
	if err != nil {
		return nil, fmt.Errorf("get operation entries: %w", err)
	}
	defer closeRows(rows)

	items := make([]OperationEntry, 0)
	for rows.Next() {
		var item OperationEntry
		var description sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Direction,
			&item.Type,
			&item.Status,
			&item.Amount,
			&item.Currency,
			&description,
			&item.FromAccountID,
			&item.ToAccountID,
			&item.FromCardID,
			&item.ToCardID,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan operation entry: %w", err)
		}

		if description.Valid {
			item.Description = description.String
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation entries: %w", err)
	}

	return items, nil
}

func accountOperationStatisticsQuery() operationStatisticsQueries {
	base := `
		WITH base AS (
			SELECT
				t.id,
				CASE
					WHEN t.to_account_id = $1 AND (t.from_account_id IS NULL OR t.from_account_id <> $1) THEN 'income'
					WHEN t.from_account_id = $1 AND (t.to_account_id IS NULL OR t.to_account_id <> $1) THEN 'expense'
					ELSE 'neutral'
				END AS direction,
				t.type,
				t.status,
				t.amount,
				t.currency,
				t.description,
				t.from_account_id,
				t.to_account_id,
				t.from_card_id,
				t.to_card_id,
				t.created_at
			FROM transactions t
			WHERE t.from_account_id = $1 OR t.to_account_id = $1
		)
	`

	return operationStatisticsQueries{
		summary: base + `
			SELECT
				COUNT(*) AS operation_count,
				COUNT(*) FILTER (WHERE direction = 'income') AS income_count,
				COUNT(*) FILTER (WHERE direction = 'expense') AS expense_count,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)::text AS total_income,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)::text AS total_expense,
				(
					COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)
					- COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)
				)::text AS net_amount
			FROM base
		`,
		byType: base + `
			SELECT
				type,
				COUNT(*) AS operation_count,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)::text AS total_income,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)::text AS total_expense,
				(
					COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)
					- COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)
				)::text AS net_amount
			FROM base
			GROUP BY type
			ORDER BY type
		`,
		byStatus: base + `
			SELECT status, COUNT(*) AS operation_count, COALESCE(SUM(amount), 0)::text AS total_amount
			FROM base
			GROUP BY status
			ORDER BY status
		`,
		entries: base + `
			SELECT
				id,
				direction,
				type,
				status,
				amount::text,
				currency,
				description,
				from_account_id,
				to_account_id,
				from_card_id,
				to_card_id,
				created_at::text
			FROM base
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`,
	}
}

func cardOperationStatisticsQuery() operationStatisticsQueries {
	base := `
		WITH base AS (
			SELECT
				t.id,
				CASE
					WHEN t.to_card_id = $1 AND (t.from_card_id IS NULL OR t.from_card_id <> $1) THEN 'income'
					WHEN t.from_card_id = $1 AND (t.to_card_id IS NULL OR t.to_card_id <> $1) THEN 'expense'
					ELSE 'neutral'
				END AS direction,
				t.type,
				t.status,
				t.amount,
				t.currency,
				t.description,
				t.from_account_id,
				t.to_account_id,
				t.from_card_id,
				t.to_card_id,
				t.created_at
			FROM transactions t
			WHERE t.from_card_id = $1 OR t.to_card_id = $1
		)
	`

	return operationStatisticsQueries{
		summary: base + `
			SELECT
				COUNT(*) AS operation_count,
				COUNT(*) FILTER (WHERE direction = 'income') AS income_count,
				COUNT(*) FILTER (WHERE direction = 'expense') AS expense_count,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)::text AS total_income,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)::text AS total_expense,
				(
					COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)
					- COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)
				)::text AS net_amount
			FROM base
		`,
		byType: base + `
			SELECT
				type,
				COUNT(*) AS operation_count,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)::text AS total_income,
				COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)::text AS total_expense,
				(
					COALESCE(SUM(amount) FILTER (WHERE direction = 'income' AND status = 'completed'), 0)
					- COALESCE(SUM(amount) FILTER (WHERE direction = 'expense' AND status = 'completed'), 0)
				)::text AS net_amount
			FROM base
			GROUP BY type
			ORDER BY type
		`,
		byStatus: base + `
			SELECT status, COUNT(*) AS operation_count, COALESCE(SUM(amount), 0)::text AS total_amount
			FROM base
			GROUP BY status
			ORDER BY status
		`,
		entries: base + `
			SELECT
				id,
				direction,
				type,
				status,
				amount::text,
				currency,
				description,
				from_account_id,
				to_account_id,
				from_card_id,
				to_card_id,
				created_at::text
			FROM base
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`,
	}
}
