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
