package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"bank-service/internal/dto"
	"bank-service/internal/integrations/cbr"

	"github.com/sirupsen/logrus"
)

const bankMargin = 5.00

type keyRateProvider interface {
	GetKeyRate(ctx context.Context) (*cbr.KeyRate, error)
}

type RateService struct {
	cbrClient keyRateProvider
	cacheTTL  time.Duration
	breaker   *cbrCircuitBreaker
	logger    *logrus.Logger
	mu        sync.RWMutex
	cached    *cachedKeyRate
}

type cachedKeyRate struct {
	value     *cbr.KeyRate
	fetchedAt time.Time
}

type cbrCircuitBreaker struct {
	failureLimit int
	resetTimeout time.Duration
	mu           sync.Mutex
	failures     int
	openedAt     time.Time
}

var ErrCBRCircuitOpen = errors.New("cbr circuit breaker open")

func NewRateService(
	cbrClient keyRateProvider,
	cacheTTL time.Duration,
	breakerFailureLimit int,
	breakerResetTimeout time.Duration,
	logger *logrus.Logger,
) *RateService {
	return &RateService{
		cbrClient: cbrClient,
		cacheTTL:  cacheTTL,
		breaker: &cbrCircuitBreaker{
			failureLimit: breakerFailureLimit,
			resetTimeout: breakerResetTimeout,
		},
		logger: logger,
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

	if s.breaker != nil && !s.breaker.allowRequest() {
		if cached := s.readStaleCache(); cached != nil {
			return cached, nil
		}
		return nil, ErrCBRCircuitOpen
	}

	keyRate, err := s.cbrClient.GetKeyRate(ctx)
	if err != nil {
		if s.breaker != nil {
			s.breaker.recordFailure()
		}

		// If the CBR integration is temporarily unavailable, a recent stale value
		// is safer than failing credit calculations for every request.
		if cached := s.readStaleCache(); cached != nil {
			if s.logger != nil {
				s.logger.WithError(err).Warn("cbr unavailable, using stale cached key rate")
			}
			return cached, nil
		}

		return nil, err
	}

	if s.breaker != nil {
		s.breaker.recordSuccess()
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

func (b *cbrCircuitBreaker) allowRequest() bool {
	if b == nil || b.failureLimit <= 0 || b.resetTimeout <= 0 {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.failures < b.failureLimit {
		return true
	}

	if time.Since(b.openedAt) >= b.resetTimeout {
		return true
	}

	return false
}

func (b *cbrCircuitBreaker) recordFailure() {
	if b == nil || b.failureLimit <= 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures++
	if b.failures >= b.failureLimit {
		b.openedAt = time.Now()
	}
}

func (b *cbrCircuitBreaker) recordSuccess() {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.openedAt = time.Time{}
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
