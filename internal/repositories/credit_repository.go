package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"bank-service/internal/models"
)

var ErrCreditNotFound = errors.New("credit not found")

type PaymentScheduleInput struct {
	PaymentDate time.Time
	Amount      string
}

type CreditRiskSummary struct {
	ActiveCreditsCount   int
	OverdueCreditsCount  int
	TotalPrincipalAmount string
	TotalMonthlyPayment  string
	MonthlyIncome        string
}

type CreditRepository struct {
	db *sql.DB
}

func NewCreditRepository(db *sql.DB) *CreditRepository {
	return &CreditRepository{
		db: db,
	}
}

// Credit creation, schedule generation, account funding and transaction history are committed atomically.
// A partially issued credit would leave the balance, repayment schedule and audit trail inconsistent.
func (r *CreditRepository) CreateWithScheduleAndIssue(
	ctx context.Context,
	userID int64,
	accountID int64,
	principalAmount string,
	interestRate string,
	termMonths int,
	monthlyPayment string,
	schedule []PaymentScheduleInput,
) (*models.Credit, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin credit transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, err
	}

	credit, err := createCredit(
		ctx,
		tx,
		userID,
		accountID,
		principalAmount,
		interestRate,
		termMonths,
		monthlyPayment,
	)
	if err != nil {
		return nil, err
	}

	for _, payment := range schedule {
		if err := createPaymentSchedule(ctx, tx, credit.ID, payment); err != nil {
			return nil, err
		}
	}

	if _, err := updateAccountBalance(ctx, tx, accountID, principalAmount, "+"); err != nil {
		return nil, fmt.Errorf("issue credit balance update: %w", err)
	}

	if _, err := createTransaction(
		ctx,
		tx,
		userID,
		nil,
		&accountID,
		principalAmount,
		"credit_issue",
		"credit issued",
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit credit transaction: %w", err)
	}

	return credit, nil
}

