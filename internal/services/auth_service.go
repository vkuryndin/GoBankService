package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
	"bank-service/internal/security"
)

var (
	ErrInvalidRegisterData = errors.New("invalid register data")
	ErrInvalidLoginData    = errors.New("invalid login data")
	ErrEmailAlreadyUsed    = errors.New("email or username already used")
	ErrInvalidCredentials  = errors.New("invalid login or password")
	ErrInvalidToken        = errors.New("invalid token")
)

type authUserStore interface {
	Create(ctx context.Context, email, username, passwordHash string) (*models.User, error)
	FindByLogin(ctx context.Context, login string) (*models.User, error)
}

type revokedTokenStore interface {
	SaveRevokedToken(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error
}

type userSessionStore interface {
	CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
}

type AuthService struct {
	userRepository        authUserStore
	tokenRepository       revokedTokenStore
	userSessionRepository userSessionStore
	jwtSecret             string
}

func NewAuthService(
	userRepository authUserStore,
	tokenRepository revokedTokenStore,
	userSessionRepository userSessionStore,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		userRepository:        userRepository,
		tokenRepository:       tokenRepository,
		userSessionRepository: userSessionRepository,
		jwtSecret:             jwtSecret,
	}
}

func (s *AuthService) Register(ctx context.Context, request dto.RegisterRequest) (*dto.RegisterResponse, error) {
	email, username, password, err := normalizeRegistrationInput(request)
	if err != nil {
		return nil, err
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepository.Create(ctx, email, username, passwordHash)
	if err != nil {
		if errors.Is(err, repositories.ErrUserAlreadyExists) {
			return nil, ErrEmailAlreadyUsed
		}

		return nil, err
	}

	return &dto.RegisterResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, request dto.LoginRequest) (*dto.LoginResponse, error) {
	login := strings.TrimSpace(strings.ToLower(request.Login))
	password := request.Password

	if login == "" || password == "" {
		return nil, fmt.Errorf("%w: login and password are required", ErrInvalidLoginData)
	}

	user, err := s.userRepository.FindByLogin(ctx, login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !security.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := security.GenerateJWT(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate jwt: %w", err)
	}

	claims, err := security.ParseJWT(token, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("parse generated jwt: %w", err)
	}

	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("generated jwt has no expiration")
	}

	tokenHash := security.HashToken(token)

	if err := s.userSessionRepository.CreateSession(ctx, user.ID, tokenHash, claims.ExpiresAt.Time); err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return ErrInvalidToken
	}

	claims, err := security.ParseJWT(tokenString, s.jwtSecret)
	if err != nil {
		return ErrInvalidToken
	}

	if claims.ExpiresAt == nil {
		return ErrInvalidToken
	}

	tokenHash := security.HashToken(tokenString)

	if err := s.tokenRepository.SaveRevokedToken(
		ctx,
		tokenHash,
		claims.UserID,
		claims.ExpiresAt.Time,
	); err != nil {
		return err
	}

	if err := s.userSessionRepository.RevokeByTokenHash(ctx, tokenHash); err != nil {
		return err
	}

	return nil
}

func normalizeRegistrationInput(request dto.RegisterRequest) (string, string, string, error) {
	email := strings.TrimSpace(strings.ToLower(request.Email))
	username := strings.TrimSpace(request.Username)
	password := request.Password

	if email == "" || username == "" || password == "" {
		return "", "", "", fmt.Errorf("%w: email, username and password are required", ErrInvalidRegisterData)
	}

	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return "", "", "", fmt.Errorf("%w: invalid email", ErrInvalidRegisterData)
	}

	if len(username) < minimumUsernameLength {
		return "", "", "", fmt.Errorf("%w: username must be at least %d characters", ErrInvalidRegisterData, minimumUsernameLength)
	}

	if len(password) < minimumPasswordLength {
		return "", "", "", fmt.Errorf("%w: password must be at least %d characters", ErrInvalidRegisterData, minimumPasswordLength)
	}

	return email, username, password, nil
}
