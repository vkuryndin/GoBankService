package handlers

import (
	"net/http"

	"bank-service/internal/audit"
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
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var request dto.MFARequest
	if !decodeJSON(w, r, &request) {
		return
	}

	if err := h.mfaService.RequestCode(r.Context(), userID, request); err != nil {
		recordRequestAudit(h.auditRecorder, r, audit.Int64Ptr(userID), "mfa.request.failed", "mfa_code", nil, audit.StatusFailed, map[string]any{
			"purpose": request.Purpose,
		})
		writeMappedError(w, err, requestMFAErrorRules, "request mfa code failed")
		return
	}

	recordRequestAudit(h.auditRecorder, r, audit.Int64Ptr(userID), "mfa.request.success", "mfa_code", nil, audit.StatusSuccess, map[string]any{
		"purpose": request.Purpose,
	})
	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "mfa code sent"})
}
