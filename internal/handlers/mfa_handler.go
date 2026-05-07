package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var requestMFAErrorRules = joinErrorRules(
	errorRules{
		{target: services.ErrInvalidAmount, status: http.StatusBadRequest, message: "invalid amount"},
	},
	cardErrorRules,
	mfaErrorRules,
	accountErrorRules,
	notificationErrorRules,
)

type MFAHandler struct {
	mfaService    *services.MFAService
	auditRecorder audit.Recorder
}

func NewMFAHandler(mfaService *services.MFAService, auditRecorder audit.Recorder) *MFAHandler {
	return &MFAHandler{
		mfaService:    mfaService,
		auditRecorder: auditRecorder,
	}
}

func (h *MFAHandler) RequestCode(w http.ResponseWriter, r *http.Request) {
	handleAuthedJSON[dto.MFARequest](w, r, requestMFAErrorRules, "request mfa code failed",
		func(ctx context.Context, userID int64, request dto.MFARequest) (int, any, error) {
			if err := h.mfaService.RequestCode(ctx, userID, request); err != nil {
				h.recordMFARequest(r, userID, request.Purpose, "mfa.request.failed", audit.StatusFailed)
				return http.StatusOK, nil, err
			}

			h.recordMFARequest(r, userID, request.Purpose, "mfa.request.success", audit.StatusSuccess)
			return http.StatusOK, dto.MessageResponse{Message: "mfa code sent"}, nil
		})
}

func (h *MFAHandler) recordMFARequest(r *http.Request, userID int64, purpose string, action string, status string) {
	recordRequestAudit(h.auditRecorder, r, audit.Int64Ptr(userID), action, "mfa_code", nil, status, map[string]any{
		"purpose": purpose,
	})
}
