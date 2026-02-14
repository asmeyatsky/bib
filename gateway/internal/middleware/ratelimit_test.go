package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(5)

	// Should allow up to 5 requests immediately (burst).
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}

	// 6th request should be denied.
	if rl.Allow() {
		t.Fatal("6th request should have been denied")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(10)

	// Drain all tokens.
	for i := 0; i < 10; i++ {
		rl.Allow()
	}

	if rl.Allow() {
		t.Fatal("should be denied after draining tokens")
	}

	// Simulate time passing for refill.
	rl.mu.Lock()
	rl.lastRefill = time.Now().Add(-1 * time.Second)
	rl.mu.Unlock()

	// After 1 second, should have ~10 tokens again.
	if !rl.Allow() {
		t.Fatal("should be allowed after refill period")
	}
}

func TestRateLimiter_MaxTokensCapped(t *testing.T) {
	rl := NewRateLimiter(5)

	// Simulate lots of time passing.
	rl.mu.Lock()
	rl.lastRefill = time.Now().Add(-10 * time.Second)
	rl.mu.Unlock()

	// Should allow 5 requests (capped at maxTokens), not 50.
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected 5 allowed requests (capped at max), got %d", allowed)
	}
}

func TestRateLimitMiddleware_Rejects(t *testing.T) {
	rl := NewRateLimiter(1)

	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Second request should be rate limited.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}
