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

func (r *TokenRepository) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM revoked_tokens
			WHERE token_hash = $1
		)
	`

	var revoked bool

	if err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(&revoked); err != nil {
		return false, fmt.Errorf("check revoked token: %w", err)
	}

	return revoked, nil
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
