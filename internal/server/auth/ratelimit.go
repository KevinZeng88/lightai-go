package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// LoginRateLimiter limits login attempts by username and source IP.
type LoginRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewLoginRateLimiter creates a new login rate limiter.
// rate is the sustained rate (e.g., 1 per second).
// burst is the max burst size (e.g., 5).
func NewLoginRateLimiter(r rate.Limit, burst int) *LoginRateLimiter {
	return &LoginRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
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
	lim, ok := l.limiters[key]
	if !ok {
		lim = rate.NewLimiter(l.rate, l.burst)
		l.limiters[key] = lim
	}
	l.mu.Unlock()
	return lim.Allow()
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
