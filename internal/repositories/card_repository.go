package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"bank-service/internal/models"

	"github.com/lib/pq"
)

var (
	ErrCardNotFound      = errors.New("card not found")
	ErrCardAlreadyExists = errors.New("card already exists")
)

type CardRepository struct {
	db *sql.DB
}

func NewCardRepository(db *sql.DB) *CardRepository {
	return &CardRepository{
		db: db,
	}
}

func (r *CardRepository) Create(
	ctx context.Context,
	userID int64,
	accountID int64,
	number string,
	expiry string,
	cvvHash string,
	numberHMAC string,
	pgpKey string,
) (*models.CardDetails, error) {
	query := `
		INSERT INTO cards (
			user_id,
			account_id,
			encrypted_number,
			encrypted_expiry,
			cvv_hash,
			number_hmac
		)
		SELECT
			$1,
			$2,
			pgp_sym_encrypt($3, $7),
			pgp_sym_encrypt($4, $7),
			$5,
			$6
		WHERE EXISTS (
			SELECT 1
			FROM accounts
			WHERE id = $2 AND user_id = $1
		)
		RETURNING
			id,
			user_id,
			account_id,
			pgp_sym_decrypt(encrypted_number, $7),
			pgp_sym_decrypt(encrypted_expiry, $7),
			cvv_hash,
			number_hmac,
			created_at
	`

	card := &models.CardDetails{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		accountID,
		number,
		expiry,
		cvvHash,
		numberHMAC,
		pgpKey,
	).Scan(
		&card.ID,
		&card.UserID,
		&card.AccountID,
		&card.Number,
		&card.Expiry,
		&card.CVVHash,
		&card.NumberHMAC,
		&card.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}

		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrCardAlreadyExists
		}

		return nil, fmt.Errorf("create card: %w", err)
	}

	return card, nil
}

func (r *CardRepository) FindByUserID(ctx context.Context, userID int64, pgpKey string) ([]models.CardDetails, error) {
	query := `
		SELECT
			id,
			user_id,
			account_id,
			pgp_sym_decrypt(encrypted_number, $2),
			pgp_sym_decrypt(encrypted_expiry, $2),
			cvv_hash,
			number_hmac,
			created_at
		FROM cards
		WHERE user_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userID, pgpKey)
	if err != nil {
		return nil, fmt.Errorf("find cards by user id: %w", err)
	}
	defer rows.Close()

	cards := make([]models.CardDetails, 0)

	for rows.Next() {
		var card models.CardDetails

		if err := rows.Scan(
			&card.ID,
			&card.UserID,
			&card.AccountID,
			&card.Number,
			&card.Expiry,
			&card.CVVHash,
			&card.NumberHMAC,
			&card.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan card: %w", err)
		}

		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cards: %w", err)
	}

	return cards, nil
}

func (r *CardRepository) FindByIDAndUserID(
	ctx context.Context,
	cardID int64,
	userID int64,
	pgpKey string,
) (*models.CardDetails, error) {
	query := `
		SELECT
			id,
			user_id,
			account_id,
			pgp_sym_decrypt(encrypted_number, $3),
			pgp_sym_decrypt(encrypted_expiry, $3),
			cvv_hash,
			number_hmac,
			created_at
		FROM cards
		WHERE id = $1 AND user_id = $2
	`

	card := &models.CardDetails{}

	err := r.db.QueryRowContext(ctx, query, cardID, userID, pgpKey).Scan(
		&card.ID,
		&card.UserID,
		&card.AccountID,
		&card.Number,
		&card.Expiry,
		&card.CVVHash,
		&card.NumberHMAC,
		&card.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCardNotFound
		}

		return nil, fmt.Errorf("find card by id and user id: %w", err)
	}

	return card, nil
}

func (r *CardRepository) FindAccountIDByIDAndUserID(ctx context.Context, cardID int64, userID int64) (int64, error) {
	query := `
		SELECT account_id
		FROM cards
		WHERE id = $1 AND user_id = $2
	`

	var accountID int64

	err := r.db.QueryRowContext(ctx, query, cardID, userID).Scan(&accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrCardNotFound
		}

		return 0, fmt.Errorf("find card account id by id and user id: %w", err)
	}

	return accountID, nil
}

func (r *CardRepository) ValidateCardOwnership(ctx context.Context, userID int64, cardID int64) error {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM cards
			WHERE id = $1 AND user_id = $2
		)
	`

	var exists bool

	err := r.db.QueryRowContext(ctx, query, cardID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("validate card ownership: %w", err)
	}

	if !exists {
		return ErrCardNotFound
	}

	return nil
}
