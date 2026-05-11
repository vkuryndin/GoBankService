package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TokenRepository struct {
	db *sql.DB
}

func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{
		db: db,
	}
}

func (r *TokenRepository) SaveRevokedToken(
	ctx context.Context,
	tokenHash string,
	userID int64,
	expiresAt time.Time,
) error {
	query := `
		INSERT INTO revoked_tokens (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO NOTHING
	`

	if _, err := r.db.ExecContext(ctx, query, tokenHash, userID, expiresAt); err != nil {
		return fmt.Errorf("save revoked token: %w", err)
	}

	return nil
}

func (r *TokenRepository) RevokeActiveUserTokens(ctx context.Context, userID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin revoke active user tokens transaction: %w", err)
	}
	defer rollbackTx(tx)

	// JWTs remain valid until their hash is added to revoked_tokens. Updating
	// user_sessions alone would only hide the session from admin views, so both
	// writes are done in the same transaction.
	insertQuery := `
		INSERT INTO revoked_tokens (token_hash, user_id, expires_at)
		SELECT token_hash, user_id, expires_at
		FROM user_sessions
		WHERE user_id = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		ON CONFLICT (token_hash) DO NOTHING
	`

	if _, err := tx.ExecContext(ctx, insertQuery, userID); err != nil {
		return fmt.Errorf("save active user tokens as revoked: %w", err)
	}

	updateQuery := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND revoked_at IS NULL
	`

	if _, err := tx.ExecContext(ctx, updateQuery, userID); err != nil {
		return fmt.Errorf("revoke active user sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit revoke active user tokens transaction: %w", err)
	}

	return nil
}

func (r *TokenRepository) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	query := `
		SELECT
			EXISTS (
				SELECT 1
				FROM revoked_tokens
				WHERE token_hash = $1
			)
			OR NOT EXISTS (
				SELECT 1
				FROM user_sessions
				WHERE token_hash = $1
				  AND revoked_at IS NULL
				  AND expires_at > NOW()
			)
	`

	var revokedOrInactive bool

	if err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(&revokedOrInactive); err != nil {
		return false, fmt.Errorf("check revoked or inactive token: %w", err)
	}

	return revokedOrInactive, nil
}

func (r *TokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM revoked_tokens
		WHERE expires_at <= NOW()
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired revoked tokens: %w", err)
	}

	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get deleted revoked tokens count: %w", err)
	}

	return deletedCount, nil
}
