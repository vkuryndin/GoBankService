package handlers

import (
	"context"
	"net/http"
)

func handleAuthed(
	w http.ResponseWriter,
	r *http.Request,
	rules errorRules,
	fallbackMessage string,
	action func(ctx context.Context, userID int64) (int, any, error),
) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	statusCode, response, err := action(r.Context(), userID)
	if err != nil {
		writeMappedError(w, err, rules, fallbackMessage)
		return
	}

	writeEndpointResponse(w, statusCode, response)
}

func handleAuthedJSON[T any](
	w http.ResponseWriter,
	r *http.Request,
	rules errorRules,
	fallbackMessage string,
	action func(ctx context.Context, userID int64, request T) (int, any, error),
) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var request T
	if !decodeJSON(w, r, &request) {
		return
	}

	statusCode, response, err := action(r.Context(), userID, request)
	if err != nil {
		writeMappedError(w, err, rules, fallbackMessage)
		return
	}

	writeEndpointResponse(w, statusCode, response)
}

func handleJSON[T any](
	w http.ResponseWriter,
	r *http.Request,
	rules errorRules,
	fallbackMessage string,
	action func(ctx context.Context, request T) (int, any, error),
) {
	var request T
	if !decodeJSON(w, r, &request) {
		return
	}

	statusCode, response, err := action(r.Context(), request)
	if err != nil {
		writeMappedError(w, err, rules, fallbackMessage)
		return
	}

	writeEndpointResponse(w, statusCode, response)
}

func writeEndpointResponse(w http.ResponseWriter, statusCode int, response any) {
	if response == nil {
		w.WriteHeader(statusCode)
		return
	}

	writeJSON(w, statusCode, response)
}
