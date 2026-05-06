package models

import "time"

type Account struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	AccountNumber string    `json:"account_number"`
	Balance       string    `json:"balance"`
	Currency      string    `json:"currency"`
	IsBlocked     bool      `json:"is_blocked"`
	CreatedAt     time.Time `json:"created_at"`
}
