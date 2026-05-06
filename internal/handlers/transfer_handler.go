package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var transferErrorRules = joinErrorRules(
	errorRules{
		{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"},
		{target: services.ErrInvalidTransfer, statusCode: http.StatusBadRequest, message: "invalid transfer"},
		{target: services.ErrInvalidDescription, statusCode: http.StatusBadRequest, message: "invalid description"},
	},
	mfaErrorRules,
	accountErrorRules,
)

type TransferHandler struct{ transferService *services.TransferService }

func NewTransferHandler(transferService *services.TransferService) *TransferHandler {
	return &TransferHandler{transferService: transferService}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.TransferRequest](w, r, transferErrorRules, "transfer failed",
		func(ctx context.Context, userID int64, request dto.TransferRequest) (int, any, error) {
			response, err := h.transferService.Transfer(ctx, userID, request)
			return http.StatusOK, response, err
		})
}
