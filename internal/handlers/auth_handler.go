package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var (
	registerErrorRules = errorRules{
		{target: services.ErrInvalidRegisterData, statusCode: http.StatusBadRequest, message: "invalid register data"},
		{target: services.ErrEmailAlreadyUsed, statusCode: http.StatusConflict, message: "email or username already used"},
	}

	loginErrorRules = errorRules{
		{target: services.ErrInvalidLoginData, statusCode: http.StatusBadRequest, message: "invalid login data"},
		{target: services.ErrInvalidCredentials, statusCode: http.StatusUnauthorized, message: "invalid login or password"},
	}

	logoutErrorRules = errorRules{
		{target: services.ErrInvalidToken, statusCode: http.StatusUnauthorized, message: "invalid token"},
	}
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	handleJSON[dto.RegisterRequest](w, r, registerErrorRules, "registration failed",
		func(ctx context.Context, request dto.RegisterRequest) (int, any, error) {
			response, err := h.authService.Register(ctx, request)
			return http.StatusCreated, response, err
		})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	handleJSON[dto.LoginRequest](w, r, loginErrorRules, "login failed",
		func(ctx context.Context, request dto.LoginRequest) (int, any, error) {
			response, err := h.authService.Login(ctx, request)
			return http.StatusOK, response, err
		})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenString, err := extractBearerToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.authService.Logout(r.Context(), tokenString); err != nil {
		writeMappedError(w, err, logoutErrorRules, "logout failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "logout successful"})
}

func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, nil, "auth check failed",
		func(ctx context.Context, userID int64) (int, any, error) {
			return http.StatusOK, map[string]any{"authenticated": true, "user_id": userID}, nil
		})
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("authorization header must start with Bearer")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenString == "" {
		return "", errors.New("token is required")
	}

	return tokenString, nil
}
