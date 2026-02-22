package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bibbank/bib/gateway/internal/proxy"
)

// testProxies creates a Proxies struct with nil-connection proxies for testing.
// The health and ledger/account routes that don't need a backend are testable.
func testProxies() *Proxies {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	// Create proxies with nil connections. Health routes don't use proxies.
	return &Proxies{
		Account:   proxy.NewAccountProxy(nil, logger),
		Ledger:    proxy.NewLedgerProxy(nil, logger),
		Payment:   proxy.NewPaymentProxy(nil, logger),
		FX:        proxy.NewFXProxy(nil, logger),
		Identity:  proxy.NewIdentityProxy(nil, logger),
		Deposit:   proxy.NewDepositProxy(nil, logger),
		Card:      proxy.NewCardProxy(nil, logger),
		Lending:   proxy.NewLendingProxy(nil, logger),
		Fraud:     proxy.NewFraudProxy(nil, logger),
		Reporting: proxy.NewReportingProxy(nil, logger),
	}
}

func TestHealthz(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testProxies())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestReadyz(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testProxies())

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf("expected status ready, got %q", body["status"])
	}
}

func TestHealthz_ContentType(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testProxies())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}
