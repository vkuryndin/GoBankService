package scheduler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type idempotencyCleanupRepository interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type IdempotencyCleanupScheduler struct {
	repository idempotencyCleanupRepository
	logger     *logrus.Logger
	interval   time.Duration
	retention  time.Duration
}

func NewIdempotencyCleanupScheduler(
	repository idempotencyCleanupRepository,
	logger *logrus.Logger,
	interval time.Duration,
	retention time.Duration,
) *IdempotencyCleanupScheduler {
	return &IdempotencyCleanupScheduler{
		repository: repository,
		logger:     logger,
		interval:   interval,
		retention:  retention,
	}
}

func (s *IdempotencyCleanupScheduler) Start(ctx context.Context) {
	if s.interval <= 0 || s.retention <= 0 {
		s.logger.Warn("idempotency cleanup scheduler disabled: non-positive interval or retention")
		return
	}

	s.logger.WithFields(logrus.Fields{
		"interval_seconds":  s.interval.Seconds(),
		"retention_seconds": s.retention.Seconds(),
	}).Info("idempotency cleanup scheduler started")

	go func() {
		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("idempotency cleanup scheduler stopped")
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *IdempotencyCleanupScheduler) runOnce(ctx context.Context) {
	cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-s.retention)
	deleted, err := s.repository.DeleteOlderThan(cleanupCtx, cutoff)
	if err != nil {
		s.logger.WithError(err).Warn("idempotency cleanup failed")
		return
	}

	s.logger.WithField("deleted_count", deleted).Info("idempotency cleanup finished")
}
