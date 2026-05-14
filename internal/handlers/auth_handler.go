package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/security"
	"bank-service/internal/security/httpauth"
	"bank-service/internal/services"
)

var (
	registerErrorRules = errorRules{
		{target: services.ErrInvalidRegisterData, status: http.StatusBadRequest, message: "invalid register data"},
		{target: services.ErrEmailAlreadyUsed, status: http.StatusConflict, message: "email or username already used"},
	}

	loginErrorRules = errorRules{
		{target: services.ErrInvalidLoginData, status: http.StatusBadRequest, message: "invalid login data"},
		{target: services.ErrInvalidCredentials, status: http.StatusUnauthorized, message: "invalid login or password"},
	}

	logoutErrorRules = errorRules{
		{target: services.ErrInvalidToken, status: http.StatusUnauthorized, message: "invalid token"},
	}

	authCheckErrorRules = errorRules{
		{target: services.ErrInvalidToken, status: http.StatusUnauthorized, message: "invalid token"},
	}
)

type AuthHandler struct {
	authService   *services.AuthService
	auditRecorder audit.Recorder
}

func NewAuthHandler(authService *services.AuthService, auditRecorder audit.Recorder) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		auditRecorder: auditRecorder,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request dto.RegisterRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.authService.Register(r.Context(), request)
	if err != nil {
		recordRequestAudit(h.auditRecorder, r, nil, "auth.register.failed", "user", nil, audit.StatusFailed, nil)
		writeMappedError(w, r, err, registerErrorRules, "registration failed")
		return
	}

	recordRequestAudit(
		h.auditRecorder,
		r,
		audit.Int64Ptr(response.ID),
		"auth.register.success",
		"user",
		audit.Int64Ptr(response.ID),
		audit.StatusSuccess,
		nil,
	)
	writeJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request dto.LoginRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.authService.Login(r.Context(), request)
	if err != nil {
		recordRequestAudit(h.auditRecorder, r, nil, "auth.login.failed", "user", nil, audit.StatusFailed, map[string]any{
			"login_provided": strings.TrimSpace(request.Login) != "",
		})
		writeMappedError(w, r, err, loginErrorRules, "login failed")
		return
	}

	recordRequestAudit(
		h.auditRecorder,
		r,
		audit.Int64Ptr(response.UserID),
		"auth.login.success",
		"user",
		audit.Int64Ptr(response.UserID),
		audit.StatusSuccess,
		nil,
	)
	setAuthCookie(w, r, response.Token)
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	tokenString, err := httpauth.ExtractTokenFromRequest(r)
	if err != nil {
		recordRequestAudit(h.auditRecorder, r, audit.Int64Ptr(userID), "auth.logout.failed", "user", audit.Int64Ptr(userID), audit.StatusFailed, nil)
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.authService.Logout(r.Context(), tokenString); err != nil {
		recordRequestAudit(h.auditRecorder, r, audit.Int64Ptr(userID), "auth.logout.failed", "user", audit.Int64Ptr(userID), audit.StatusFailed, nil)
		writeMappedError(w, r, err, logoutErrorRules, "logout failed")
		return
	}

	recordRequestAudit(h.auditRecorder, r, audit.Int64Ptr(userID), "auth.logout.success", "user", audit.Int64Ptr(userID), audit.StatusSuccess, nil)
	clearAuthCookie(w, r)
	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "logout successful"})
}

func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, authCheckErrorRules, "auth check failed",
		func(ctx context.Context, userID int64) (int, any, error) {
			response, err := h.authService.CheckAuth(ctx, userID)
			return http.StatusOK, response, err
		})
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     httpauth.AuthCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(security.TokenTTL),
		MaxAge:   int(security.TokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     httpauth.AuthCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
