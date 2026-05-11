package models

import (
	"database/sql"
	"time"
)

const (
	AccountStatusActive = "active"
	AccountStatusClosed = "closed"
)

type Account struct {
	ID            int64
	UserID        int64
	AccountNumber string
	Balance       string
	Currency      string
	IsBlocked     bool
	Status        string
	ClosedAt      sql.NullTime
	CreatedAt     time.Time
}

func (a *Account) IsClosed() bool {
	return a != nil && a.Status == AccountStatusClosed
}
