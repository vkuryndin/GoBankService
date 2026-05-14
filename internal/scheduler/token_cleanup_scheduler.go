package scheduler

import (
	"context"
	"sync"
	"time"

	"bank-service/internal/repositories"

	"github.com/sirupsen/logrus"
)

type TokenCleanupScheduler struct {
	tokenRepository *repositories.TokenRepository
	interval        time.Duration
	logger          *logrus.Logger
}

func NewTokenCleanupScheduler(
	tokenRepository *repositories.TokenRepository,
	logger *logrus.Logger,
) *TokenCleanupScheduler {
	return &TokenCleanupScheduler{
		tokenRepository: tokenRepository,
		interval:        24 * time.Hour,
		logger:          logger,
	}
}

func (s *TokenCleanupScheduler) Start(ctx context.Context, wg *sync.WaitGroup) {
	s.logger.WithField("interval", s.interval.String()).Info("token cleanup scheduler started")

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("token cleanup scheduler stopped")
				return

			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *TokenCleanupScheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	deletedCount, err := s.tokenRepository.DeleteExpired(runCtx)
	if err != nil {
		s.logger.WithError(err).Error("token cleanup scheduler failed")
		return
	}

	s.logger.WithField("deleted_count", deletedCount).Info("token cleanup scheduler completed")
}