func (r *CreditRepository) FindByUserID(ctx context.Context, userID int64) ([]models.Credit, error) {
	query := `
		SELECT
			id,
			user_id,
			account_id,
			principal_amount::text,
			interest_rate::text,
			term_months,
			monthly_payment::text,
			status,
			created_at
		FROM credits
		WHERE user_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("find credits by user id: %w", err)
	}
	defer closeRows(rows)

	credits := make([]models.Credit, 0)

	for rows.Next() {
		var credit models.Credit

		if err := rows.Scan(
			&credit.ID,
			&credit.UserID,
			&credit.AccountID,
			&credit.PrincipalAmount,
			&credit.InterestRate,
			&credit.TermMonths,
			&credit.MonthlyPayment,
			&credit.Status,
			&credit.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan credit: %w", err)
		}

		credits = append(credits, credit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credits: %w", err)
	}

	return credits, nil
}

func (r *CreditRepository) FindByIDAndUserID(ctx context.Context, creditID, userID int64) (*models.Credit, error) {
	query := `
		SELECT
			id,
			user_id,
			account_id,
			principal_amount::text,
			interest_rate::text,
			term_months,
			monthly_payment::text,
			status,
			created_at
		FROM credits
		WHERE id = $1 AND user_id = $2
	`

	credit := &models.Credit{}

	err := r.db.QueryRowContext(ctx, query, creditID, userID).Scan(
		&credit.ID,
		&credit.UserID,
		&credit.AccountID,
		&credit.PrincipalAmount,
		&credit.InterestRate,
		&credit.TermMonths,
		&credit.MonthlyPayment,
		&credit.Status,
		&credit.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCreditNotFound
		}

		return nil, fmt.Errorf("find credit by id and user id: %w", err)
	}

	return credit, nil
}

func (r *CreditRepository) FindScheduleByCreditIDAndUserID(
	ctx context.Context,
	creditID int64,
	userID int64,
) ([]models.PaymentSchedule, error) {
	query := `
		SELECT
			ps.id,
			ps.credit_id,
			ps.payment_date,
			ps.amount::text,
			ps.penalty_amount::text,
			ps.status,
			ps.paid_at,
			ps.created_at
		FROM payment_schedules ps
		INNER JOIN credits c ON c.id = ps.credit_id
		WHERE ps.credit_id = $1 AND c.user_id = $2
		ORDER BY ps.payment_date, ps.id
	`

	rows, err := r.db.QueryContext(ctx, query, creditID, userID)
	if err != nil {
		return nil, fmt.Errorf("find payment schedule: %w", err)
	}
	defer closeRows(rows)

	schedule := make([]models.PaymentSchedule, 0)

	for rows.Next() {
		var payment models.PaymentSchedule

		if err := rows.Scan(
			&payment.ID,
			&payment.CreditID,
			&payment.PaymentDate,
			&payment.Amount,
			&payment.PenaltyAmount,
			&payment.Status,
			&payment.PaidAt,
			&payment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment schedule: %w", err)
		}

		schedule = append(schedule, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment schedule: %w", err)
	}

	if len(schedule) == 0 {
		if _, err := r.FindByIDAndUserID(ctx, creditID, userID); err != nil {
			return nil, err
		}
	}

	return schedule, nil
}

func (r *CreditRepository) GetCreditRiskSummary(ctx context.Context, userID int64) (*CreditRiskSummary, error) {
	query := `
		WITH credit_data AS (
			SELECT
				COUNT(*) FILTER (WHERE status IN ('active', 'overdue')) AS active_count,
				COUNT(*) FILTER (WHERE status = 'overdue') AS overdue_count,
				COALESCE(SUM(principal_amount) FILTER (WHERE status IN ('active', 'overdue')), 0)::text AS total_principal,
				COALESCE(SUM(monthly_payment) FILTER (WHERE status IN ('active', 'overdue')), 0)::text AS total_monthly_payment
			FROM credits
			WHERE user_id = $1
		),
		income_data AS (
			SELECT COALESCE(SUM(t.amount), 0)::text AS monthly_income
			FROM transactions t
			LEFT JOIN accounts to_acc ON to_acc.id = t.to_account_id
			LEFT JOIN accounts from_acc ON from_acc.id = t.from_account_id
			WHERE t.status = 'completed'
			  AND t.created_at >= NOW() - INTERVAL '30 days'
			  AND (
				(t.type = 'deposit' AND t.user_id = $1)
				OR (
					t.type = 'transfer'
					AND to_acc.user_id = $1
					AND (from_acc.user_id IS NULL OR from_acc.user_id <> $1)
				)
			  )
		)
		SELECT
			cd.active_count,
			cd.overdue_count,
			cd.total_principal,
			cd.total_monthly_payment,
			id.monthly_income
		FROM credit_data cd
		CROSS JOIN income_data id
	`

	summary := &CreditRiskSummary{}
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&summary.ActiveCreditsCount,
		&summary.OverdueCreditsCount,
		&summary.TotalPrincipalAmount,
		&summary.TotalMonthlyPayment,
		&summary.MonthlyIncome,
	); err != nil {
		return nil, fmt.Errorf("get credit risk summary: %w", err)
	}

	return summary, nil
}

func createCredit(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	accountID int64,
	principalAmount string,
	interestRate string,
	termMonths int,
	monthlyPayment string,
) (*models.Credit, error) {
	query := `
		INSERT INTO credits (
			user_id,
			account_id,
			principal_amount,
			interest_rate,
			term_months,
			monthly_payment,
			status
		)
		VALUES ($1, $2, $3::numeric, $4::numeric, $5, $6::numeric, 'active')
		RETURNING
			id,
			user_id,
			account_id,
			principal_amount::text,
			interest_rate::text,
			term_months,
			monthly_payment::text,
			status,
			created_at
	`

	credit := &models.Credit{}

	err := tx.QueryRowContext(
		ctx,
		query,
		userID,
		accountID,
		principalAmount,
		interestRate,
		termMonths,
		monthlyPayment,
	).Scan(
		&credit.ID,
		&credit.UserID,
		&credit.AccountID,
		&credit.PrincipalAmount,
		&credit.InterestRate,
		&credit.TermMonths,
		&credit.MonthlyPayment,
		&credit.Status,
		&credit.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create credit: %w", err)
	}

	return credit, nil
}

func createPaymentSchedule(ctx context.Context, tx *sql.Tx, creditID int64, payment PaymentScheduleInput) error {
	query := `
		INSERT INTO payment_schedules (
			credit_id,
			payment_date,
			amount,
			status
		)
		VALUES ($1, $2, $3::numeric, 'pending')
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		creditID,
		payment.PaymentDate.Format("2006-01-02"),
		payment.Amount,
	)
	if err != nil {
		return fmt.Errorf("create payment schedule: %w", err)
	}

	return nil
}
