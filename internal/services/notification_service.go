package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	smtpclient "bank-service/internal/integrations/smtp"
	"bank-service/internal/models"
	"bank-service/internal/repositories"
)

var (
	ErrNotificationsDisabled    = errors.New("notifications disabled")
	ErrNotificationUserNotFound = errors.New("notification user not found")
)

type notificationUserStore interface {
	FindByID(ctx context.Context, userID int64) (*models.User, error)
}

type emailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type NotificationService struct {
	userRepository notificationUserStore
	smtpClient     emailSender
}

func NewNotificationService(
	userRepository notificationUserStore,
	smtpClient emailSender,
) *NotificationService {
	return &NotificationService{
		userRepository: userRepository,
		smtpClient:     smtpClient,
	}
}

func (s *NotificationService) SendTestEmail(ctx context.Context, userID int64) error {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return ErrNotificationUserNotFound
	}

	subject := "Bank Service: test notification"
	body := fmt.Sprintf(`
		<h2>Тестовое уведомление Bank Service</h2>
		<p>Здравствуйте, %s.</p>
		<p>SMTP-интеграция работает.</p>
		<p>Время проверки: %s</p>
	`, user.Username, time.Now().Format(time.RFC3339))

	err = s.smtpClient.Send(ctx, user.Email, subject, body)
	if err != nil {
		if errors.Is(err, smtpclient.ErrSMTPDisabled) {
			return ErrNotificationsDisabled
		}

		return err
	}

	return nil
}

func (s *NotificationService) SendCreditPaymentEmail(
	ctx context.Context,
	userID int64,
	amount string,
	status string,
) error {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return ErrNotificationUserNotFound
	}

	subject := "Bank Service: credit payment notification"
	body := fmt.Sprintf(`
		<h2>Уведомление по кредитному платежу</h2>
		<p>Сумма: <strong>%s RUB</strong></p>
		<p>Статус: <strong>%s</strong></p>
	`, amount, status)

	err = s.smtpClient.Send(ctx, user.Email, subject, body)
	if err != nil {
		if errors.Is(err, smtpclient.ErrSMTPDisabled) {
			return ErrNotificationsDisabled
		}

		return err
	}

	return nil
}

func (s *NotificationService) SendMFAEmail(
	ctx context.Context,
	userID int64,
	purpose string,
	code string,
) error {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return ErrNotificationUserNotFound
	}

	subject := "Bank Service: MFA code"
	body := fmt.Sprintf(`
		<h2>Код подтверждения операции</h2>
		<p>Операция: <strong>%s</strong></p>
		<p>Ваш код: <strong>%s</strong></p>
		<p>Код действует 5 минут.</p>
	`, purpose, code)

	err = s.smtpClient.Send(ctx, user.Email, subject, body)
	if err != nil {
		if errors.Is(err, smtpclient.ErrSMTPDisabled) {
			return ErrNotificationsDisabled
		}

		return err
	}

	return nil
}

var _ notificationUserStore = (*repositories.UserRepository)(nil)
