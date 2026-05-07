package services

import (
	"sync"
	"time"
)

type attemptLimiter struct {
	maxFailures int
	lockout     time.Duration
	mu          sync.Mutex
	entries     map[string]attemptEntry
}

type attemptEntry struct {
	failures  int
	lockedTil time.Time
}

func newAttemptLimiter(maxFailures int, lockout time.Duration) *attemptLimiter {
	return &attemptLimiter{
		maxFailures: maxFailures,
		lockout:     lockout,
		entries:     make(map[string]attemptEntry),
	}
}

func (l *attemptLimiter) isLocked(key string) bool {
	if l == nil || l.maxFailures <= 0 || l.lockout <= 0 {
		return false
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return false
	}

	if entry.lockedTil.IsZero() {
		return false
	}

	if now.After(entry.lockedTil) {
		delete(l.entries, key)
		return false
	}

	return true
}

func (l *attemptLimiter) recordFailure(key string) {
	if l == nil || l.maxFailures <= 0 || l.lockout <= 0 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[key]
	entry.failures++
	if entry.failures >= l.maxFailures {
		entry.lockedTil = time.Now().Add(l.lockout)
	}

	l.entries[key] = entry
}

func (l *attemptLimiter) reset(key string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, key)
}
