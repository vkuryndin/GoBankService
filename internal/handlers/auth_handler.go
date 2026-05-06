package handlers

import (
	"errors"
	"net/http"
	"strings"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var (
	registerErrors = errorMap{
		services.ErrInvalidRegisterData: {statusCode: http.StatusBadRequest, message: "invalid register data"},
		services.ErrEmailAlreadyUsed:    {statusCode: http.StatusConflict, message: "email or username already used"},
	}

	loginErrors = errorMap{
		services.ErrInvalidLoginData:   {statusCode: http.StatusBadRequest, message: "invalid login data"},
		services.ErrInvalidCredentials: {statusCode: http.StatusUnauthorized, message: "invalid login or password"},
	}

	logoutErrors = errorMap{
		services.ErrInvalidToken: {statusCode: http.StatusUnauthorized, message: "invalid token"},
	}
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request dto.RegisterRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.authService.Register(r.Context(), request)
	if err != nil {
		writeMappedError(w, err, registerErrors, "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request dto.LoginRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.authService.Login(r.Context(), request)
	if err != nil {
		writeMappedError(w, err, loginErrors, "login failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenString, err := extractBearerToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.authService.Logout(r.Context(), tokenString); err != nil {
		writeMappedError(w, err, logoutErrors, "logout failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "logout successful",
	})
}

func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user_id":       userID,
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
