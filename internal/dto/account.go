package dto

// AccountResponse is returned to the account owner. Currency is fixed to RUB by the service/database rules.
type AccountResponse struct {
	ID            int64  `json:"id"`
	AccountNumber string `json:"account_number"`
	Balance       string `json:"balance"`
	Currency      string `json:"currency"`
	IsBlocked     bool   `json:"is_blocked"`
	Status        string `json:"status"`
	ClosedAt      string `json:"closed_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type CloseAccountResponse struct {
	ID            int64  `json:"id"`
	AccountNumber string `json:"account_number"`
	Status        string `json:"status"`
	ClosedAt      string `json:"closed_at"`
	Message       string `json:"message"`
}

type DepositRequest struct {
	Amount string `json:"amount"`
}

type WithdrawRequest struct {
	Amount  string `json:"amount"`
	MFACode string `json:"mfa_code"`
}

type PredictBalanceResponse struct {
	AccountID               int64  `json:"account_id"`
	Days                    int    `json:"days"`
	CurrentBalance          string `json:"current_balance"`
	ExpectedIncome          string `json:"expected_income"`
	ExpectedExpense         string `json:"expected_expense"`
	ScheduledCreditPayments string `json:"scheduled_credit_payments"`
	PredictedBalance        string `json:"predicted_balance"`
	AverageDailyIncome      string `json:"average_daily_income"`
	AverageDailyExpense     string `json:"average_daily_expense"`
	StatisticsPeriodDays    int    `json:"statistics_period_days"`
}
