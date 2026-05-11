package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrMFACodeNotFound = errors.New("mfa code not found")

type MFACode struct {
	ID       int64
	CodeHash string
}

type MFARepository struct {
	db *sql.DB
}

func NewMFARepository(db *sql.DB) *MFARepository {
	return &MFARepository{
		db: db,
	}
}

func (r *MFARepository) SaveCode(
	ctx context.Context,
	userID int64,
	purpose string,
	operationHash string,
	codeHash string,
	expiresAt time.Time,
) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin mfa transaction: %w", err)
	}
	defer rollbackTx(tx)

	invalidateQuery := `
		UPDATE mfa_codes
		SET used_at = NOW()
		WHERE user_id = $1
		  AND purpose = $2
		  AND used_at IS NULL
		  AND expires_at > NOW()
	`

	if _, err := tx.ExecContext(ctx, invalidateQuery, userID, purpose); err != nil {
		return fmt.Errorf("invalidate previous mfa codes: %w", err)
	}

	insertQuery := `
		INSERT INTO mfa_codes (
			user_id,
			purpose,
			operation_hash,
			code_hash,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5)
	`

	if _, err := tx.ExecContext(ctx, insertQuery, userID, purpose, operationHash, codeHash, expiresAt); err != nil {
		return fmt.Errorf("save mfa code: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit mfa transaction: %w", err)
	}

	return nil
}

func (r *MFARepository) FindActiveCode(
	ctx context.Context,
	userID int64,
	purpose string,
	operationHash string,
) (*MFACode, error) {
	query := `
		SELECT id, code_hash
		FROM mfa_codes
		WHERE user_id = $1
		  AND purpose = $2
		  AND operation_hash = $3
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	code := &MFACode{}

	err := r.db.QueryRowContext(ctx, query, userID, purpose, operationHash).Scan(
		&code.ID,
		&code.CodeHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMFACodeNotFound
		}

		return nil, fmt.Errorf("find active mfa code: %w", err)
	}

	return code, nil
}

func (r *MFARepository) MarkUsed(ctx context.Context, codeID int64) error {
	query := `
		UPDATE mfa_codes
		SET used_at = NOW()
		WHERE id = $1
	`

	if _, err := r.db.ExecContext(ctx, query, codeID); err != nil {
		return fmt.Errorf("mark mfa code used: %w", err)
	}

	return nil
}

func (r *MFARepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM mfa_codes
		WHERE expires_at <= NOW()
		   OR used_at IS NOT NULL
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired mfa codes: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get deleted mfa codes count: %w", err)
	}

	return count, nil
}
