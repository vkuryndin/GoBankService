package middleware

import (
	"errors"
	"net/http"

	"bank-service/internal/repositories"
)

func AdminMiddleware(userRepository *repositories.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "user is not authenticated")
				return
			}

			isAdmin, err := userRepository.IsAdmin(r.Context(), userID)
			if err != nil {
				if errors.Is(err, repositories.ErrUserNotFound) {
					writeAuthError(w, http.StatusUnauthorized, "user not found")
					return
				}

				writeAuthError(w, http.StatusInternalServerError, "admin check failed")
				return
			}

			if !isAdmin {
				writeAuthError(w, http.StatusForbidden, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
