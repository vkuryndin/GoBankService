package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/services"
)

type TransferHandler struct {
	transferService *services.TransferService
}

func NewTransferHandler(transferService *services.TransferService) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
	}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	var request dto.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.transferService.Transfer(r.Context(), userID, request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidAmount) {
			writeError(w, http.StatusBadRequest, "invalid amount")
			return
		}

		if errors.Is(err, services.ErrInvalidTransfer) {
			writeError(w, http.StatusBadRequest, "invalid transfer")
			return
		}

		if errors.Is(err, services.ErrInvalidDescription) {
			writeError(w, http.StatusBadRequest, "invalid description")
			return
		}

		if errors.Is(err, services.ErrMFACodeRequired) {
			writeError(w, http.StatusBadRequest, "mfa code required")
			return
		}

		if errors.Is(err, services.ErrInvalidMFACode) {
			writeError(w, http.StatusForbidden, "invalid mfa code")
			return
		}

		if errors.Is(err, services.ErrInvalidMFAPurpose) {
			writeError(w, http.StatusBadRequest, "invalid mfa purpose")
			return
		}

		if errors.Is(err, services.ErrInvalidMFAOperation) {
			writeError(w, http.StatusBadRequest, "invalid mfa operation")
			return
		}

		if errors.Is(err, services.ErrAccountNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}

		if errors.Is(err, services.ErrInsufficientFunds) {
			writeError(w, http.StatusConflict, "insufficient funds")
			return
		}
		if errors.Is(err, services.ErrAccountBlocked) {
			writeError(w, http.StatusForbidden, "account is blocked")
			return
		}

		writeError(w, http.StatusInternalServerError, "transfer failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}
