package dto

type CreateCreditRequest struct {
	AccountID       int64  `json:"account_id"`
	PrincipalAmount string `json:"principal_amount"`
	TermMonths      int    `json:"term_months"`
	MFACode         string `json:"mfa_code"`
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
