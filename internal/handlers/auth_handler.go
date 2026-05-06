package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/services"
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

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.authService.Register(r.Context(), request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRegisterData) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if errors.Is(err, services.ErrEmailAlreadyUsed) {
			writeError(w, http.StatusConflict, "email or username already used")
			return
		}

		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request dto.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.authService.Login(r.Context(), request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidLoginData) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if errors.Is(err, services.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid login or password")
			return
		}

		writeError(w, http.StatusInternalServerError, "login failed")
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
		if errors.Is(err, services.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "logout successful",
	})
}

func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
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
