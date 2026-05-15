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

type AdminSystemStatistics struct {
	GeneratedAt  time.Time
	Users        AdminUsersStatistics
	Accounts     AdminAccountsStatistics
	Cards        AdminCardsStatistics
	Credits      AdminCreditsStatistics
	Transactions AdminTransactionsStatistics
	Audit        AdminAuditStatistics
}

type AdminUsersStatistics struct {
	Total          int64
	Admins         int64
	RegularUsers   int64
	NewLast24h     int64
	ActiveSessions int64
}

type AdminAccountsStatistics struct {
	Total        int64
	Active       int64
	Closed       int64
	Blocked      int64
	TotalBalance string
	Currency     string
}

type AdminCardsStatistics struct {
	Total  int64
	Active int64
	Closed int64
}

type AdminCreditsStatistics struct {
	Total                 int64
	Active                int64
	Closed                int64
	Overdue               int64
	ActivePrincipalAmount string
	ActiveMonthlyPayment  string
	Currency              string
}

type AdminTransactionsStatistics struct {
	Total              int64
	Completed          int64
	Failed             int64
	Last24h            int64
	CompletedAmount    string
	CompletedThisMonth string
	Currency           string
	ByType             []AdminTransactionTypeStatistics
	Recent             []AdminRecentTransaction
}

type AdminTransactionTypeStatistics struct {
	Type        string
	Count       int64
	TotalAmount string
}

type AdminRecentTransaction struct {
	ID          int64
	UserID      int64
	Type        string
	Status      string
	Amount      string
	Currency    string
	Description sql.NullString
	CreatedAt   time.Time
}

type AdminAuditStatistics struct {
	Total   int64
	Success int64
	Failed  int64
	Blocked int64
	Recent  []AdminRecentAudit
}

type AdminRecentAudit struct {
	ID           int64
	UserID       sql.NullInt64
	Action       string
	ResourceType sql.NullString
	ResourceID   sql.NullInt64
	Status       string
	CreatedAt    time.Time
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
	defer closeRows(rows)

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
	defer closeRows(rows)

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

func (r *AdminRepository) GetSystemStatistics(ctx context.Context) (*AdminSystemStatistics, error) {
	statistics := &AdminSystemStatistics{
		GeneratedAt: time.Now().UTC(),
	}

	if err := r.fillUserStatistics(ctx, &statistics.Users); err != nil {
		return nil, err
	}

	if err := r.fillAccountStatistics(ctx, &statistics.Accounts); err != nil {
		return nil, err
	}

	if err := r.fillCardStatistics(ctx, &statistics.Cards); err != nil {
		return nil, err
	}

	if err := r.fillCreditStatistics(ctx, &statistics.Credits); err != nil {
		return nil, err
	}

	if err := r.fillTransactionStatistics(ctx, &statistics.Transactions); err != nil {
		return nil, err
	}

	if err := r.fillAuditStatistics(ctx, &statistics.Audit); err != nil {
		return nil, err
	}

	return statistics, nil
}

func (r *AdminRepository) fillUserStatistics(ctx context.Context, statistics *AdminUsersStatistics) error {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(CASE WHEN is_admin THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN created_at >= NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END), 0)::bigint
		FROM users
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(
		&statistics.Total,
		&statistics.Admins,
		&statistics.NewLast24h,
	); err != nil {
		return fmt.Errorf("get admin user statistics: %w", err)
	}

	statistics.RegularUsers = statistics.Total - statistics.Admins

	activeSessionsQuery := `
		SELECT COUNT(*)::bigint
		FROM user_sessions
		WHERE revoked_at IS NULL
		  AND expires_at > NOW()
	`
	if err := r.db.QueryRowContext(ctx, activeSessionsQuery).Scan(&statistics.ActiveSessions); err != nil {
		return fmt.Errorf("get admin active session statistics: %w", err)
	}

	return nil
}

func (r *AdminRepository) fillAccountStatistics(ctx context.Context, statistics *AdminAccountsStatistics) error {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN is_blocked THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(balance), 0)::text
		FROM accounts
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(
		&statistics.Total,
		&statistics.Active,
		&statistics.Closed,
		&statistics.Blocked,
		&statistics.TotalBalance,
	); err != nil {
		return fmt.Errorf("get admin account statistics: %w", err)
	}

	statistics.Currency = "RUB"
	return nil
}

