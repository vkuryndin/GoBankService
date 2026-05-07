package middleware

import "net/http"

func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", "DENY")
			header.Set("Referrer-Policy", "no-referrer")
			header.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Banking API responses should not be cached by browsers or proxies.
			header.Set("Cache-Control", "no-store")
			header.Set("Pragma", "no-cache")

			next.ServeHTTP(w, r)
		})
	}
}
