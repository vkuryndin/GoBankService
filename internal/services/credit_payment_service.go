package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bank-service/internal/audit"
	"bank-service/internal/repositories"

	"github.com/sirupsen/logrus"
)

type creditPaymentStore interface {
	FindDuePayments(ctx context.Context, limit int) ([]repositories.DueCreditPayment, error)
	ProcessPayment(ctx context.Context, payment repositories.DueCreditPayment) (*repositories.CreditPaymentProcessResult, error)
}

type creditPaymentNotifier interface {
	SendCreditPaymentEmail(ctx context.Context, userID int64, amount string, status string) error
}

type CreditPaymentService struct {
	creditPaymentRepository creditPaymentStore
	notificationService     creditPaymentNotifier
	auditRecorder           audit.Recorder
	logger                  *logrus.Logger
}

func NewCreditPaymentService(
	creditPaymentRepository creditPaymentStore,
	notificationService creditPaymentNotifier,
	auditRecorder audit.Recorder,
	logger *logrus.Logger,
) *CreditPaymentService {
	return &CreditPaymentService{
		creditPaymentRepository: creditPaymentRepository,
		notificationService:     notificationService,
		auditRecorder:           auditRecorder,
		logger:                  logger,
	}
}

func (s *CreditPaymentService) ProcessDuePayments(ctx context.Context) error {
	const batchLimit = 100

	payments, err := s.creditPaymentRepository.FindDuePayments(ctx, batchLimit)
	if err != nil {
		return err
	}

	if len(payments) == 0 {
		s.logger.Info("credit payment scheduler: no due payments")
		return nil
	}

	s.logger.WithField("count", len(payments)).Info("credit payment scheduler: due payments found")

	for _, payment := range payments {
		if err := s.processPayment(ctx, payment); err != nil {
			s.recordCreditPaymentAudit(ctx, payment, nil, "finance.credit_payment.failed", audit.StatusFailed)
			s.logger.WithError(err).WithFields(logrus.Fields{
				"schedule_id": payment.ScheduleID,
				"credit_id":   payment.CreditID,
				"user_id":     payment.UserID,
			}).Error("credit payment scheduler: payment processing failed")
		}
	}

	return nil
}

func (s *CreditPaymentService) processPayment(ctx context.Context, payment repositories.DueCreditPayment) error {
	result, err := s.creditPaymentRepository.ProcessPayment(ctx, payment)
	if err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id":    result.ScheduleID,
		"credit_id":      result.CreditID,
		"user_id":        result.UserID,
		"amount":         result.Amount,
		"penalty_amount": result.PenaltyAmount,
		"status":         result.Status,
	}).Info("credit payment scheduler: payment processed")

	action := "finance.credit_payment.success"
	status := audit.StatusSuccess
	if result.Status == "overdue" {
		action = "finance.credit_payment.failed"
		status = audit.StatusFailed
	}
	s.recordCreditPaymentAudit(ctx, payment, result, action, status)

	notifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	notificationStatus := result.Status
	if result.Status == "overdue" {
		notificationStatus = fmt.Sprintf("overdue, penalty %s RUB", result.PenaltyAmount)
	}

	err = s.notificationService.SendCreditPaymentEmail(notifyCtx, result.UserID, result.Amount, notificationStatus)
	if err != nil {
		if errors.Is(err, ErrNotificationsDisabled) {
			s.logger.WithField("user_id", result.UserID).Warn("credit payment notification skipped: smtp disabled")
			return nil
		}

		s.logger.WithError(err).WithField("user_id", result.UserID).Warn("credit payment notification failed")
		return nil
	}

	return nil
}

func (s *CreditPaymentService) recordCreditPaymentAudit(
	ctx context.Context,
	payment repositories.DueCreditPayment,
	result *repositories.CreditPaymentProcessResult,
	action string,
	status string,
) {
	if s.auditRecorder == nil {
		return
	}

	amount := payment.Amount
	penaltyAmount := payment.PenaltyAmount
	if result != nil {
		amount = result.Amount
		penaltyAmount = result.PenaltyAmount
	}

	s.auditRecorder.Record(ctx, audit.Event{
		UserID:       audit.Int64Ptr(payment.UserID),
		Action:       action,
		ResourceType: "credit",
		ResourceID:   audit.Int64Ptr(payment.CreditID),
		Status:       status,
		Details: map[string]any{
			"schedule_id":    payment.ScheduleID,
			"account_id":     payment.AccountID,
			"amount":         amount,
			"penalty_amount": penaltyAmount,
			"source":         "scheduler",
		},
	})
}
