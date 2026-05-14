package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !config.Enabled {
			return next
		}

		allowedOrigins := normalizeList(config.AllowedOrigins)
		allowedMethods := normalizeList(config.AllowedMethods)
		allowedHeaders := normalizeList(config.AllowedHeaders)
		allowAnyOrigin := contains(allowedOrigins, "*")

		if len(allowedMethods) == 0 {
			allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodOptions}
		}
		if len(allowedHeaders) == 0 {
			allowedHeaders = []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"}
		}

		methodsHeader := strings.Join(allowedMethods, ", ")
		headersHeader := strings.Join(allowedHeaders, ", ")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" && (allowAnyOrigin || contains(allowedOrigins, origin)) {
				if allowAnyOrigin && !config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}

				w.Header().Set("Access-Control-Allow-Methods", methodsHeader)
				w.Header().Set("Access-Control-Allow-Headers", headersHeader)
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				if config.MaxAgeSeconds > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAgeSeconds))
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
