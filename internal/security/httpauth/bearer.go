package httpauth

import (
	"errors"
	"net/http"
	"strings"
)

const AuthCookieName = "bank_service_session"

func ExtractBearerToken(r *http.Request) (string, error) {
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

func ExtractTokenFromRequest(r *http.Request) (string, error) {
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return ExtractBearerToken(r)
	}

	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		return "", errors.New("authentication token is required")
	}

	tokenString := strings.TrimSpace(cookie.Value)
	if tokenString == "" {
		return "", errors.New("authentication token is required")
	}

	return tokenString, nil
}
