package services

import (
	"context"
	"time"

	"bank-service/internal/audit"

	"github.com/sirupsen/logrus"
)

type auditStore interface {
	Create(ctx context.Context, event audit.Event) error
}

type AuditService struct {
	auditRepository auditStore
	logger          *logrus.Logger
}

func NewAuditService(auditRepository auditStore, logger *logrus.Logger) *AuditService {
	return &AuditService{
		auditRepository: auditRepository,
		logger:          logger,
	}
}

func (s *AuditService) Record(ctx context.Context, event audit.Event) {
	if s == nil || s.auditRepository == nil {
		return
	}

	if event.Status == "" {
		event.Status = audit.StatusSuccess
	}

	if event.Details == nil {
		event.Details = map[string]any{}
	}

	// Audit logging must not break the main banking operation. A short timeout
	// keeps a slow audit insert from delaying auth, MFA or admin requests.
	auditCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.auditRepository.Create(auditCtx, event); err != nil && s.logger != nil {
		s.logger.WithError(err).Warn("audit log write failed")
	}
}
