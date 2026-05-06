package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var requestMFAErrorRules = joinErrorRules(
	errorRules{
		{target: services.ErrInvalidAmount, statusCode: http.StatusBadRequest, message: "invalid amount"},
		{target: services.ErrCardNotFound, statusCode: http.StatusNotFound, message: "card not found"},
	},
	mfaErrorRules,
	accountErrorRules,
	notificationErrorRules,
)

type MFAHandler struct{ mfaService *services.MFAService }

func NewMFAHandler(mfaService *services.MFAService) *MFAHandler {
	return &MFAHandler{mfaService: mfaService}
}

func (h *MFAHandler) RequestCode(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.MFARequest](w, r, requestMFAErrorRules, "request mfa code failed",
		func(ctx context.Context, userID int64, request dto.MFARequest) (int, any, error) {
			err := h.mfaService.RequestCode(ctx, userID, request)
			return http.StatusOK, dto.MessageResponse{Message: "mfa code sent"}, err
		})
}
