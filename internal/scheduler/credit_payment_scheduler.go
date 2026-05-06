package scheduler

import (
	"context"
	"time"

	"bank-service/internal/services"

	"github.com/sirupsen/logrus"
)

type CreditPaymentScheduler struct {
	creditPaymentService *services.CreditPaymentService
	interval             time.Duration
	logger               *logrus.Logger
}

func NewCreditPaymentScheduler(
	creditPaymentService *services.CreditPaymentService,
	logger *logrus.Logger,
) *CreditPaymentScheduler {
	return &CreditPaymentScheduler{
		creditPaymentService: creditPaymentService,
		interval:             12 * time.Hour,
		logger:               logger,
	}
}

func (s *CreditPaymentScheduler) Start(ctx context.Context) {
	s.logger.WithField("interval", s.interval.String()).Info("credit payment scheduler started")

	go func() {
		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("credit payment scheduler stopped")
				return

			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *CreditPaymentScheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := s.creditPaymentService.ProcessDuePayments(runCtx); err != nil {
		s.logger.WithError(err).Error("credit payment scheduler failed")
	}
}
