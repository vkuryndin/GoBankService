package services

import (
	"errors"
	"fmt"
	"strings"

	"bank-service/internal/models"
	"bank-service/internal/security"
)

var ErrInvalidCVV = errors.New("invalid cvv")

type CardProcessingService struct {
}

func NewCardProcessingService() *CardProcessingService {
	return &CardProcessingService{}
}

func (s *CardProcessingService) GenerateCVVAndHash() (string, string, error) {
	cvv, err := security.GenerateCVV()
	if err != nil {
		return "", "", fmt.Errorf("generate cvv: %w", err)
	}

	cvvHash, err := security.HashPassword(cvv)
	if err != nil {
		return "", "", fmt.Errorf("hash cvv: %w", err)
	}

	return cvv, cvvHash, nil
}

func (s *CardProcessingService) VerifyCVV(card *models.CardDetails, cvv string) error {
	cvv = strings.TrimSpace(cvv)

	if !isValidCVV(cvv) {
		return ErrInvalidCVV
	}

	if !security.CheckPassword(cvv, card.CVVHash) {
		return ErrInvalidCVV
	}

	return nil
}

func isValidCVV(cvv string) bool {
	if len(cvv) != 3 {
		return false
	}

	for _, symbol := range cvv {
		if symbol < '0' || symbol > '9' {
			return false
		}
	}

	return true
}
