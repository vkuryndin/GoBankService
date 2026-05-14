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
	Status       string `json:"status"`
	ClosedAt     string `json:"closed_at,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type CardRevealRequest struct {
	MFACode string `json:"mfa_code"`
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

type CloseCardResponse struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"account_id"`
	Status    string `json:"status"`
	ClosedAt  string `json:"closed_at"`
}

type CardTransferRequest struct {
	ToCardID    int64  `json:"to_card_id"`
	Amount      string `json:"amount"`
	CVV         string `json:"cvv"`
	MFACode     string `json:"mfa_code"`
	Description string `json:"description"`
}

type CardTransferResponse struct {
	TransactionID int64  `json:"transaction_id"`
	FromCardID    int64  `json:"from_card_id"`
	ToCardID      int64  `json:"to_card_id"`
	FromAccountID int64  `json:"from_account_id"`
	ToAccountID   int64  `json:"to_account_id"`
	Amount        string `json:"amount"`
	Status        string `json:"status"`
}
