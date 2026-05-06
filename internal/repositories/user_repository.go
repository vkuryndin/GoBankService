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
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, username, passwordHash string) (*models.User, error) {
	// The first registered user becomes an admin to bootstrap the educational setup
	// without a separate seed script.
	query := `
		INSERT INTO users (
			email,
			username,
			password_hash,
			is_admin
		)
		VALUES (
			$1,
			$2,
			$3,
			NOT EXISTS (SELECT 1 FROM users)
		)
		RETURNING id, email, username, password_hash, created_at
	`

	user := &models.User{}

	err := r.db.QueryRowContext(ctx, query, email, username, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrUserAlreadyExists
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, email, username, password_hash, created_at
		FROM users
		WHERE LOWER(email) = LOWER($1) OR LOWER(username) = LOWER($1)
	`

	user := &models.User{}

	err := r.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find user by login: %w", err)
	}

	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, userID int64) (*models.User, error) {
	query := `
		SELECT id, email, username, password_hash, created_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	query := `
		SELECT is_admin
		FROM users
		WHERE id = $1
	`

	var isAdmin bool

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrUserNotFound
		}

		return false, fmt.Errorf("check user admin: %w", err)
	}

	return isAdmin, nil
}
