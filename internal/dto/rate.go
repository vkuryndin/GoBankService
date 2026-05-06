package dto

type KeyRateResponse struct {
	KeyRate    string `json:"key_rate"`
	BankRate   string `json:"bank_rate"`
	BankMargin string `json:"bank_margin"`
	Date       string `json:"date,omitempty"`
	Source     string `json:"source"`
}
