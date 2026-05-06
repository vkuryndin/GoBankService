package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrAccountStatusNotChanged = errors.New("account status not changed")

type AdminUser struct {
	ID                   int64
	Email                string
	Username             string
	IsAdmin              bool
	AccountsCount        int
	BlockedAccountsCount int
	CreatedAt            time.Time
}

type AdminActiveSession struct {
	SessionID int64
	UserID    int64
	Email     string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AdminAccountStatus struct {
	ID            int64
	UserID        int64
	AccountNumber string
	IsBlocked     bool
}

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{
		db: db,
	}
}

func (r *AdminRepository) FindUsers(ctx context.Context) ([]AdminUser, error) {
	query := `
		SELECT
			u.id,
			u.email,
			u.username,
			u.is_admin,
			COUNT(a.id)::int AS accounts_count,
			COALESCE(SUM(CASE WHEN a.is_blocked THEN 1 ELSE 0 END), 0)::int AS blocked_accounts_count,
			u.created_at
		FROM users u
		LEFT JOIN accounts a ON a.user_id = u.id
		GROUP BY u.id, u.email, u.username, u.is_admin, u.created_at
		ORDER BY u.id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find admin users: %w", err)
	}
	defer rows.Close()

	users := make([]AdminUser, 0)

	for rows.Next() {
		var user AdminUser

		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.IsAdmin,
			&user.AccountsCount,
			&user.BlockedAccountsCount,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan admin user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin users: %w", err)
	}

	return users, nil
}

func (r *AdminRepository) FindActiveSessions(ctx context.Context) ([]AdminActiveSession, error) {
	query := `
		SELECT
			s.id,
			u.id,
			u.email,
			u.username,
			s.created_at,
			s.expires_at
		FROM user_sessions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		ORDER BY s.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find active sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]AdminActiveSession, 0)

	for rows.Next() {
		var session AdminActiveSession

		if err := rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.Email,
			&session.Username,
			&session.CreatedAt,
			&session.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan active session: %w", err)
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active sessions: %w", err)
	}

	return sessions, nil
}

func (r *AdminRepository) SetAccountBlocked(
	ctx context.Context,
	accountID int64,
	blocked bool,
) (*AdminAccountStatus, error) {
	query := `
		UPDATE accounts
		SET is_blocked = $2
		WHERE id = $1
		  AND is_blocked <> $2
		RETURNING id, user_id, account_number, is_blocked
	`

	account := &AdminAccountStatus{}

	err := r.db.QueryRowContext(ctx, query, accountID, blocked).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.IsBlocked,
	)
	if err == nil {
		return account, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("set account blocked: %w", err)
	}

	currentStatus, err := r.findAccountBlockedStatus(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if currentStatus == blocked {
		return nil, ErrAccountStatusNotChanged
	}

	return nil, ErrAccountNotFound
}

func (r *AdminRepository) findAccountBlockedStatus(ctx context.Context, accountID int64) (bool, error) {
	query := `
		SELECT is_blocked
		FROM accounts
		WHERE id = $1
	`

	var isBlocked bool

	err := r.db.QueryRowContext(ctx, query, accountID).Scan(&isBlocked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrAccountNotFound
		}

		return false, fmt.Errorf("find account blocked status: %w", err)
	}

	return isBlocked, nil
}
