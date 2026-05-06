package scheduler

import (
	"context"
	"time"

	"bank-service/internal/repositories"

	"github.com/sirupsen/logrus"
)

type MFACleanupScheduler struct {
	mfaRepository *repositories.MFARepository
	interval      time.Duration
	logger        *logrus.Logger
}

func NewMFACleanupScheduler(
	mfaRepository *repositories.MFARepository,
	logger *logrus.Logger,
) *MFACleanupScheduler {
	return &MFACleanupScheduler{
		mfaRepository: mfaRepository,
		interval:      time.Hour,
		logger:        logger,
	}
}

func (s *MFACleanupScheduler) Start(ctx context.Context) {
	s.logger.WithField("interval", s.interval.String()).Info("mfa cleanup scheduler started")

	go func() {
		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("mfa cleanup scheduler stopped")
				return

			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *MFACleanupScheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	deletedCount, err := s.mfaRepository.DeleteExpired(runCtx)
	if err != nil {
		s.logger.WithError(err).Error("mfa cleanup scheduler failed")
		return
	}

	s.logger.WithField("deleted_count", deletedCount).Info("mfa cleanup scheduler completed")
}
