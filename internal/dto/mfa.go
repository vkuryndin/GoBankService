package dto

type MFARequest struct {
	Purpose         string `json:"purpose"`
	FromAccountID   int64  `json:"from_account_id,omitempty"`
	ToAccountID     int64  `json:"to_account_id,omitempty"`
	CardID          int64  `json:"card_id,omitempty"`
	ToCardID        int64  `json:"to_card_id,omitempty"`
	AccountID       int64  `json:"account_id,omitempty"`
	Amount          string `json:"amount,omitempty"`
	PrincipalAmount string `json:"principal_amount,omitempty"`
	TermMonths      int    `json:"term_months,omitempty"`
}
