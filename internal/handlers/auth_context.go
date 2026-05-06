package handlers

import (
	"net/http"

	"bank-service/internal/middleware"
)

func requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return 0, false
	}

	return userID, true
}
