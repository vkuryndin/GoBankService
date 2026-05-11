package handlers

import (
	"encoding/json"
	"net/http"

	"bank-service/internal/dto"
)

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, dto.ErrorResponse{
		Error: message,
	})
}

func writeErrorWithDetails(w http.ResponseWriter, statusCode int, message string, details any) {
	writeJSON(w, statusCode, dto.ErrorResponse{
		Error:   message,
		Details: details,
	})
}
