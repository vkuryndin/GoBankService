package dto

type TransferRequest struct {
	FromAccountID        int64  `json:"from_account_id"`
	ToAccountID          int64  `json:"to_account_id,omitempty"`
	ToAccountNumber      string `json:"to_account_number,omitempty"`
	ToAccountNumberCamel string `json:"toAccountNumber,omitempty"`
	Amount               string `json:"amount"`
	Description          string `json:"description"`
	MFACode              string `json:"mfa_code"`
}

func (r TransferRequest) RecipientAccountNumber() string {
	if r.ToAccountNumber != "" {
		return r.ToAccountNumber
	}

	return r.ToAccountNumberCamel
}

type TransferResponse struct {
	TransactionID int64  `json:"transaction_id"`
	FromAccountID int64  `json:"from_account_id"`
	ToAccountID   int64  `json:"to_account_id"`
	Amount        string `json:"amount"`
	Status        string `json:"status"`
}
