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

func TestPerClientRateLimiter_IsolatesClients(t *testing.T) {
	pcrl := NewPerClientRateLimiter(2)

	// Client A should get 2 requests.
	for i := 0; i < 2; i++ {
		if !pcrl.Allow("client-a") {
			t.Fatalf("client-a request %d should have been allowed", i+1)
		}
	}

	// Client A's 3rd request should be denied.
	if pcrl.Allow("client-a") {
		t.Fatal("client-a 3rd request should have been denied")
	}

	// Client B should still be allowed (separate bucket).
	if !pcrl.Allow("client-b") {
		t.Fatal("client-b should have been allowed (separate bucket)")
	}
}

func TestPerClientRateLimitMiddleware_KeysByIP(t *testing.T) {
	pcrl := NewPerClientRateLimiter(1)

	handler := PerClientRateLimitMiddleware(pcrl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request from 10.0.0.1 should succeed.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 for first request from 10.0.0.1, got %d", rec1.Code)
	}

	// Second request from 10.0.0.1 should be rate limited.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req1)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for second request from 10.0.0.1, got %d", rec2.Code)
	}

	// First request from 10.0.0.2 should succeed (different client).
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "10.0.0.2:12345"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 for first request from 10.0.0.2, got %d", rec3.Code)
	}
}
