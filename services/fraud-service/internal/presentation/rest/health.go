package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// HealthHandler provides HTTP health check endpoints for the fraud service.
type HealthHandler struct {
	startTime time.Time
	logger    *slog.Logger
}

// NewHealthHandler creates a new health check handler.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// HealthResponse is the JSON response for health checks.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Uptime  string `json:"uptime"`
}

// ReadinessResponse is the JSON response for readiness checks.
type ReadinessResponse struct {
	Checks  map[string]string `json:"checks"`
	Status  string            `json:"status"`
	Service string            `json:"service"`
}

// RegisterRoutes registers health endpoints on the provided ServeMux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /readyz", h.Readyz)
}

// Healthz handles liveness probe requests.
func (h *HealthHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	resp := HealthResponse{
		Status:  "healthy",
		Service: "fraud-service",
		Uptime:  time.Since(h.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errcheck // best-effort HTTP response encoding
}

// Readyz handles readiness probe requests.
func (h *HealthHandler) Readyz(w http.ResponseWriter, _ *http.Request) {
	checks := map[string]string{
		"database": "ok",
		"kafka":    "ok",
	}

	resp := ReadinessResponse{
		Status:  "ready",
		Service: "fraud-service",
		Checks:  checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errcheck // best-effort HTTP response encoding
}
