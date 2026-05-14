package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/security"
	"bank-service/internal/security/httpauth"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func AuthMiddleware(
	jwtSecret string,
	tokenChecker TokenRevocationChecker,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := httpauth.ExtractTokenFromRequest(r)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, err.Error())
				return
			}

			claims, err := security.ParseJWT(tokenString, jwtSecret)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			tokenHash := security.HashToken(tokenString)

			revoked, err := tokenChecker.IsTokenRevoked(r.Context(), tokenHash)
			if err != nil {
				writeAuthError(w, http.StatusInternalServerError, "token check failed")
				return
			}

			if revoked {
				writeAuthError(w, http.StatusUnauthorized, "token revoked")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	return userID, ok
}

func writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: message,
	})
}
