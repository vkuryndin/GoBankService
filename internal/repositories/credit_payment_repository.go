package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"
)

var ErrPaymentScheduleNotFound = errors.New("payment schedule not found")

type DueCreditPayment struct {
	ScheduleID    int64
	CreditID      int64
	UserID        int64
	AccountID     int64
	Amount        string
	PenaltyAmount string
	PaymentDate   time.Time
}

type CreditPaymentProcessResult struct {
	ScheduleID    int64
	CreditID      int64
	UserID        int64
	Amount        string
	PenaltyAmount string
	Status        string
}

type CreditPaymentRepository struct {
	db *sql.DB
}

func NewCreditPaymentRepository(db *sql.DB) *CreditPaymentRepository {
	return &CreditPaymentRepository{
		db: db,
	}
}

// FindDuePayments includes overdue rows so the scheduler can retry failed payments instead of losing them.
func (r *CreditPaymentRepository) FindDuePayments(ctx context.Context, limit int) ([]DueCreditPayment, error) {
	query := `
		SELECT
			ps.id,
			ps.credit_id,
			c.user_id,
			c.account_id,
			ps.amount::text,
			ps.penalty_amount::text,
			ps.payment_date
		FROM payment_schedules ps
		INNER JOIN credits c ON c.id = ps.credit_id
		WHERE ps.status IN ('pending', 'overdue')
		  AND ps.payment_date <= CURRENT_DATE
		  AND c.status IN ('active', 'overdue')
		ORDER BY ps.payment_date, ps.id
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("find due credit payments: %w", err)
	}
	defer rows.Close()

	payments := make([]DueCreditPayment, 0)

	for rows.Next() {
		var payment DueCreditPayment

		if err := rows.Scan(
			&payment.ScheduleID,
			&payment.CreditID,
			&payment.UserID,
			&payment.AccountID,
			&payment.Amount,
			&payment.PenaltyAmount,
			&payment.PaymentDate,
		); err != nil {
			return nil, fmt.Errorf("scan due credit payment: %w", err)
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due credit payments: %w", err)
	}

	return payments, nil
}

func (r *CreditPaymentRepository) ProcessPayment(
	ctx context.Context,
	payment DueCreditPayment,
) (*CreditPaymentProcessResult, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin credit payment transaction: %w", err)
	}
	defer tx.Rollback()

	lockedPayment, err := lockPaymentSchedule(ctx, tx, payment.ScheduleID)
	if err != nil {
		return nil, err
	}

	if lockedPayment.Status != "pending" && lockedPayment.Status != "overdue" {
		return &CreditPaymentProcessResult{
			ScheduleID:    lockedPayment.ScheduleID,
			CreditID:      lockedPayment.CreditID,
			UserID:        lockedPayment.UserID,
			Amount:        lockedPayment.Amount,
			PenaltyAmount: lockedPayment.PenaltyAmount,
			Status:        "skipped",
		}, nil
	}

	if err := lockAccount(ctx, tx, lockedPayment.AccountID, lockedPayment.UserID); err != nil {
		return nil, err
	}

	// Overdue schedules remain eligible for processing. Once the account has enough funds,
	// the scheduler charges the original payment plus the accumulated penalty.
	withdrawAmount, err := paymentAmountForWithdrawal(lockedPayment)
	if err != nil {
		return nil, err
	}

	_, err = withdrawAccountBalance(ctx, tx, lockedPayment.AccountID, withdrawAmount)
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			if lockedPayment.Status == "pending" {
				return r.markPaymentOverdue(ctx, tx, lockedPayment)
			}

			return &CreditPaymentProcessResult{
				ScheduleID:    lockedPayment.ScheduleID,
				CreditID:      lockedPayment.CreditID,
				UserID:        lockedPayment.UserID,
				Amount:        lockedPayment.Amount,
				PenaltyAmount: lockedPayment.PenaltyAmount,
				Status:        "overdue",
			}, nil
		}

		return nil, err
	}

	if err := markPaymentPaid(ctx, tx, lockedPayment.ScheduleID); err != nil {
		return nil, err
	}

	if _, err := createTransaction(
		ctx,
		tx,
		lockedPayment.UserID,
		&lockedPayment.AccountID,
		nil,
		lockedPayment.Amount,
		"credit_payment",
		"credit payment",
	); err != nil {
		return nil, err
	}

	if isPositiveMoney(lockedPayment.PenaltyAmount) {
		if _, err := createTransaction(
			ctx,
			tx,
			lockedPayment.UserID,
			&lockedPayment.AccountID,
			nil,
			lockedPayment.PenaltyAmount,
			"penalty",
			"credit payment penalty",
		); err != nil {
			return nil, err
		}
	}

	if err := closeCreditIfFullyPaid(ctx, tx, lockedPayment.CreditID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit credit payment transaction: %w", err)
	}

	return &CreditPaymentProcessResult{
		ScheduleID:    lockedPayment.ScheduleID,
		CreditID:      lockedPayment.CreditID,
		UserID:        lockedPayment.UserID,
		Amount:        lockedPayment.Amount,
		PenaltyAmount: lockedPayment.PenaltyAmount,
		Status:        "paid",
	}, nil
}

func (r *CreditPaymentRepository) markPaymentOverdue(
	ctx context.Context,
	tx *sql.Tx,
	payment *lockedPaymentSchedule,
) (*CreditPaymentProcessResult, error) {
	penaltyAmount, err := markPaymentOverdue(ctx, tx, payment.ScheduleID)
	if err != nil {
		return nil, err
	}

	if err := markCreditOverdue(ctx, tx, payment.CreditID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit overdue credit payment transaction: %w", err)
	}

	return &CreditPaymentProcessResult{
		ScheduleID:    payment.ScheduleID,
		CreditID:      payment.CreditID,
		UserID:        payment.UserID,
		Amount:        payment.Amount,
		PenaltyAmount: penaltyAmount,
		Status:        "overdue",
	}, nil
}

type lockedPaymentSchedule struct {
	ScheduleID    int64
	CreditID      int64
	UserID        int64
	AccountID     int64
	Amount        string
	PenaltyAmount string
	Status        string
}

func lockPaymentSchedule(
	ctx context.Context,
	tx *sql.Tx,
	scheduleID int64,
) (*lockedPaymentSchedule, error) {
	query := `
		SELECT
			ps.id,
			ps.credit_id,
			c.user_id,
			c.account_id,
			ps.amount::text,
			ps.penalty_amount::text,
			ps.status
		FROM payment_schedules ps
		INNER JOIN credits c ON c.id = ps.credit_id
		WHERE ps.id = $1
		FOR UPDATE OF ps, c
	`

	payment := &lockedPaymentSchedule{}

	err := tx.QueryRowContext(ctx, query, scheduleID).Scan(
		&payment.ScheduleID,
		&payment.CreditID,
		&payment.UserID,
		&payment.AccountID,
		&payment.Amount,
		&payment.PenaltyAmount,
		&payment.Status,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentScheduleNotFound
		}

		return nil, fmt.Errorf("lock payment schedule: %w", err)
	}

	return payment, nil
}

func markPaymentPaid(ctx context.Context, tx *sql.Tx, scheduleID int64) error {
	query := `
		UPDATE payment_schedules
		SET status = 'paid',
		    paid_at = NOW()
		WHERE id = $1
	`

	if _, err := tx.ExecContext(ctx, query, scheduleID); err != nil {
		return fmt.Errorf("mark payment paid: %w", err)
	}

	return nil
}

func markPaymentOverdue(ctx context.Context, tx *sql.Tx, scheduleID int64) (string, error) {
	query := `
		UPDATE payment_schedules
		SET status = 'overdue',
		    penalty_amount = CASE
		        WHEN penalty_amount > 0 THEN penalty_amount
		        ELSE ROUND(amount * 0.10, 2)
		    END
		WHERE id = $1
		RETURNING penalty_amount::text
	`

	var penaltyAmount string

	if err := tx.QueryRowContext(ctx, query, scheduleID).Scan(&penaltyAmount); err != nil {
		return "", fmt.Errorf("mark payment overdue: %w", err)
	}

	return penaltyAmount, nil
}

func markCreditOverdue(ctx context.Context, tx *sql.Tx, creditID int64) error {
	query := `
		UPDATE credits
		SET status = 'overdue'
		WHERE id = $1
	`

	if _, err := tx.ExecContext(ctx, query, creditID); err != nil {
		return fmt.Errorf("mark credit overdue: %w", err)
	}

	return nil
}

func closeCreditIfFullyPaid(ctx context.Context, tx *sql.Tx, creditID int64) error {
	query := `
		UPDATE credits
		SET status = 'closed'
		WHERE id = $1
		  AND NOT EXISTS (
			SELECT 1
			FROM payment_schedules
			WHERE credit_id = $1
			  AND status <> 'paid'
		  )
	`

	if _, err := tx.ExecContext(ctx, query, creditID); err != nil {
		return fmt.Errorf("close credit if fully paid: %w", err)
	}

	return nil
}

// paymentAmountForWithdrawal adds the penalty only when an overdue schedule is actually paid.
// This avoids writing a successful penalty transaction before money is collected.
func paymentAmountForWithdrawal(payment *lockedPaymentSchedule) (string, error) {
	if payment.Status != "overdue" || !isPositiveMoney(payment.PenaltyAmount) {
		return payment.Amount, nil
	}

	return addMoneyAmounts(payment.Amount, payment.PenaltyAmount)
}

func addMoneyAmounts(first string, second string) (string, error) {
	firstValue, ok := new(big.Rat).SetString(first)
	if !ok {
		return "", fmt.Errorf("invalid first money amount: %s", first)
	}

	secondValue, ok := new(big.Rat).SetString(second)
	if !ok {
		return "", fmt.Errorf("invalid second money amount: %s", second)
	}

	result := new(big.Rat).Add(firstValue, secondValue)

	return result.FloatString(2), nil
}

func isPositiveMoney(amount string) bool {
	value, ok := new(big.Rat).SetString(amount)
	if !ok {
		return false
	}

	return value.Sign() > 0
}
