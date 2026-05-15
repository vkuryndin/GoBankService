package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bank-service/internal/middleware"

	"github.com/sirupsen/logrus"
)

type requestValidator interface {
	Validate() error
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		logInvalidRequestBody(r, target, "decode_json", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		logInvalidRequestBody(r, target, "extra_json_data", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	if validator, ok := target.(requestValidator); ok {
		if err := validator.Validate(); err != nil {
			logInvalidRequestBody(r, target, "validate_json", err)
			writeError(w, http.StatusBadRequest, "invalid request body")
			return false
		}
	}

	return true
}

func logInvalidRequestBody(r *http.Request, target any, stage string, err error) {
	fields := logrus.Fields{
		"request_id":   middleware.RequestIDFromContext(r.Context()),
		"method":       r.Method,
		"path":         r.URL.Path,
		"query":        r.URL.RawQuery,
		"content_type": r.Header.Get("Content-Type"),
		"remote_addr":  r.RemoteAddr,
		"user_agent":   r.UserAgent(),
		"target_type":  fmt.Sprintf("%T", target),
		"stage":        stage,
		"error":        err.Error(),
	}

	if userID, ok := middleware.GetUserIDFromContext(r.Context()); ok {
		fields["user_id"] = userID
	}

	logrus.WithFields(fields).Warn("invalid request body")
}
