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
	ErrIdempotencyKeyAlreadyUsed    = errors.New("idempotency key already used")
	ErrIdempotencyKeyConflict       = errors.New("idempotency key reused with different request")
	ErrIdempotencyRequestInProgress = errors.New("idempotency request is still processing")
)

type IdempotencyResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

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
) (*IdempotencyResponse, bool, error) {
	query := `
		INSERT INTO idempotency_keys (user_id, method, path, key, request_hash)
		VALUES ($1, $2, $3, $4, $5)
	`

	if _, err := r.db.ExecContext(ctx, query, userID, method, path, key, requestHash); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			response, err := r.resolveDuplicate(ctx, userID, method, path, key, requestHash)
			if err != nil {
				return nil, false, err
			}

			return response, true, nil
		}

		return nil, false, fmt.Errorf("claim idempotency key: %w", err)
	}

	return nil, false, nil
}

func (r *IdempotencyRepository) resolveDuplicate(
	ctx context.Context,
	userID int64,
	method string,
	path string,
	key string,
	requestHash string,
) (*IdempotencyResponse, error) {
	query := `
		SELECT request_hash, response_status, response_content_type, response_body
		FROM idempotency_keys
		WHERE user_id = $1
		  AND method = $2
		  AND path = $3
		  AND key = $4
	`

	var storedHash string
	var responseStatus sql.NullInt64
	var responseContentType sql.NullString
	var responseBody []byte

	if err := r.db.QueryRowContext(ctx, query, userID, method, path, key).Scan(
		&storedHash,
		&responseStatus,
		&responseContentType,
		&responseBody,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIdempotencyKeyAlreadyUsed
		}

		return nil, fmt.Errorf("read existing idempotency key: %w", err)
	}

	if storedHash != requestHash {
		return nil, ErrIdempotencyKeyConflict
	}

	if !responseStatus.Valid {
		return nil, ErrIdempotencyRequestInProgress
	}

	return &IdempotencyResponse{
		StatusCode:  int(responseStatus.Int64),
		ContentType: responseContentType.String,
		Body:        append([]byte(nil), responseBody...),
	}, nil
}

func (r *IdempotencyRepository) StoreResponse(
	ctx context.Context,
	userID int64,
	method string,
	path string,
	key string,
	statusCode int,
	contentType string,
	body []byte,
) error {
	query := `
		UPDATE idempotency_keys
		SET response_status = $5,
		    response_content_type = $6,
		    response_body = $7,
		    completed_at = NOW()
		WHERE user_id = $1
		  AND method = $2
		  AND path = $3
		  AND key = $4
	`

	result, err := r.db.ExecContext(ctx, query, userID, method, path, key, statusCode, contentType, body)
	if err != nil {
		return fmt.Errorf("store idempotency response: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read stored idempotency response count: %w", err)
	}
	if updated == 0 {
		return ErrIdempotencyKeyAlreadyUsed
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
