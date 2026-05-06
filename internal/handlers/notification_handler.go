package handlers

import (
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

var notificationErrors = errorMap{
	services.ErrNotificationsDisabled:    {statusCode: http.StatusServiceUnavailable, message: "smtp notifications disabled"},
	services.ErrNotificationUserNotFound: {statusCode: http.StatusNotFound, message: "user not found"},
}

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

func (h *NotificationHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	if err := h.notificationService.SendTestEmail(r.Context(), userID); err != nil {
		writeMappedError(w, err, notificationErrors, "send test email failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "test email sent",
	})
}
