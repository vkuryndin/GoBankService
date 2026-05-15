package dto

type MFARequest struct {
	Purpose           string `json:"purpose"`
	FromAccountID     int64  `json:"from_account_id,omitempty"`
	ToAccountID       int64  `json:"to_account_id,omitempty"`
	CardID            int64  `json:"card_id,omitempty"`
	ToCardID          int64  `json:"to_card_id,omitempty"`
	ToCardIDCamel     int64  `json:"toCardId,omitempty"`
	ToCardNumber      string `json:"to_card_number,omitempty"`
	ToCardNumberCamel string `json:"toCardNumber,omitempty"`
	AccountID         int64  `json:"account_id,omitempty"`
	Amount            string `json:"amount,omitempty"`
	PrincipalAmount   string `json:"principal_amount,omitempty"`
	TermMonths        int    `json:"term_months,omitempty"`
	CreditID          int64  `json:"credit_id,omitempty"`
	PrepaymentMode    string `json:"prepayment_mode,omitempty"`
	Mode              string `json:"mode,omitempty"`
}

func (r MFARequest) RecipientCardID() int64 {
	if r.ToCardID > 0 {
		return r.ToCardID
	}

	return r.ToCardIDCamel
}

func (r MFARequest) RecipientCardNumber() string {
	if r.ToCardNumber != "" {
		return r.ToCardNumber
	}

	return r.ToCardNumberCamel
}

func (r MFARequest) CreditPrepaymentMode() string {
	if r.PrepaymentMode != "" {
		return r.PrepaymentMode
	}

	return r.Mode
}
