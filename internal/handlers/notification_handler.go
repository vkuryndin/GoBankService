package handlers

import (
	"errors"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/services"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

func (h *NotificationHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	err := h.notificationService.SendTestEmail(r.Context(), userID)
	if err != nil {
		if errors.Is(err, services.ErrNotificationsDisabled) {
			writeError(w, http.StatusServiceUnavailable, "smtp notifications disabled")
			return
		}

		if errors.Is(err, services.ErrNotificationUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "send test email failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{
		Message: "test email sent",
	})
}
