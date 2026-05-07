package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var ErrIdempotencyKeyAlreadyUsed = errors.New("idempotency key already used")

type IdempotencyRepository struct {
	db *sql.DB
}

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (r *IdempotencyRepository) ClaimKey(
	ctx context.Context,
	userID int64,
	method string,
	path string,
	key string,
) error {
	query := `
		INSERT INTO idempotency_keys (user_id, method, path, key)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.db.ExecContext(ctx, query, userID, method, path, key); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrIdempotencyKeyAlreadyUsed
		}

		return fmt.Errorf("claim idempotency key: %w", err)
	}

	return nil
}

func (r *IdempotencyRepository) ReleaseKey(
	ctx context.Context,
	userID int64,
	method string,
	path string,
	key string,
) error {
	query := `
		DELETE FROM idempotency_keys
		WHERE user_id = $1
		  AND method = $2
		  AND path = $3
		  AND key = $4
	`

	if _, err := r.db.ExecContext(ctx, query, userID, method, path, key); err != nil {
		return fmt.Errorf("release idempotency key: %w", err)
	}

	return nil
}
