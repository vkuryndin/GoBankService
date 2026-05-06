package handlers

import (
	"errors"
	"net/http"
)

type errorMap map[error]errorResponse

type errorResponse struct {
	statusCode int
	message    string
}

func writeMappedError(w http.ResponseWriter, err error, mappings errorMap, fallbackMessage string) {
	for target, response := range mappings {
		if errors.Is(err, target) {
			writeError(w, response.statusCode, response.message)
			return
		}
	}

	writeError(w, http.StatusInternalServerError, fallbackMessage)
}
