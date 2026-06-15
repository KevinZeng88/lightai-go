package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// LoginRateLimiter limits login attempts by username and source IP.
type LoginRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int
	// lastCleanup tracks when stale entries were last pruned.
	lastCleanup time.Time
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

// NewLoginRateLimiter creates a new login rate limiter.
// rate is the sustained rate (e.g., 1 per second).
// burst is the max burst size (e.g., 5).
func NewLoginRateLimiter(r rate.Limit, burst int) *LoginRateLimiter {
	return &LoginRateLimiter{
		limiters:    make(map[string]*rateLimiterEntry),
		rate:        r,
		burst:       burst,
		lastCleanup: time.Now(),
	}
}

// Allow checks if a login attempt from the given request is allowed.
// It rate-limits by username and by source IP.
func (l *LoginRateLimiter) Allow(r *http.Request, username string) bool {
	// Rate limit by username.
	if !l.checkKey("user:" + strings.ToLower(username)) {
		return false
	}
	// Rate limit by IP.
	ip := clientIP(r)
	if !l.checkKey("ip:" + ip) {
		return false
	}
	return true
}

func (l *LoginRateLimiter) checkKey(key string) bool {
	l.mu.Lock()
	entry, ok := l.limiters[key]
	if !ok {
		entry = &rateLimiterEntry{
			limiter:  rate.NewLimiter(l.rate, l.burst),
			lastUsed: time.Now(),
		}
		l.limiters[key] = entry
	}
	entry.lastUsed = time.Now()

	// Periodically evict stale entries: entries older than 1 hour are removed.
	// Runs at most once per minute to avoid excessive scanning.
	if time.Since(l.lastCleanup) > time.Minute {
		for k, e := range l.limiters {
			if time.Since(e.lastUsed) > time.Hour {
				delete(l.limiters, k)
			}
		}
		l.lastCleanup = time.Now()
	}
	l.mu.Unlock()
	return entry.limiter.Allow()
}

func clientIP(r *http.Request) string {
	// Check X-Forwarded-For header first.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Fall back to X-Real-IP.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Default to RemoteAddr.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
