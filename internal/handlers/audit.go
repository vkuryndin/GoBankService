package handlers

import (
	"net/http"

	"bank-service/internal/audit"
	"bank-service/internal/middleware"
)

func recordRequestAudit(
	auditRecorder audit.Recorder,
	r *http.Request,
	userID *int64,
	action string,
	resourceType string,
	resourceID *int64,
	status string,
	details map[string]any,
) {
	if auditRecorder == nil {
		return
	}

	if details == nil {
		details = map[string]any{}
	}
	details["request_id"] = middleware.RequestIDFromContext(r.Context())
	if r.Method != "" {
		details["method"] = r.Method
	}
	if r.URL != nil {
		details["path"] = r.URL.Path
	}

	auditRecorder.Record(r.Context(), audit.Event{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       status,
		IPAddress:    middleware.ClientIP(r),
		UserAgent:    r.UserAgent(),
		Details:      details,
	})
}
