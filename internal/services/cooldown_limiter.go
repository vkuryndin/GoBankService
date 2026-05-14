package services

import (
	"sync"
	"time"
)

type cooldownLimiter struct {
	cooldown time.Duration
	mu       sync.Mutex
	next     map[string]time.Time
}

func newCooldownLimiter(cooldown time.Duration) *cooldownLimiter {
	return &cooldownLimiter{
		cooldown: cooldown,
		next:     make(map[string]time.Time),
	}
}

func (l *cooldownLimiter) allow(key string) bool {
	if l == nil || l.cooldown <= 0 {
		return true
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if allowedAt, exists := l.next[key]; exists && now.Before(allowedAt) {
		return false
	}

	l.next[key] = now.Add(l.cooldown)
	l.cleanup(now)
	return true
}

func (l *cooldownLimiter) cleanup(now time.Time) {
	for key, allowedAt := range l.next {
		if now.After(allowedAt.Add(l.cooldown)) {
			delete(l.next, key)
		}
	}
}
