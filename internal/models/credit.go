package models

import "time"

type Credit struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	AccountID       int64     `json:"account_id"`
	PrincipalAmount string    `json:"principal_amount"`
	InterestRate    string    `json:"interest_rate"`
	TermMonths      int       `json:"term_months"`
	MonthlyPayment  string    `json:"monthly_payment"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}
