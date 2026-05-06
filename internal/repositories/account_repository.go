package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"bank-service/internal/models"
)

var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountBlocked    = errors.New("account is blocked")
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
		RETURNING id, user_id, account_number, balance::text, currency, is_blocked, created_at
	`

	account := &models.Account{}

	err := r.db.QueryRowContext(ctx, query, userID, accountNumber).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

func (r *AccountRepository) FindByUserID(ctx context.Context, userID int64) ([]models.Account, error) {
	query := `
		SELECT id, user_id, account_number, balance::text, currency, is_blocked, created_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("find accounts by user id: %w", err)
	}
	defer rows.Close()

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
		SELECT id, user_id, account_number, balance::text, currency, is_blocked, created_at
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
			)
	`

	var fromExists bool
	var toExists bool
	var fromBlocked bool
	var toBlocked bool

	err := r.db.QueryRowContext(ctx, query, fromAccountID, userID, toAccountID).Scan(
		&fromExists,
		&toExists,
		&fromBlocked,
		&toBlocked,
	)
	if err != nil {
		return fmt.Errorf("validate transfer accounts: %w", err)
	}

	if !fromExists || !toExists {
		return ErrAccountNotFound
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
	defer tx.Rollback()

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
	defer tx.Rollback()

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
	defer tx.Rollback()

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
	defer tx.Rollback()

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

func lockAccount(ctx context.Context, tx *sql.Tx, accountID, userID int64) error {
	query := `
		SELECT id, is_blocked
		FROM accounts
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`

	var id int64
	var isBlocked bool

	err := tx.QueryRowContext(ctx, query, accountID, userID).Scan(&id, &isBlocked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}

		return fmt.Errorf("lock account: %w", err)
	}

	if isBlocked {
		return ErrAccountBlocked
	}

	return nil
}

func lockTransferAccounts(ctx context.Context, tx *sql.Tx, userID, fromAccountID, toAccountID int64) error {
	query := `
		SELECT id, user_id, is_blocked
		FROM accounts
		WHERE id IN ($1, $2)
		ORDER BY id
		FOR UPDATE
	`

	rows, err := tx.QueryContext(ctx, query, fromAccountID, toAccountID)
	if err != nil {
		return fmt.Errorf("lock transfer accounts: %w", err)
	}
	defer rows.Close()

	foundFrom := false
	foundTo := false
	fromOwnedByUser := false
	hasBlockedAccount := false

	for rows.Next() {
		var accountID int64
		var accountUserID int64
		var isBlocked bool

		if err := rows.Scan(&accountID, &accountUserID, &isBlocked); err != nil {
			return fmt.Errorf("scan locked account: %w", err)
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

	if hasBlockedAccount {
		return ErrAccountBlocked
	}

	return nil
}

func updateAccountBalance(ctx context.Context, tx *sql.Tx, accountID int64, amount, operation string) (*models.Account, error) {
	query := fmt.Sprintf(`
		UPDATE accounts
		SET balance = balance %s $1::numeric
		WHERE id = $2
		RETURNING id, user_id, account_number, balance::text, currency, is_blocked, created_at
	`, operation)

	account := &models.Account{}

	err := tx.QueryRowContext(ctx, query, amount, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
		&account.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func withdrawAccountBalance(ctx context.Context, tx *sql.Tx, accountID int64, amount string) (*models.Account, error) {
	query := `
		UPDATE accounts
		SET balance = balance - $1::numeric
		WHERE id = $2 AND balance >= $1::numeric
		RETURNING id, user_id, account_number, balance::text, currency, is_blocked, created_at
	`

	account := &models.Account{}

	err := tx.QueryRowContext(ctx, query, amount, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.IsBlocked,
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
