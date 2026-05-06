package handlers

import (
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var transferErrors = errorMap{
	services.ErrInvalidAmount:       {statusCode: http.StatusBadRequest, message: "invalid amount"},
	services.ErrInvalidTransfer:     {statusCode: http.StatusBadRequest, message: "invalid transfer"},
	services.ErrInvalidDescription:  {statusCode: http.StatusBadRequest, message: "invalid description"},
	services.ErrMFACodeRequired:     {statusCode: http.StatusBadRequest, message: "mfa code required"},
	services.ErrInvalidMFACode:      {statusCode: http.StatusForbidden, message: "invalid mfa code"},
	services.ErrInvalidMFAPurpose:   {statusCode: http.StatusBadRequest, message: "invalid mfa purpose"},
	services.ErrInvalidMFAOperation: {statusCode: http.StatusBadRequest, message: "invalid mfa operation"},
	services.ErrAccountNotFound:     {statusCode: http.StatusNotFound, message: "account not found"},
	services.ErrInsufficientFunds:   {statusCode: http.StatusConflict, message: "insufficient funds"},
	services.ErrAccountBlocked:      {statusCode: http.StatusForbidden, message: "account is blocked"},
}

type TransferHandler struct {
	transferService *services.TransferService
}

func NewTransferHandler(transferService *services.TransferService) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
	}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var request dto.TransferRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	response, err := h.transferService.Transfer(r.Context(), userID, request)
	if err != nil {
		writeMappedError(w, err, transferErrors, "transfer failed")
		return
	}

	writeJSON(w, http.StatusOK, response)
}
