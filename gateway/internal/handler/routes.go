package handler

import (
	"encoding/json"
	"net/http"
)

// RegisterRoutes registers all REST API routes on the given ServeMux.
func RegisterRoutes(mux *http.ServeMux) {
	// Health
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/readyz", readyz)

	// API v1 routes - these proxy to gRPC services

	// Ledger
	mux.HandleFunc("POST /api/v1/ledger/entries", notImplemented)
	mux.HandleFunc("GET /api/v1/ledger/entries/{id}", notImplemented)
	mux.HandleFunc("GET /api/v1/ledger/balances/{account_code}", notImplemented)

	// Accounts
	mux.HandleFunc("POST /api/v1/accounts", notImplemented)
	mux.HandleFunc("GET /api/v1/accounts/{id}", notImplemented)
	mux.HandleFunc("POST /api/v1/accounts/{id}/freeze", notImplemented)
	mux.HandleFunc("POST /api/v1/accounts/{id}/close", notImplemented)

	// Payments
	mux.HandleFunc("POST /api/v1/payments", notImplemented)
	mux.HandleFunc("GET /api/v1/payments/{id}", notImplemented)

	// FX
	mux.HandleFunc("GET /api/v1/fx/rates/{pair}", notImplemented)
	mux.HandleFunc("POST /api/v1/fx/convert", notImplemented)

	// Identity
	mux.HandleFunc("POST /api/v1/identity/verifications", notImplemented)
	mux.HandleFunc("GET /api/v1/identity/verifications/{id}", notImplemented)

	// Deposits
	mux.HandleFunc("POST /api/v1/deposits/products", notImplemented)
	mux.HandleFunc("POST /api/v1/deposits/positions", notImplemented)
	mux.HandleFunc("GET /api/v1/deposits/positions/{id}", notImplemented)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not yet implemented - gRPC proxy pending"})
}
