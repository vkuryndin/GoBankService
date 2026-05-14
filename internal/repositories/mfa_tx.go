package repositories

import (
	"context"
	"database/sql"
	"fmt"
)

func markMFACodeUsedTx(ctx context.Context, tx *sql.Tx, codeID int64) error {
	query := `
		UPDATE mfa_codes
		SET used_at = NOW()
		WHERE id = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
	`

	result, err := tx.ExecContext(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("mark mfa code used: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get marked mfa code count: %w", err)
	}

	if affected == 0 {
		return ErrMFACodeNotFound
	}

	return nil
}
