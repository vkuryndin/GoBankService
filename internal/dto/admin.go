package dto

type AdminUserResponse struct {
	ID                   int64  `json:"id"`
	Email                string `json:"email"`
	Username             string `json:"username"`
	IsAdmin              bool   `json:"is_admin"`
	AccountsCount        int    `json:"accounts_count"`
	BlockedAccountsCount int    `json:"blocked_accounts_count"`
	CreatedAt            string `json:"created_at"`
}

type AdminActiveSessionResponse struct {
	SessionID int64  `json:"session_id"`
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type AdminAccountStatusResponse struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	AccountNumber string `json:"account_number"`
	IsBlocked     bool   `json:"is_blocked"`
	Message       string `json:"message"`
}

type AdminSystemStatisticsResponse struct {
	GeneratedAt  string                              `json:"generated_at"`
	Users        AdminUsersStatisticsResponse        `json:"users"`
	Accounts     AdminAccountsStatisticsResponse     `json:"accounts"`
	Cards        AdminCardsStatisticsResponse        `json:"cards"`
	Credits      AdminCreditsStatisticsResponse      `json:"credits"`
	Transactions AdminTransactionsStatisticsResponse `json:"transactions"`
	Audit        AdminAuditStatisticsResponse        `json:"audit"`
}

type AdminUsersStatisticsResponse struct {
	Total          int64 `json:"total"`
	Admins         int64 `json:"admins"`
	RegularUsers   int64 `json:"regular_users"`
	NewLast24h     int64 `json:"new_last_24h"`
	ActiveSessions int64 `json:"active_sessions"`
}

type AdminAccountsStatisticsResponse struct {
	Total        int64  `json:"total"`
	Active       int64  `json:"active"`
	Closed       int64  `json:"closed"`
	Blocked      int64  `json:"blocked"`
	TotalBalance string `json:"total_balance"`
	Currency     string `json:"currency"`
}

type AdminCardsStatisticsResponse struct {
	Total  int64 `json:"total"`
	Active int64 `json:"active"`
	Closed int64 `json:"closed"`
}

type AdminCreditsStatisticsResponse struct {
	Total                 int64  `json:"total"`
	Active                int64  `json:"active"`
	Closed                int64  `json:"closed"`
	Overdue               int64  `json:"overdue"`
	ActivePrincipalAmount string `json:"active_principal_amount"`
	ActiveMonthlyPayment  string `json:"active_monthly_payment"`
	Currency              string `json:"currency"`
}

type AdminTransactionsStatisticsResponse struct {
	Total              int64                            `json:"total"`
	Completed          int64                            `json:"completed"`
	Failed             int64                            `json:"failed"`
	Last24h            int64                            `json:"last_24h"`
	CompletedAmount    string                           `json:"completed_amount"`
	CompletedThisMonth string                           `json:"completed_this_month"`
	Currency           string                           `json:"currency"`
	ByType             []AdminTransactionTypeResponse   `json:"by_type"`
	Recent             []AdminRecentTransactionResponse `json:"recent"`
}

type AdminTransactionTypeResponse struct {
	Type        string `json:"type"`
	Count       int64  `json:"count"`
	TotalAmount string `json:"total_amount"`
}

type AdminRecentTransactionResponse struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type AdminAuditStatisticsResponse struct {
	Total   int64                      `json:"total"`
	Success int64                      `json:"success"`
	Failed  int64                      `json:"failed"`
	Blocked int64                      `json:"blocked"`
	Recent  []AdminRecentAuditResponse `json:"recent"`
}

type AdminRecentAuditResponse struct {
	ID           int64  `json:"id"`
	UserID       *int64 `json:"user_id,omitempty"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   *int64 `json:"resource_id,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}
