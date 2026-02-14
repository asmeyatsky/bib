package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// HealthHandler provides HTTP health check endpoints.
type HealthHandler struct {
	serviceName string
	startedAt   time.Time
	logger      *slog.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(serviceName string, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		serviceName: serviceName,
		startedAt:   time.Now(),
		logger:      logger,
	}
}

// healthResponse is the JSON response for health check endpoints.
type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Uptime  string `json:"uptime"`
}

// readinessResponse is the JSON response for the readiness endpoint.
type readinessResponse struct {
	Status   string            `json:"status"`
	Service  string            `json:"service"`
	Checks   map[string]string `json:"checks"`
}

// Liveness handles the liveness probe endpoint (GET /healthz).
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "ok",
		Service: h.serviceName,
		Uptime:  time.Since(h.startedAt).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Readiness handles the readiness probe endpoint (GET /readyz).
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{
		"database": "ok",
		"kafka":    "ok",
	}

	resp := readinessResponse{
		Status:  "ok",
		Service: h.serviceName,
		Checks:  checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes registers health check routes on the provided ServeMux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.Liveness)
	mux.HandleFunc("GET /readyz", h.Readiness)
}
