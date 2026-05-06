package models

import (
	"database/sql"
	"time"
)

type Transaction struct {
	ID            int64          `json:"id"`
	UserID        int64          `json:"user_id"`
	FromAccountID sql.NullInt64  `json:"-"`
	ToAccountID   sql.NullInt64  `json:"-"`
	Amount        string         `json:"amount"`
	Currency      string         `json:"currency"`
	Type          string         `json:"type"`
	Status        string         `json:"status"`
	Description   sql.NullString `json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
}
