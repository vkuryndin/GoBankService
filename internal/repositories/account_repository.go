package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"bank-service/internal/models"
)

var (
	ErrAccountNotFound          = errors.New("account not found")
	ErrInsufficientFunds        = errors.New("insufficient funds")
	ErrAccountBlocked           = errors.New("account is blocked")
	ErrAccountClosed            = errors.New("account is closed")
	ErrAccountAlreadyClosed     = errors.New("account already closed")
	ErrAccountBalanceMustBeZero = errors.New("account balance must be zero")
	ErrAccountHasActiveCredit   = errors.New("account has active credit")
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, userID int64, accountNumber string) (*models.Account, error) {
	query := `
		INSERT INTO accounts (user_id, account_number)
		VALUES ($1, $2)
		RETURNING id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
	`

	account := &models.Account{}

	err := r.db.QueryRowContext(ctx, query, userID, accountNumber).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.Status,
		&account.ClosedAt,
		&account.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

func (r *AccountRepository) FindByUserID(ctx context.Context, userID int64) ([]models.Account, error) {
	query := `
		SELECT id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("find accounts by user id: %w", err)
	}
	defer closeRows(rows)

	accounts := make([]models.Account, 0)

	for rows.Next() {
		var account models.Account

		if err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.AccountNumber,
			&account.Balance,
			&account.Currency,
			&account.IsBlocked,
			&account.Status,
			&account.ClosedAt,
			&account.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}

	return accounts, nil
}

func (r *AccountRepository) FindByIDAndUserID(ctx context.Context, accountID, userID int64) (*models.Account, error) {
	query := `
		SELECT id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
		FROM accounts
		WHERE id = $1 AND user_id = $2
	`

	account := &models.Account{}

	err := r.db.QueryRowContext(ctx, query, accountID, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.Status,
		&account.ClosedAt,
		&account.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}

		return nil, fmt.Errorf("find account by id and user id: %w", err)
	}

	return account, nil
}

// ValidateTransferAccounts keeps transfers safe before an MFA code is issued.
// The source account must belong to the authenticated user; the destination account may belong to another user,
// but both accounts have to exist and be available for operations.
func (r *AccountRepository) ValidateTransferAccounts(
	ctx context.Context,
	userID int64,
	fromAccountID int64,
	toAccountID int64,
) error {
	query := `
		SELECT
			EXISTS (
				SELECT 1 FROM accounts
				WHERE id = $1 AND user_id = $2
			),
			EXISTS (
				SELECT 1 FROM accounts
				WHERE id = $3
			),
			EXISTS (
				SELECT 1 FROM accounts
				WHERE id = $1 AND user_id = $2 AND is_blocked = TRUE
			),
			EXISTS (
				SELECT 1 FROM accounts
				WHERE id = $3 AND is_blocked = TRUE
			),
			EXISTS (
				SELECT 1 FROM accounts
				WHERE id = $1 AND user_id = $2 AND status = 'closed'
			),
			EXISTS (
				SELECT 1 FROM accounts
				WHERE id = $3 AND status = 'closed'
			)
	`

	var fromExists bool
	var toExists bool
	var fromBlocked bool
	var toBlocked bool
	var fromClosed bool
	var toClosed bool

	err := r.db.QueryRowContext(ctx, query, fromAccountID, userID, toAccountID).Scan(
		&fromExists,
		&toExists,
		&fromBlocked,
		&toBlocked,
		&fromClosed,
		&toClosed,
	)
	if err != nil {
		return fmt.Errorf("validate transfer accounts: %w", err)
	}

	if !fromExists || !toExists {
		return ErrAccountNotFound
	}

	if fromClosed || toClosed {
		return ErrAccountClosed
	}

	if fromBlocked || toBlocked {
		return ErrAccountBlocked
	}

	return nil
}

func (r *AccountRepository) Deposit(ctx context.Context, userID, accountID int64, amount, description string) (*models.Account, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("begin deposit transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, 0, err
	}

	account, err := updateAccountBalance(ctx, tx, accountID, amount, "+")
	if err != nil {
		return nil, 0, fmt.Errorf("deposit balance update: %w", err)
	}

	transactionID, err := createTransaction(ctx, tx, userID, nil, &accountID, amount, "deposit", description)
	if err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit deposit transaction: %w", err)
	}

	return account, transactionID, nil
}

func (r *AccountRepository) Withdraw(ctx context.Context, userID, accountID int64, amount, description string) (*models.Account, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("begin withdraw transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, 0, err
	}

	account, err := withdrawAccountBalance(ctx, tx, accountID, amount)
	if err != nil {
		return nil, 0, err
	}

	transactionID, err := createTransaction(ctx, tx, userID, &accountID, nil, amount, "withdraw", description)
	if err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit withdraw transaction: %w", err)
	}

	return account, transactionID, nil
}

func (r *AccountRepository) WithdrawWithMFA(
	ctx context.Context,
	userID int64,
	accountID int64,
	amount string,
	description string,
	mfaCodeID int64,
) (*models.Account, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("begin withdraw transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, 0, err
	}

	if err := markMFACodeUsedTx(ctx, tx, mfaCodeID); err != nil {
		return nil, 0, err
	}

	account, err := withdrawAccountBalance(ctx, tx, accountID, amount)
	if err != nil {
		return nil, 0, err
	}

	transactionID, err := createTransaction(ctx, tx, userID, &accountID, nil, amount, "withdraw", description)
	if err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit withdraw transaction: %w", err)
	}

	return account, transactionID, nil
}

func (r *AccountRepository) CardPayment(
	ctx context.Context,
	userID int64,
	accountID int64,
	amount string,
	description string,
) (*models.Account, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("begin card payment transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, 0, err
	}

	account, err := withdrawAccountBalance(ctx, tx, accountID, amount)
	if err != nil {
		return nil, 0, err
	}

	transactionID, err := createTransaction(ctx, tx, userID, &accountID, nil, amount, "card_payment", description)
	if err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit card payment transaction: %w", err)
	}

	return account, transactionID, nil
}

func (r *AccountRepository) CardPaymentWithMFA(
	ctx context.Context,
	userID int64,
	accountID int64,
	amount string,
	description string,
	mfaCodeID int64,
) (*models.Account, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("begin card payment transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockAccount(ctx, tx, accountID, userID); err != nil {
		return nil, 0, err
	}

	if err := markMFACodeUsedTx(ctx, tx, mfaCodeID); err != nil {
		return nil, 0, err
	}

	account, err := withdrawAccountBalance(ctx, tx, accountID, amount)
	if err != nil {
		return nil, 0, err
	}

	transactionID, err := createTransaction(ctx, tx, userID, &accountID, nil, amount, "card_payment", description)
	if err != nil {
		return nil, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit card payment transaction: %w", err)
	}

	return account, transactionID, nil
}

func (r *AccountRepository) Transfer(
	ctx context.Context,
	userID int64,
	fromAccountID int64,
	toAccountID int64,
	amount string,
	description string,
) (int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin transfer transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockTransferAccounts(ctx, tx, userID, fromAccountID, toAccountID); err != nil {
		return 0, err
	}

	if _, err := withdrawAccountBalance(ctx, tx, fromAccountID, amount); err != nil {
		return 0, err
	}

	if _, err := updateAccountBalance(ctx, tx, toAccountID, amount, "+"); err != nil {
		return 0, fmt.Errorf("transfer balance update: %w", err)
	}

	transactionID, err := createTransaction(
		ctx,
		tx,
		userID,
		&fromAccountID,
		&toAccountID,
		amount,
		"transfer",
		description,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transfer transaction: %w", err)
	}

	return transactionID, nil
}

func (r *AccountRepository) TransferWithMFA(
	ctx context.Context,
	userID int64,
	fromAccountID int64,
	toAccountID int64,
	amount string,
	description string,
	mfaCodeID int64,
) (int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin transfer transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := lockTransferAccounts(ctx, tx, userID, fromAccountID, toAccountID); err != nil {
		return 0, err
	}

	if err := markMFACodeUsedTx(ctx, tx, mfaCodeID); err != nil {
		return 0, err
	}

	if _, err := withdrawAccountBalance(ctx, tx, fromAccountID, amount); err != nil {
		return 0, err
	}

	if _, err := updateAccountBalance(ctx, tx, toAccountID, amount, "+"); err != nil {
		return 0, fmt.Errorf("transfer balance update: %w", err)
	}

	transactionID, err := createTransaction(
		ctx,
		tx,
		userID,
		&fromAccountID,
		&toAccountID,
		amount,
		"transfer",
		description,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transfer transaction: %w", err)
	}

	return transactionID, nil
}

func (r *AccountRepository) Close(ctx context.Context, userID int64, accountID int64) (*models.Account, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin close account transaction: %w", err)
	}
	defer rollbackTx(tx)

	account, balanceIsZero, err := lockAccountForClose(ctx, tx, accountID, userID)
	if err != nil {
		return nil, err
	}

	if account.IsClosed() {
		return nil, ErrAccountAlreadyClosed
	}

	if account.IsBlocked {
		return nil, ErrAccountBlocked
	}

	if !balanceIsZero {
		return nil, ErrAccountBalanceMustBeZero
	}

	hasBlockingCredit, err := accountHasBlockingCredit(ctx, tx, accountID)
	if err != nil {
		return nil, err
	}
	if hasBlockingCredit {
		return nil, ErrAccountHasActiveCredit
	}

	closedAccount, err := closeAccountRow(ctx, tx, accountID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit close account transaction: %w", err)
	}

	return closedAccount, nil
}

// lockAccount prevents concurrent balance changes and verifies ownership in the same database transaction.
// This prevents a user from modifying another user's balance even if they guess an account ID.
func lockAccount(ctx context.Context, tx *sql.Tx, accountID, userID int64) error {
	query := `
		SELECT id, is_blocked, status
		FROM accounts
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`

	var id int64
	var isBlocked bool
	var status string

	err := tx.QueryRowContext(ctx, query, accountID, userID).Scan(&id, &isBlocked, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}

		return fmt.Errorf("lock account: %w", err)
	}

	if status == models.AccountStatusClosed {
		return ErrAccountClosed
	}

	if isBlocked {
		return ErrAccountBlocked
	}

	return nil
}

// lockTransferAccounts locks accounts in deterministic order to reduce deadlock risk during transfers.
// Only the source account must belong to the authenticated user; the destination account may belong to another user.
func lockTransferAccounts(ctx context.Context, tx *sql.Tx, userID, fromAccountID, toAccountID int64) error {
	query := `
		SELECT id, user_id, is_blocked, status
		FROM accounts
		WHERE id IN ($1, $2)
		ORDER BY id
		FOR UPDATE
	`

	rows, err := tx.QueryContext(ctx, query, fromAccountID, toAccountID)
	if err != nil {
		return fmt.Errorf("lock transfer accounts: %w", err)
	}
	defer closeRows(rows)

	foundFrom := false
	foundTo := false
	fromOwnedByUser := false
	hasBlockedAccount := false
	hasClosedAccount := false

	for rows.Next() {
		var accountID int64
		var accountUserID int64
		var isBlocked bool
		var status string

		if err := rows.Scan(&accountID, &accountUserID, &isBlocked, &status); err != nil {
			return fmt.Errorf("scan locked account: %w", err)
		}

		if status == models.AccountStatusClosed {
			hasClosedAccount = true
		}

		if isBlocked {
			hasBlockedAccount = true
		}

		if accountID == fromAccountID {
			foundFrom = true
			fromOwnedByUser = accountUserID == userID
		}

		if accountID == toAccountID {
			foundTo = true
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate locked accounts: %w", err)
	}

	if !foundFrom || !foundTo || !fromOwnedByUser {
		return ErrAccountNotFound
	}

	if hasClosedAccount {
		return ErrAccountClosed
	}

	if hasBlockedAccount {
		return ErrAccountBlocked
	}

	return nil
}

func lockAccountForClose(ctx context.Context, tx *sql.Tx, accountID int64, userID int64) (*models.Account, bool, error) {
	query := `
		SELECT
			id,
			user_id,
			account_number,
			balance::text,
			currency,
			is_blocked,
			status,
			closed_at,
			created_at,
			balance = 0
		FROM accounts
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`

	account := &models.Account{}
	var balanceIsZero bool

	err := tx.QueryRowContext(ctx, query, accountID, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.Status,
		&account.ClosedAt,
		&account.CreatedAt,
		&balanceIsZero,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, ErrAccountNotFound
		}

		return nil, false, fmt.Errorf("lock account for close: %w", err)
	}

	return account, balanceIsZero, nil
}

func accountHasBlockingCredit(ctx context.Context, tx *sql.Tx, accountID int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM credits c
			WHERE c.account_id = $1
			  AND (
				c.status IN ('active', 'overdue')
				OR EXISTS (
					SELECT 1
					FROM payment_schedules ps
					WHERE ps.credit_id = c.id
					  AND ps.status IN ('pending', 'overdue')
				)
			  )
		)
	`

	var exists bool
	if err := tx.QueryRowContext(ctx, query, accountID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check account active credits: %w", err)
	}

	return exists, nil
}

func closeAccountRow(ctx context.Context, tx *sql.Tx, accountID int64) (*models.Account, error) {
	query := `
		UPDATE accounts
		SET status = 'closed', closed_at = NOW()
		WHERE id = $1 AND status = 'active'
		RETURNING id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
	`

	account := &models.Account{}
	err := tx.QueryRowContext(ctx, query, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.Status,
		&account.ClosedAt,
		&account.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("close account row: %w", err)
	}

	return account, nil
}

func updateAccountBalance(ctx context.Context, tx *sql.Tx, accountID int64, amount, operation string) (*models.Account, error) {
	var query string

	switch operation {
	case "+":
		query = `
			UPDATE accounts
			SET balance = balance + $1::numeric
			WHERE id = $2 AND status = 'active' AND is_blocked = FALSE
			RETURNING id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
		`
	case "-":
		query = `
			UPDATE accounts
			SET balance = balance - $1::numeric
			WHERE id = $2 AND status = 'active' AND is_blocked = FALSE
			RETURNING id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
		`
	default:
		return nil, fmt.Errorf("unsupported balance operation: %s", operation)
	}

	account := &models.Account{}

	err := tx.QueryRowContext(ctx, query, amount, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.Status,
		&account.ClosedAt,
		&account.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("update account balance: %w", err)
	}

	return account, nil
}

func withdrawAccountBalance(ctx context.Context, tx *sql.Tx, accountID int64, amount string) (*models.Account, error) {
	query := `
		UPDATE accounts
		SET balance = balance - $1::numeric
		WHERE id = $2 AND balance >= $1::numeric AND status = 'active' AND is_blocked = FALSE
		RETURNING id, user_id, account_number, balance::text, currency, is_blocked, status, closed_at, created_at
	`

	account := &models.Account{}

	err := tx.QueryRowContext(ctx, query, amount, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.Status,
		&account.ClosedAt,
		&account.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInsufficientFunds
		}

		return nil, fmt.Errorf("withdraw balance update: %w", err)
	}

	return account, nil
}

func createTransaction(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	fromAccountID *int64,
	toAccountID *int64,
	amount string,
	transactionType string,
	description string,
) (int64, error) {
	query := `
		INSERT INTO transactions (
			user_id,
			from_account_id,
			to_account_id,
			amount,
			type,
			status,
			description
		)
		VALUES ($1, $2, $3, $4::numeric, $5, 'completed', $6)
		RETURNING id
	`

	var transactionID int64

	err := tx.QueryRowContext(
		ctx,
		query,
		userID,
		fromAccountID,
		toAccountID,
		amount,
		transactionType,
		description,
	).Scan(&transactionID)
	if err != nil {
		return 0, fmt.Errorf("create transaction: %w", err)
	}

	return transactionID, nil
}
