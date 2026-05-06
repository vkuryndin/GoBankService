package models

import (
	"database/sql"
	"time"
)

type PaymentSchedule struct {
	ID            int64        `json:"id"`
	CreditID      int64        `json:"credit_id"`
	PaymentDate   time.Time    `json:"payment_date"`
	Amount        string       `json:"amount"`
	PenaltyAmount string       `json:"penalty_amount"`
	Status        string       `json:"status"`
	PaidAt        sql.NullTime `json:"-"`
	CreatedAt     time.Time    `json:"created_at"`
}
