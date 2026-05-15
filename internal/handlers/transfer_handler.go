package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var transferErrorRules = joinErrorRules(
	errorRules{
		{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"},
		{target: services.ErrInvalidTransfer, status: http.StatusBadRequest, message: "invalid transfer"},
		{target: services.ErrInvalidDescription, status: http.StatusBadRequest, message: "invalid description"},
	},
	mfaErrorRules,
	accountErrorRules,
)

type TransferHandler struct {
	transferService *services.TransferService
	auditRecorder   audit.Recorder
}

func NewTransferHandler(transferService *services.TransferService, auditRecorder audit.Recorder) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
		auditRecorder:   auditRecorder,
	}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.TransferRequest](w, r, transferErrorRules, "transfer failed",
		func(ctx context.Context, userID int64, request dto.TransferRequest) (int, any, error) {
			response, err := h.transferService.Transfer(ctx, userID, request)
			if err != nil {
				h.recordTransfer(r, userID, 0, "finance.transfer.failed", audit.StatusFailed, request)
				return http.StatusOK, nil, err
			}

			h.recordTransfer(r, userID, response.TransactionID, "finance.transfer.success", audit.StatusSuccess, request)
			return http.StatusOK, response, nil
		})
}

func (h *TransferHandler) recordTransfer(
	r *http.Request,
	userID int64,
	transactionID int64,
	action string,
	status string,
	request dto.TransferRequest,
) {
	resourceID := transactionID
	resourceType := "transaction"
	if transactionID == 0 {
		resourceID = request.FromAccountID
		resourceType = "account"
	}

	recordFinancialAudit(h.auditRecorder, r, userID, action, resourceType, resourceID, status, map[string]any{
		"from_account_id":   request.FromAccountID,
		"to_account_id":     request.ToAccountID,
		"to_account_number": request.RecipientAccountNumber(),
		"amount":            request.Amount,
	})
}