func (r *AdminRepository) fillCardStatistics(ctx context.Context, statistics *AdminCardsStatistics) error {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END), 0)::bigint
		FROM cards
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(
		&statistics.Total,
		&statistics.Active,
		&statistics.Closed,
	); err != nil {
		return fmt.Errorf("get admin card statistics: %w", err)
	}

	return nil
}

func (r *AdminRepository) fillCreditStatistics(ctx context.Context, statistics *AdminCreditsStatistics) error {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'overdue' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status IN ('active', 'overdue') THEN principal_amount ELSE 0 END), 0)::text,
			COALESCE(SUM(CASE WHEN status IN ('active', 'overdue') THEN monthly_payment ELSE 0 END), 0)::text
		FROM credits
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(
		&statistics.Total,
		&statistics.Active,
		&statistics.Closed,
		&statistics.Overdue,
		&statistics.ActivePrincipalAmount,
		&statistics.ActiveMonthlyPayment,
	); err != nil {
		return fmt.Errorf("get admin credit statistics: %w", err)
	}

	statistics.Currency = "RUB"
	return nil
}

func (r *AdminRepository) fillTransactionStatistics(ctx context.Context, statistics *AdminTransactionsStatistics) error {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN created_at >= NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END), 0)::text,
			COALESCE(SUM(CASE WHEN status = 'completed' AND created_at >= date_trunc('month', NOW()) THEN amount ELSE 0 END), 0)::text
		FROM transactions
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(
		&statistics.Total,
		&statistics.Completed,
		&statistics.Failed,
		&statistics.Last24h,
		&statistics.CompletedAmount,
		&statistics.CompletedThisMonth,
	); err != nil {
		return fmt.Errorf("get admin transaction statistics: %w", err)
	}

	statistics.Currency = "RUB"

	byType, err := r.findTransactionStatisticsByType(ctx)
	if err != nil {
		return err
	}
	statistics.ByType = byType

	recent, err := r.findRecentTransactions(ctx)
	if err != nil {
		return err
	}
	statistics.Recent = recent

	return nil
}

func (r *AdminRepository) findTransactionStatisticsByType(ctx context.Context) ([]AdminTransactionTypeStatistics, error) {
	query := `
		SELECT type, COUNT(*)::bigint, COALESCE(SUM(amount), 0)::text
		FROM transactions
		WHERE status = 'completed'
		GROUP BY type
		ORDER BY type
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find admin transaction type statistics: %w", err)
	}
	defer closeRows(rows)

	items := make([]AdminTransactionTypeStatistics, 0)
	for rows.Next() {
		var item AdminTransactionTypeStatistics
		if err := rows.Scan(&item.Type, &item.Count, &item.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan admin transaction type statistics: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin transaction type statistics: %w", err)
	}

	return items, nil
}

func (r *AdminRepository) findRecentTransactions(ctx context.Context) ([]AdminRecentTransaction, error) {
	query := `
		SELECT id, user_id, type, status, amount::text, currency, description, created_at
		FROM transactions
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find recent admin transactions: %w", err)
	}
	defer closeRows(rows)

	items := make([]AdminRecentTransaction, 0)
	for rows.Next() {
		var item AdminRecentTransaction
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Type,
			&item.Status,
			&item.Amount,
			&item.Currency,
			&item.Description,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent admin transaction: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent admin transactions: %w", err)
	}

	return items, nil
}

func (r *AdminRepository) fillAuditStatistics(ctx context.Context, statistics *AdminAuditStatistics) error {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)::bigint,
			COALESCE(SUM(CASE WHEN status = 'blocked' THEN 1 ELSE 0 END), 0)::bigint
		FROM audit_logs
	`

	if err := r.db.QueryRowContext(ctx, query).Scan(
		&statistics.Total,
		&statistics.Success,
		&statistics.Failed,
		&statistics.Blocked,
	); err != nil {
		return fmt.Errorf("get admin audit statistics: %w", err)
	}

	recent, err := r.findRecentAuditEvents(ctx)
	if err != nil {
		return err
	}
	statistics.Recent = recent

	return nil
}

func (r *AdminRepository) findRecentAuditEvents(ctx context.Context) ([]AdminRecentAudit, error) {
	query := `
		SELECT id, user_id, action, resource_type, resource_id, status, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find recent admin audit events: %w", err)
	}
	defer closeRows(rows)

	items := make([]AdminRecentAudit, 0)
	for rows.Next() {
		var item AdminRecentAudit
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Action,
			&item.ResourceType,
			&item.ResourceID,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent admin audit event: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent admin audit events: %w", err)
	}

	return items, nil
}
