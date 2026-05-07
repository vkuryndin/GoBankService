package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"bank-service/internal/audit"
)

type RateLimitRule struct {
	Name   string
	Limit  int
	Window time.Duration
	Match  func(r *http.Request) bool
	Key    func(r *http.Request) string
}

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	enabled         bool
	rules           []RateLimitRule
	cleanupInterval time.Duration
	auditRecorder   audit.Recorder
	mu              sync.Mutex
	entries         map[string]rateLimitEntry
}

func NewRateLimiter(
	enabled bool,
	rules []RateLimitRule,
	cleanupInterval time.Duration,
	auditRecorder audit.Recorder,
) func(http.Handler) http.Handler {
	limiter := &rateLimiter{
		enabled:         enabled,
		rules:           rules,
		cleanupInterval: cleanupInterval,
		auditRecorder:   auditRecorder,
		entries:         make(map[string]rateLimitEntry),
	}

	if enabled && cleanupInterval > 0 {
		go limiter.cleanupLoop()
	}

	return limiter.middleware
}

func (l *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.enabled {
			next.ServeHTTP(w, r)
			return
		}

		rule, ok := l.matchRule(r)
		if !ok || rule.Limit <= 0 || rule.Window <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		key := strings.TrimSpace(rule.Key(r))
		if key == "" {
			key = ClientIP(r)
		}

		allowed, retryAfter := l.allow(rule, key)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			l.recordBlockedRequest(r, rule, retryAfter)
			writeMiddlewareError(w, http.StatusTooManyRequests, "too many requests")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *rateLimiter) recordBlockedRequest(r *http.Request, rule RateLimitRule, retryAfter time.Duration) {
	if l.auditRecorder == nil {
		return
	}

	var userID *int64
	if value, ok := GetUserIDFromContext(r.Context()); ok {
		userID = audit.Int64Ptr(value)
	}

	l.auditRecorder.Record(context.Background(), audit.Event{
		UserID:    userID,
		Action:    "security.rate_limit.blocked",
		Status:    audit.StatusBlocked,
		IPAddress: ClientIP(r),
		UserAgent: r.UserAgent(),
		Details: map[string]any{
			"request_id":          RequestIDFromContext(r.Context()),
			"rule":                rule.Name,
			"method":              r.Method,
			"path":                r.URL.Path,
			"retry_after_seconds": int(retryAfter.Seconds()),
		},
	})
}

func (l *rateLimiter) matchRule(r *http.Request) (RateLimitRule, bool) {
	for _, rule := range l.rules {
		if rule.Match != nil && rule.Match(r) {
			return rule, true
		}
	}

	return RateLimitRule{}, false
}

func (l *rateLimiter) allow(rule RateLimitRule, key string) (bool, time.Duration) {
	now := time.Now()
	storageKey := rule.Name + ":" + key

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.entries[storageKey]
	if !exists || now.Sub(entry.windowStart) >= rule.Window {
		l.entries[storageKey] = rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return true, 0
	}

	if entry.count >= rule.Limit {
		return false, rule.Window - now.Sub(entry.windowStart)
	}

	entry.count++
	l.entries[storageKey] = entry

	return true, 0
}

func (l *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

func (l *rateLimiter) cleanup() {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, entry := range l.entries {
		if now.Sub(entry.windowStart) > 2*l.cleanupInterval {
			delete(l.entries, key)
		}
	}
}

func MatchMethodPath(method string, path string) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return r.Method == method && r.URL.Path == path
	}
}

func MatchPathPrefix(prefix string) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, prefix)
	}
}

func MatchAny() func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return true
	}
}

func ClientIPKey(r *http.Request) string {
	return ClientIP(r)
}

func UserIDKey(r *http.Request) string {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		return ClientIP(r)
	}

	return "user:" + strconv.FormatInt(userID, 10)
}

func ClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		candidate := strings.TrimSpace(parts[0])
		if candidate != "" {
			return candidate
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}
