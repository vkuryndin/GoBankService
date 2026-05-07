package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	ErrIdempotencyKeyAlreadyUsed = errors.New("idempotency key already used")
	ErrIdempotencyKeyConflict    = errors.New("idempotency key reused with different request")
)

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
	requestHash string,
) error {
	query := `
		INSERT INTO idempotency_keys (user_id, method, path, key, request_hash)
		VALUES ($1, $2, $3, $4, $5)
	`

	if _, err := r.db.ExecContext(ctx, query, userID, method, path, key, requestHash); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return r.resolveDuplicate(ctx, userID, method, path, key, requestHash)
		}

		return fmt.Errorf("claim idempotency key: %w", err)
	}

	return nil
}

func (r *IdempotencyRepository) resolveDuplicate(
	ctx context.Context,
	userID int64,
	method string,
	path string,
	key string,
	requestHash string,
) error {
	query := `
		SELECT request_hash
		FROM idempotency_keys
		WHERE user_id = $1
		  AND method = $2
		  AND path = $3
		  AND key = $4
	`

	var storedHash string
	if err := r.db.QueryRowContext(ctx, query, userID, method, path, key).Scan(&storedHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrIdempotencyKeyAlreadyUsed
		}

		return fmt.Errorf("read existing idempotency key: %w", err)
	}

	if storedHash != requestHash {
		return ErrIdempotencyKeyConflict
	}

	return ErrIdempotencyKeyAlreadyUsed
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

func (r *IdempotencyRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	query := `
		DELETE FROM idempotency_keys
		WHERE created_at < $1
	`

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old idempotency keys: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted idempotency key count: %w", err)
	}

	return deleted, nil
}
