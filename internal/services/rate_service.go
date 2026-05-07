package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/integrations/cbr"
)

const bankMargin = 5.00

type keyRateProvider interface {
	GetKeyRate(ctx context.Context) (*cbr.KeyRate, error)
}

type RateService struct {
	cbrClient keyRateProvider
	cacheTTL  time.Duration
	mu        sync.RWMutex
	cached    *cachedKeyRate
}

type cachedKeyRate struct {
	value     *cbr.KeyRate
	fetchedAt time.Time
}

func NewRateService(cbrClient keyRateProvider, cacheTTL time.Duration) *RateService {
	return &RateService{
		cbrClient: cbrClient,
		cacheTTL:  cacheTTL,
	}
}

func (s *RateService) GetKeyRate(ctx context.Context) (*dto.KeyRateResponse, error) {
	keyRate, err := s.getKeyRate(ctx)
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
	keyRate, err := s.getKeyRate(ctx)
	if err != nil {
		return 0, err
	}

	return keyRate.Rate + bankMargin, nil
}

func (s *RateService) getKeyRate(ctx context.Context) (*cbr.KeyRate, error) {
	if cached := s.readCache(); cached != nil {
		return cached, nil
	}

	keyRate, err := s.cbrClient.GetKeyRate(ctx)
	if err != nil {
		// If the CBR integration is temporarily unavailable, a recent stale value
		// is safer than failing credit calculations for every request.
		if cached := s.readStaleCache(); cached != nil {
			return cached, nil
		}

		return nil, err
	}

	s.writeCache(keyRate)

	return keyRate, nil
}

func (s *RateService) readCache() *cbr.KeyRate {
	if s.cacheTTL <= 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cached == nil {
		return nil
	}

	if time.Since(s.cached.fetchedAt) > s.cacheTTL {
		return nil
	}

	return s.cached.value
}

func (s *RateService) readStaleCache() *cbr.KeyRate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cached == nil {
		return nil
	}

	return s.cached.value
}

func (s *RateService) writeCache(keyRate *cbr.KeyRate) {
	if s.cacheTTL <= 0 || keyRate == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cached = &cachedKeyRate{
		value:     keyRate,
		fetchedAt: time.Now(),
	}
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
