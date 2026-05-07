package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bank-service/internal/repositories"

	"github.com/sirupsen/logrus"
)

const idempotencyHeader = "Idempotency-Key"

type idempotencyStore interface {
	ClaimKey(ctx context.Context, userID int64, method string, path string, key string) error
	ReleaseKey(ctx context.Context, userID int64, method string, path string, key string) error
}

type IdempotencyConfig struct {
	Enabled  bool
	Required bool
}

func IdempotencyMiddleware(
	store idempotencyStore,
	config IdempotencyConfig,
	logger *logrus.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled || !isIdempotentProtectedOperation(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := strings.TrimSpace(r.Header.Get(idempotencyHeader))
			if key == "" {
				if config.Required {
					writeMiddlewareError(w, http.StatusBadRequest, "idempotency key required")
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			if len(key) > 128 {
				writeMiddlewareError(w, http.StatusBadRequest, "invalid idempotency key")
				return
			}

			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				writeMiddlewareError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if err := store.ClaimKey(r.Context(), userID, r.Method, r.URL.Path, key); err != nil {
				if errors.Is(err, repositories.ErrIdempotencyKeyAlreadyUsed) {
					writeMiddlewareError(w, http.StatusConflict, "duplicate idempotency key")
					return
				}

				logger.WithError(err).Error("idempotency claim failed")
				writeMiddlewareError(w, http.StatusInternalServerError, "idempotency check failed")
				return
			}

			recorder := newStatusRecorder(w)
			next.ServeHTTP(recorder, r)

			// Validation and business errors should not permanently reserve a key.
			// Successful requests keep the key to prevent double execution.
			if recorder.statusCode >= http.StatusBadRequest {
				if err := store.ReleaseKey(context.Background(), userID, r.Method, r.URL.Path, key); err != nil {
					logger.WithError(err).Warn("idempotency key release failed")
				}
			}
		})
	}
}

func isIdempotentProtectedOperation(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	path := r.URL.Path
	return path == "/transfer" ||
		path == "/credits" ||
		(strings.HasPrefix(path, "/accounts/") && (strings.HasSuffix(path, "/deposit") || strings.HasSuffix(path, "/withdraw"))) ||
		(strings.HasPrefix(path, "/cards/") && strings.HasSuffix(path, "/pay"))
}
