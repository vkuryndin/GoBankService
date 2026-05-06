package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/services"
)

type MFAHandler struct {
	mfaService *services.MFAService
}

func NewMFAHandler(mfaService *services.MFAService) *MFAHandler {
	return &MFAHandler{
		mfaService: mfaService,
	}
}

func (h *MFAHandler) RequestCode(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	var request dto.MFARequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.mfaService.RequestCode(r.Context(), userID, request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidMFAPurpose) {
			writeError(w, http.StatusBadRequest, "invalid mfa purpose")
			return
		}

		if errors.Is(err, services.ErrInvalidMFAOperation) {
			writeError(w, http.StatusBadRequest, "invalid mfa operation")
			return
		}

		if errors.Is(err, services.ErrInvalidAmount) {
			writeError(w, http.StatusBadRequest, "invalid amount")
			return
		}

		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		if errors.Is(err, services.ErrCardNotFound) {
			writeError(w, http.StatusNotFound, "card not found")
			return
		}

		if errors.Is(err, services.ErrNotificationsDisabled) {
			writeError(w, http.StatusServiceUnavailable, "smtp notifications disabled")
			return
		}

		if errors.Is(err, services.ErrNotificationUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		if errors.Is(err, services.ErrAccountBlocked) {
			writeError(w, http.StatusForbidden, "account is blocked")
			return
		}

		writeError(w, http.StatusInternalServerError, "request mfa code failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "mfa code sent",
	})
}
