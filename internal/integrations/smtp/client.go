package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"bank-service/internal/config"

	"github.com/go-mail/mail/v2"
	"github.com/sirupsen/logrus"
)

var ErrSMTPDisabled = errors.New("smtp is disabled")

type Client struct {
	config config.SMTPConfig
}

func NewClient(config config.SMTPConfig) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) Send(ctx context.Context, to string, subject string, body string) error {
	if !c.config.Enabled {
		logrus.WithFields(logrus.Fields{
			"enabled": c.config.Enabled,
			"host":    c.config.Host,
			"port":    c.config.Port,
			"user":    c.config.User,
			"from":    c.config.From,
		}).Warn("smtp send skipped: smtp disabled")

		return ErrSMTPDisabled
	}

	logrus.WithFields(logrus.Fields{
		"host":    c.config.Host,
		"port":    c.config.Port,
		"user":    c.config.User,
		"from":    c.config.From,
		"to":      to,
		"subject": subject,
	}).Info("smtp send started")

	message := mail.NewMessage()
	message.SetHeader("From", c.config.From)
	message.SetHeader("To", to)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", body)

	dialer := mail.NewDialer(
		c.config.Host,
		c.config.Port,
		c.config.User,
		c.config.Password,
	)

	dialer.TLSConfig = &tls.Config{
		ServerName:         c.config.Host,
		InsecureSkipVerify: false,
	}

	result := make(chan error, 1)

	go func() {
		result <- dialer.DialAndSend(message)
	}()

	select {
	case <-ctx.Done():
		logrus.WithError(ctx.Err()).Error("smtp send timeout")
		return fmt.Errorf("smtp send timeout: %w", ctx.Err())

	case err := <-result:
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"host": c.config.Host,
				"port": c.config.Port,
				"user": c.config.User,
				"from": c.config.From,
			}).Error("smtp send failed")

			return fmt.Errorf("smtp send failed: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"host": c.config.Host,
			"port": c.config.Port,
			"from": c.config.From,
			"to":   to,
		}).Info("smtp send completed")

		return nil
	}
}
