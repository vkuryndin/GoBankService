package dto

type CreateCreditRequest struct {
	AccountID       int64  `json:"account_id"`
	PrincipalAmount string `json:"principal_amount"`
	TermMonths      int    `json:"term_months"`
	MFACode         string `json:"mfa_code"`
}

type CheckCreditRequest struct {
	AccountID       int64  `json:"account_id"`
	PrincipalAmount string `json:"principal_amount"`
	TermMonths      int    `json:"term_months"`
}

type CreditPrepaymentRequest struct {
	Amount  string `json:"amount"`
	Mode    string `json:"mode"`
	MFACode string `json:"mfa_code"`
}

type CreditPrepaymentResponse struct {
	TransactionID     int64          `json:"transaction_id"`
	Credit            CreditResponse `json:"credit"`
	Amount            string         `json:"amount"`
	Mode              string         `json:"mode"`
	OldMonthlyPayment string         `json:"old_monthly_payment"`
	NewMonthlyPayment string         `json:"new_monthly_payment"`
	OldTermMonths     int            `json:"old_term_months"`
	NewTermMonths     int            `json:"new_term_months"`
	RemainingDebt     string         `json:"remaining_debt"`
	Closed            bool           `json:"closed"`
}

type CreditCheckResponse struct {
	Eligible                  bool     `json:"eligible"`
	Reason                    string   `json:"reason,omitempty"`
	Reasons                   []string `json:"reasons,omitempty"`
	PolicyEnabled             bool     `json:"policy_enabled"`
	AccountID                 int64    `json:"account_id"`
	PrincipalAmount           string   `json:"principal_amount"`
	MaxPrincipalAmount        string   `json:"max_principal_amount"`
	TermMonths                int      `json:"term_months"`
	InterestRate              string   `json:"interest_rate"`
	MonthlyPayment            string   `json:"monthly_payment"`
	ActiveCreditsCount        int      `json:"active_credits_count"`
	MaxActiveCredits          int      `json:"max_active_credits"`
	HasOverdueCredit          bool     `json:"has_overdue_credit"`
	TotalPrincipalAmount      string   `json:"total_principal_amount"`
	MaxTotalPrincipalAmount   string   `json:"max_total_principal_amount"`
	MonthlyIncome             string   `json:"monthly_income"`
	CurrentMonthlyPayments    string   `json:"current_monthly_payments"`
	RequestedMonthlyPayment   string   `json:"requested_monthly_payment"`
	TotalMonthlyPayments      string   `json:"total_monthly_payments"`
	MaxAllowedMonthlyPayments string   `json:"max_allowed_monthly_payments"`
	DebtLoadPercent           string   `json:"debt_load_percent"`
	MaxDebtLoadPercent        int      `json:"max_debt_load_percent"`
}

type CreditResponse struct {
	ID              int64  `json:"id"`
	AccountID       int64  `json:"account_id"`
	PrincipalAmount string `json:"principal_amount"`
	InterestRate    string `json:"interest_rate"`
	TermMonths      int    `json:"term_months"`
	MonthlyPayment  string `json:"monthly_payment"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
}

type PaymentScheduleResponse struct {
	ID            int64  `json:"id"`
	CreditID      int64  `json:"credit_id"`
	PaymentDate   string `json:"payment_date"`
	Amount        string `json:"amount"`
	PenaltyAmount string `json:"penalty_amount"`
	Status        string `json:"status"`
	PaidAt        string `json:"paid_at,omitempty"`
}

type CreditOperationResponse struct {
	Source        string `json:"source"`
	EventType     string `json:"event_type"`
	TransactionID *int64 `json:"transaction_id,omitempty"`
	ScheduleID    *int64 `json:"schedule_id,omitempty"`
	Amount        string `json:"amount,omitempty"`
	PenaltyAmount string `json:"penalty_amount,omitempty"`
	Status        string `json:"status,omitempty"`
	Description   string `json:"description,omitempty"`
	PaymentDate   string `json:"payment_date,omitempty"`
	OccurredAt    string `json:"occurred_at,omitempty"`
}
