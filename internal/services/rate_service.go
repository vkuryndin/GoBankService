package services

import (
	"context"
	"fmt"

	"bank-service/internal/dto"
	"bank-service/internal/integrations/cbr"
)

const bankMargin = 5.00

type RateService struct {
	cbrClient *cbr.Client
}

func NewRateService(cbrClient *cbr.Client) *RateService {
	return &RateService{
		cbrClient: cbrClient,
	}
}

func (s *RateService) GetKeyRate(ctx context.Context) (*dto.KeyRateResponse, error) {
	keyRate, err := s.cbrClient.GetKeyRate(ctx)
	if err != nil {
		return nil, err
	}

	bankRate := keyRate.Rate + bankMargin

	return &dto.KeyRateResponse{
		KeyRate:    formatPercent(keyRate.Rate),
		BankRate:   formatPercent(bankRate),
		BankMargin: formatPercent(bankMargin),
		Date:       keyRate.Date,
		Source:     "cbr.ru",
	}, nil
}

func (s *RateService) GetBankRateValue(ctx context.Context) (float64, error) {
	keyRate, err := s.cbrClient.GetKeyRate(ctx)
	if err != nil {
		return 0, err
	}

	return keyRate.Rate + bankMargin, nil
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
