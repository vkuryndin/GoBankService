package handlers

import (
	"context"
	"net/http"

	"bank-service/internal/dto"
	"bank-service/internal/services"
)

type NotificationHandler struct{ notificationService *services.NotificationService }

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	handleAuthed(w, r, notificationErrorRules, "send test email failed", func(ctx context.Context, userID int64) (int, any, error) {
		err := h.notificationService.SendTestEmail(ctx, userID)
		return http.StatusOK, dto.MessageResponse{Message: "test email sent"}, err
	})
}
