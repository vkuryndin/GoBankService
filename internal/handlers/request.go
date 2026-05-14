package handlers

import (
	"encoding/json"
	"io"
	"net/http"
)

type requestValidator interface {
	Validate() error
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	if validator, ok := target.(requestValidator); ok {
		if err := validator.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return false
		}
	}

	return true
}
