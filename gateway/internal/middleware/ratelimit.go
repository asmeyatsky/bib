package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/bibbank/bib/pkg/auth"
)

// RateLimiter implements a simple token bucket rate limiter.
type RateLimiter struct {
	lastRefill time.Time
	tokens     float64
	maxTokens  float64
	refillRate float64
	mu         sync.Mutex
}

// NewRateLimiter creates a rate limiter that allows rps requests per second.
func NewRateLimiter(rps int) *RateLimiter {
	return &RateLimiter{
		tokens:     float64(rps),
		maxTokens:  float64(rps),
		refillRate: float64(rps),
		lastRefill: time.Now(),
	}
}

// Allow reports whether a single request is permitted.
// It consumes one token if available.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

// RateLimitMiddleware applies global rate limiting to incoming HTTP requests.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PerClientRateLimiter maintains per-client token bucket rate limiters.
// Clients are identified by tenant ID from JWT claims, falling back to
// the remote IP address for unauthenticated requests.
type PerClientRateLimiter struct {
	limiters map[string]*RateLimiter
	rps      int
	mu       sync.Mutex
}

// NewPerClientRateLimiter creates a per-client rate limiter.
func NewPerClientRateLimiter(rps int) *PerClientRateLimiter {
	return &PerClientRateLimiter{
		limiters: make(map[string]*RateLimiter),
		rps:      rps,
	}
}

// getLimiter returns (or creates) the rate limiter for a given client key.
func (pcrl *PerClientRateLimiter) getLimiter(key string) *RateLimiter {
	pcrl.mu.Lock()
	defer pcrl.mu.Unlock()

	if rl, ok := pcrl.limiters[key]; ok {
		return rl
	}
	rl := NewRateLimiter(pcrl.rps)
	pcrl.limiters[key] = rl
	return rl
}

// Allow checks if a request from the identified client is allowed.
func (pcrl *PerClientRateLimiter) Allow(key string) bool {
	return pcrl.getLimiter(key).Allow()
}

// clientKey extracts a per-client key from the request. It uses the tenant
// ID from JWT claims if available, otherwise falls back to the client IP.
func clientKey(r *http.Request) string {
	// Try JWT claims first.
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		return "tenant:" + claims.TenantID.String()
	}

	// Fall back to IP address.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + ip
}

// PerClientRateLimitMiddleware applies per-client rate limiting.
func PerClientRateLimitMiddleware(limiter *PerClientRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientKey(r)
			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
