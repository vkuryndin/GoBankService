package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bank-service/internal/models"
)

var (
	ErrCreditNotFound          = errors.New("credit not found")
	ErrInvalidCreditPrepayment = errors.New("invalid credit prepayment")
)

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

type CreditPolicyValidator func(summary *CreditRiskSummary) error

const (
	CreditPrepaymentModeReducePayment = "reduce_payment"
	CreditPrepaymentModeReduceTerm    = "reduce_term"
	CreditPrepaymentModeFullClose     = "full_close"
)

type CreditPrepaymentResult struct {
	Credit            *models.Credit
	TransactionID     int64
	Amount            string
	Mode              string
	OldMonthlyPayment string
	NewMonthlyPayment string
	OldTermMonths     int
	NewTermMonths     int
	RemainingDebt     string
	Closed            bool
}

type pendingCreditPayment struct {
	ID          int64
	PaymentDate time.Time
	AmountCents int64
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
	mfaCodeID int64,
	validatePolicy CreditPolicyValidator,
) (*models.Credit, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin credit transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockUserCreditCreation(ctx, tx, userID); err != nil {
		return nil, err
	}

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, err
	}

	if validatePolicy != nil {
		summary, err := getCreditRiskSummaryTx(ctx, tx, userID)
		if err != nil {
			return nil, err
		}

		if err := validatePolicy(summary); err != nil {
			return nil, err
		}
	}

	if err := markMFACodeUsedTx(ctx, tx, mfaCodeID); err != nil {
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
		nil,
		nil,
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

func (r *CreditRepository) Prepay(
	ctx context.Context,
	userID int64,
	creditID int64,
	amount string,
	mode string,
	mfaCodeID int64,
) (*CreditPrepaymentResult, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin credit prepayment transaction: %w", err)
	}
	defer rollbackTx(tx)

	credit, err := lockCreditForPrepayment(ctx, tx, creditID, userID)
	if err != nil {
		return nil, err
	}

	if credit.Status != "active" {
		return nil, ErrInvalidCreditPrepayment
	}

	if err := lockAccount(ctx, tx, credit.AccountID, userID); err != nil {
		return nil, err
	}

	hasOverdue, err := creditHasOverdueSchedule(ctx, tx, creditID)
	if err != nil {
		return nil, err
	}
	if hasOverdue {
		return nil, ErrInvalidCreditPrepayment
	}

	payments, err := findPendingCreditPaymentsForUpdate(ctx, tx, creditID)
	if err != nil {
		return nil, err
	}
	if len(payments) == 0 {
		return nil, ErrInvalidCreditPrepayment
	}

	amountCents, err := moneyStringToCents(amount)
	if err != nil || amountCents <= 0 {
		return nil, ErrInvalidCreditPrepayment
	}

	outstandingCents := sumPendingPaymentCents(payments)
	if amountCents > outstandingCents {
		return nil, ErrInvalidCreditPrepayment
	}

	if err := markMFACodeUsedTx(ctx, tx, mfaCodeID); err != nil {
		return nil, err
	}

	if _, err := withdrawAccountBalance(ctx, tx, credit.AccountID, amount); err != nil {
		return nil, err
	}

	transactionID, err := createTransaction(
		ctx,
		tx,
		userID,
		&credit.AccountID,
		nil,
		nil,
		nil,
		amount,
		"credit_prepayment",
		"credit prepayment",
	)
	if err != nil {
		return nil, err
	}

	result := &CreditPrepaymentResult{
		TransactionID:     transactionID,
		Amount:            amount,
		Mode:              mode,
		OldMonthlyPayment: credit.MonthlyPayment,
		OldTermMonths:     credit.TermMonths,
	}

	if mode == CreditPrepaymentModeFullClose && amountCents != outstandingCents {
		return nil, ErrInvalidCreditPrepayment
	}

	remainingCents := outstandingCents - amountCents
	if remainingCents == 0 {
		if err := markPendingCreditPaymentsPaid(ctx, tx, creditID); err != nil {
			return nil, err
		}

		updatedCredit, err := closeCreditAfterPrepayment(ctx, tx, creditID)
		if err != nil {
			return nil, err
		}

		result.Credit = updatedCredit
		result.NewMonthlyPayment = "0.00"
		result.NewTermMonths = 0
		result.RemainingDebt = "0.00"
		result.Closed = true

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit credit prepayment transaction: %w", err)
		}

		return result, nil
	}

	var newMonthlyPayment string
	var newTermMonths int

	switch mode {
	case CreditPrepaymentModeReduceTerm:
		newMonthlyPayment, newTermMonths, err = applyReduceTermPrepayment(ctx, tx, payments, amountCents)
	case CreditPrepaymentModeReducePayment:
		newMonthlyPayment, newTermMonths, err = applyReducePaymentPrepayment(ctx, tx, payments, remainingCents)
	case CreditPrepaymentModeFullClose:
		return nil, ErrInvalidCreditPrepayment
	default:
		return nil, ErrInvalidCreditPrepayment
	}
	if err != nil {
		return nil, err
	}

	updatedCredit, err := updateCreditPaymentPlan(ctx, tx, creditID, newTermMonths, newMonthlyPayment, centsToMoneyString(remainingCents))
	if err != nil {
		return nil, err
	}

	result.Credit = updatedCredit
	result.NewMonthlyPayment = newMonthlyPayment
	result.NewTermMonths = newTermMonths
	result.RemainingDebt = centsToMoneyString(remainingCents)
	result.Closed = false

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit credit prepayment transaction: %w", err)
	}

	return result, nil
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
	return getCreditRiskSummary(ctx, r.db, userID)
}

type creditRiskQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func getCreditRiskSummary(ctx context.Context, queryer creditRiskQueryer, userID int64) (*CreditRiskSummary, error) {
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
	if err := queryer.QueryRowContext(ctx, query, userID).Scan(
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

func getCreditRiskSummaryTx(ctx context.Context, tx *sql.Tx, userID int64) (*CreditRiskSummary, error) {
	return getCreditRiskSummary(ctx, tx, userID)
}

func lockUserCreditCreation(ctx context.Context, tx *sql.Tx, userID int64) error {
	const creditCreationLockNamespace int64 = 920000000000

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, creditCreationLockNamespace+userID); err != nil {
		return fmt.Errorf("lock user credit creation: %w", err)
	}

	return nil
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

func lockCreditForPrepayment(ctx context.Context, tx *sql.Tx, creditID int64, userID int64) (*models.Credit, error) {
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
		FOR UPDATE
	`

	credit := &models.Credit{}
	err := tx.QueryRowContext(ctx, query, creditID, userID).Scan(
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

		return nil, fmt.Errorf("lock credit for prepayment: %w", err)
	}

	return credit, nil
}

func creditHasOverdueSchedule(ctx context.Context, tx *sql.Tx, creditID int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM payment_schedules
			WHERE credit_id = $1
			  AND status = 'overdue'
		)
	`

	var exists bool
	if err := tx.QueryRowContext(ctx, query, creditID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check overdue schedule: %w", err)
	}

	return exists, nil
}

func findPendingCreditPaymentsForUpdate(ctx context.Context, tx *sql.Tx, creditID int64) ([]pendingCreditPayment, error) {
	query := `
		SELECT id, payment_date, amount::text
		FROM payment_schedules
		WHERE credit_id = $1
		  AND status = 'pending'
		ORDER BY payment_date, id
		FOR UPDATE
	`

	rows, err := tx.QueryContext(ctx, query, creditID)
	if err != nil {
		return nil, fmt.Errorf("find pending credit payments for update: %w", err)
	}
	defer closeRows(rows)

	payments := make([]pendingCreditPayment, 0)

	for rows.Next() {
		var payment pendingCreditPayment
		var amount string

		if err := rows.Scan(&payment.ID, &payment.PaymentDate, &amount); err != nil {
			return nil, fmt.Errorf("scan pending credit payment: %w", err)
		}

		amountCents, err := moneyStringToCents(amount)
		if err != nil {
			return nil, err
		}
		payment.AmountCents = amountCents

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending credit payments: %w", err)
	}

	return payments, nil
}

func sumPendingPaymentCents(payments []pendingCreditPayment) int64 {
	var total int64
	for _, payment := range payments {
		total += payment.AmountCents
	}

	return total
}

func applyReduceTermPrepayment(
	ctx context.Context,
	tx *sql.Tx,
	payments []pendingCreditPayment,
	prepaymentCents int64,
) (string, int, error) {
	remainingPrepayment := prepaymentCents

	for i := len(payments) - 1; i >= 0 && remainingPrepayment > 0; i-- {
		payment := payments[i]

		if remainingPrepayment >= payment.AmountCents {
			if err := markPaymentSchedulePaidByPrepayment(ctx, tx, payment.ID); err != nil {
				return "", 0, err
			}
			remainingPrepayment -= payment.AmountCents
			payments[i].AmountCents = 0
			continue
		}

		newAmountCents := payment.AmountCents - remainingPrepayment
		if newAmountCents <= 0 {
			return "", 0, ErrInvalidCreditPrepayment
		}

		if err := updatePaymentScheduleAmount(ctx, tx, payment.ID, centsToMoneyString(newAmountCents)); err != nil {
			return "", 0, err
		}
		payments[i].AmountCents = newAmountCents
		remainingPrepayment = 0
	}

	remainingPayments := filterPositivePendingPayments(payments)
	if len(remainingPayments) == 0 {
		return "0.00", 0, nil
	}

	return centsToMoneyString(remainingPayments[0].AmountCents), len(remainingPayments), nil
}

func applyReducePaymentPrepayment(
	ctx context.Context,
	tx *sql.Tx,
	payments []pendingCreditPayment,
	remainingDebtCents int64,
) (string, int, error) {
	if len(payments) == 0 || remainingDebtCents <= 0 {
		return "", 0, ErrInvalidCreditPrepayment
	}

	if remainingDebtCents < int64(len(payments)) {
		return "", 0, ErrInvalidCreditPrepayment
	}

	basePaymentCents := remainingDebtCents / int64(len(payments))
	extraCents := remainingDebtCents % int64(len(payments))
	newMonthlyPayment := "0.00"

	for index, payment := range payments {
		nextAmountCents := basePaymentCents
		if int64(index) < extraCents {
			nextAmountCents++
		}
		if nextAmountCents <= 0 {
			return "", 0, ErrInvalidCreditPrepayment
		}

		if index == 0 {
			newMonthlyPayment = centsToMoneyString(nextAmountCents)
		}

		if err := updatePaymentScheduleAmount(ctx, tx, payment.ID, centsToMoneyString(nextAmountCents)); err != nil {
			return "", 0, err
		}
	}

	return newMonthlyPayment, len(payments), nil
}

func filterPositivePendingPayments(payments []pendingCreditPayment) []pendingCreditPayment {
	result := make([]pendingCreditPayment, 0, len(payments))
	for _, payment := range payments {
		if payment.AmountCents > 0 {
			result = append(result, payment)
		}
	}

	return result
}

func markPaymentSchedulePaidByPrepayment(ctx context.Context, tx *sql.Tx, scheduleID int64) error {
	query := `
		UPDATE payment_schedules
		SET status = 'paid',
		    paid_at = NOW()
		WHERE id = $1
		  AND status = 'pending'
	`

	if _, err := tx.ExecContext(ctx, query, scheduleID); err != nil {
		return fmt.Errorf("mark payment schedule paid by prepayment: %w", err)
	}

	return nil
}

func markPendingCreditPaymentsPaid(ctx context.Context, tx *sql.Tx, creditID int64) error {
	query := `
		UPDATE payment_schedules
		SET status = 'paid',
		    paid_at = NOW()
		WHERE credit_id = $1
		  AND status = 'pending'
	`

	if _, err := tx.ExecContext(ctx, query, creditID); err != nil {
		return fmt.Errorf("mark pending credit payments paid: %w", err)
	}

	return nil
}

func updatePaymentScheduleAmount(ctx context.Context, tx *sql.Tx, scheduleID int64, amount string) error {
	query := `
		UPDATE payment_schedules
		SET amount = $2::numeric
		WHERE id = $1
		  AND status = 'pending'
	`

	if _, err := tx.ExecContext(ctx, query, scheduleID, amount); err != nil {
		return fmt.Errorf("update payment schedule amount: %w", err)
	}

	return nil
}

func updateCreditPaymentPlan(
	ctx context.Context,
	tx *sql.Tx,
	creditID int64,
	termMonths int,
	monthlyPayment string,
	remainingPrincipal string,
) (*models.Credit, error) {
	query := `
		UPDATE credits
		SET term_months = $2,
		    monthly_payment = $3::numeric,
		    principal_amount = $4::numeric,
		    status = 'active'
		WHERE id = $1
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
	if err := tx.QueryRowContext(ctx, query, creditID, termMonths, monthlyPayment, remainingPrincipal).Scan(
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
		return nil, fmt.Errorf("update credit payment plan: %w", err)
	}

	return credit, nil
}

func closeCreditAfterPrepayment(ctx context.Context, tx *sql.Tx, creditID int64) (*models.Credit, error) {
	query := `
		UPDATE credits
		SET principal_amount = 0.00,
		    status = 'closed'
		WHERE id = $1
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
	if err := tx.QueryRowContext(ctx, query, creditID).Scan(
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
		return nil, fmt.Errorf("close credit after prepayment: %w", err)
	}

	return credit, nil
}

func moneyStringToCents(amount string) (int64, error) {
	normalized := strings.TrimSpace(amount)
	if normalized == "" {
		return 0, ErrInvalidCreditPrepayment
	}

	parts := strings.Split(normalized, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidCreditPrepayment
	}

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return 0, ErrInvalidCreditPrepayment
	}

	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 2 {
			return 0, ErrInvalidCreditPrepayment
		}
		fraction = fraction + strings.Repeat("0", 2-len(fraction))
	}

	fractionValue, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil || fractionValue < 0 {
		return 0, ErrInvalidCreditPrepayment
	}

	return whole*100 + fractionValue, nil
}

func centsToMoneyString(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
