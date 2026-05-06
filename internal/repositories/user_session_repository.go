package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type UserSessionRepository struct {
	db *sql.DB
}

func NewUserSessionRepository(db *sql.DB) *UserSessionRepository {
	return &UserSessionRepository{
		db: db,
	}
}

func (r *UserSessionRepository) CreateSession(
	ctx context.Context,
	userID int64,
	tokenHash string,
	expiresAt time.Time,
) error {
	query := `
		INSERT INTO user_sessions (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO NOTHING
	`

	if _, err := r.db.ExecContext(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create user session: %w", err)
	}

	return nil
}

func (r *UserSessionRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`

	if _, err := r.db.ExecContext(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoke user session: %w", err)
	}

	return nil
}
