package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := strings.TrimSpace(r.Header.Get(RequestIDHeader))
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set(RequestIDHeader, requestID)
			ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	if !ok {
		return ""
	}

	return requestID
}

func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}

	return hex.EncodeToString(buf)
}
