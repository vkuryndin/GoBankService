package dto

type AccountResponse struct {
	ID            int64  `json:"id"`
	AccountNumber string `json:"account_number"`
	Balance       string `json:"balance"`
	Currency      string `json:"currency"`
	IsBlocked     bool   `json:"is_blocked"`
	CreatedAt     string `json:"created_at"`
}

type DepositRequest struct {
	Amount string `json:"amount"`
}

type WithdrawRequest struct {
	Amount string `json:"amount"`
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
