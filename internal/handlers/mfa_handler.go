package handlers

import (
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var requestMFAErrors = errorMap{
	services.ErrInvalidMFAPurpose:        {statusCode: http.StatusBadRequest, message: "invalid mfa purpose"},
	services.ErrInvalidMFAOperation:      {statusCode: http.StatusBadRequest, message: "invalid mfa operation"},
	services.ErrInvalidAmount:            {statusCode: http.StatusBadRequest, message: "invalid amount"},
	services.ErrAccountNotFound:          {statusCode: http.StatusNotFound, message: "account not found"},
	services.ErrCardNotFound:             {statusCode: http.StatusNotFound, message: "card not found"},
	services.ErrNotificationsDisabled:    {statusCode: http.StatusServiceUnavailable, message: "smtp notifications disabled"},
	services.ErrNotificationUserNotFound: {statusCode: http.StatusNotFound, message: "user not found"},
	services.ErrAccountBlocked:           {statusCode: http.StatusForbidden, message: "account is blocked"},
}

type MFAHandler struct {
	mfaService *services.MFAService
}

func NewMFAHandler(mfaService *services.MFAService) *MFAHandler {
	return &MFAHandler{
		mfaService: mfaService,
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
		writeMappedError(w, err, requestMFAErrors, "request mfa code failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "mfa code sent",
	})
}
