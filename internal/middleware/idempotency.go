package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"bank-service/internal/audit"
	"bank-service/internal/repositories"

	"github.com/sirupsen/logrus"
)

const idempotencyHeader = "Idempotency-Key"

type idempotencyStore interface {
	ClaimKey(ctx context.Context, userID int64, method string, path string, key string, requestHash string) error
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
	auditRecorder audit.Recorder,
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

			requestHash, err := hashRequestBody(r)
			if err != nil {
				logger.WithError(err).Warn("idempotency request hash failed")
				writeMiddlewareError(w, http.StatusBadRequest, "invalid request body")
				return
			}

			if err := store.ClaimKey(r.Context(), userID, r.Method, r.URL.Path, key, requestHash); err != nil {
				if errors.Is(err, repositories.ErrIdempotencyKeyAlreadyUsed) {
					recordIdempotencyDuplicate(r, auditRecorder, userID, false)
					writeMiddlewareError(w, http.StatusConflict, "duplicate idempotency key")
					return
				}

				if errors.Is(err, repositories.ErrIdempotencyKeyConflict) {
					recordIdempotencyDuplicate(r, auditRecorder, userID, true)
					writeMiddlewareError(w, http.StatusConflict, "idempotency key reused with different request")
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

func hashRequestBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return hashBytes(nil), nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	return hashBytes(body), nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func recordIdempotencyDuplicate(r *http.Request, auditRecorder audit.Recorder, userID int64, requestConflict bool) {
	if auditRecorder == nil {
		return
	}

	details := map[string]any{
		"request_id": RequestIDFromContext(r.Context()),
		"method":     r.Method,
		"path":       r.URL.Path,
	}
	if requestConflict {
		details["request_conflict"] = true
	}

	auditRecorder.Record(context.Background(), audit.Event{
		UserID:    audit.Int64Ptr(userID),
		Action:    "security.idempotency.duplicate",
		Status:    audit.StatusBlocked,
		IPAddress: ClientIP(r),
		UserAgent: r.UserAgent(),
		Details:   details,
	})
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
