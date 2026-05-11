package repositories

import (
	"database/sql"
)

func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}

func rollbackTx(tx *sql.Tx) {
	_ = tx.Rollback()
}
