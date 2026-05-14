package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type TokenRevocationChecker interface {
	IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error)
}

type cachedRevocationEntry struct {
	revoked   bool
	expiresAt time.Time
}

type CachedTokenRevocationChecker struct {
	store  TokenRevocationChecker
	ttl    time.Duration
	logger *logrus.Logger
	mu     sync.RWMutex
	cache  map[string]cachedRevocationEntry
}

func NewCachedTokenRevocationChecker(
	store TokenRevocationChecker,
	ttl time.Duration,
	logger *logrus.Logger,
) *CachedTokenRevocationChecker {
	return &CachedTokenRevocationChecker{
		store:  store,
		ttl:    ttl,
		logger: logger,
		cache:  make(map[string]cachedRevocationEntry),
	}
}

func (c *CachedTokenRevocationChecker) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	if c == nil || c.store == nil {
		return false, nil
	}

	if c.ttl <= 0 {
		return c.store.IsTokenRevoked(ctx, tokenHash)
	}

	now := time.Now()
	if revoked, ok := c.read(tokenHash, now); ok {
		return revoked, nil
	}

	revoked, err := c.store.IsTokenRevoked(ctx, tokenHash)
	if err != nil {
		return false, err
	}

	c.write(tokenHash, revoked, now.Add(c.ttl))
	return revoked, nil
}

func (c *CachedTokenRevocationChecker) read(tokenHash string, now time.Time) (bool, bool) {
	c.mu.RLock()
	entry, ok := c.cache[tokenHash]
	c.mu.RUnlock()

	if !ok {
		return false, false
	}

	if now.After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.cache, tokenHash)
		c.mu.Unlock()
		return false, false
	}

	return entry.revoked, true
}

func (c *CachedTokenRevocationChecker) write(tokenHash string, revoked bool, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[tokenHash] = cachedRevocationEntry{revoked: revoked, expiresAt: expiresAt}
}
