package dto

type CreateCardRequest struct {
	AccountID int64 `json:"account_id"`
}

type CardResponse struct {
	ID           int64  `json:"id"`
	AccountID    int64  `json:"account_id"`
	Number       string `json:"number,omitempty"`
	MaskedNumber string `json:"masked_number"`
	Expiry       string `json:"expiry,omitempty"`
	CVV          string `json:"cvv,omitempty"`
	NumberHMAC   string `json:"number_hmac,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type CardPaymentRequest struct {
	Amount      string `json:"amount"`
	CVV         string `json:"cvv"`
	MFACode     string `json:"mfa_code"`
	Description string `json:"description"`
}

type CardPaymentResponse struct {
	TransactionID int64  `json:"transaction_id"`
	CardID        int64  `json:"card_id"`
	AccountID     int64  `json:"account_id"`
	Amount        string `json:"amount"`
	Status        string `json:"status"`
}
