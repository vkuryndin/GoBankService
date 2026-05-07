package models

import (
	"database/sql"
	"time"
)

const (
	CardStatusActive = "active"
	CardStatusClosed = "closed"
)

type Card struct {
	ID              int64        `json:"id"`
	UserID          int64        `json:"user_id"`
	AccountID       int64        `json:"account_id"`
	EncryptedNumber []byte       `json:"-"`
	EncryptedExpiry []byte       `json:"-"`
	CVVHash         string       `json:"-"`
	NumberHMAC      string       `json:"number_hmac"`
	Status          string       `json:"status"`
	ClosedAt        sql.NullTime `json:"-"`
	CreatedAt       time.Time    `json:"created_at"`
}

type CardDetails struct {
	ID         int64
	UserID     int64
	AccountID  int64
	Number     string
	Expiry     string
	CVVHash    string
	CVV        string
	NumberHMAC string
	Status     string
	ClosedAt   sql.NullTime
	CreatedAt  time.Time
}
