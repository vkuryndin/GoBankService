package middleware

import (
	"encoding/json"
	"net/http"

	"bank-service/internal/dto"
)

func writeMiddlewareError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{Error: message})
}
