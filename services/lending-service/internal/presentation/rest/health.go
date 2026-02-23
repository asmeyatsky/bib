package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthHandler serves liveness and readiness probes over HTTP.
type HealthHandler struct {
	logger *slog.Logger
}

// NewHealthHandler creates a health check HTTP handler.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

// RegisterRoutes attaches health-check routes to the given mux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.liveness)
	mux.HandleFunc("GET /readyz", h.readiness)
}

func (h *HealthHandler) liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "lending-service",
	})
}

func (h *HealthHandler) readiness(w http.ResponseWriter, _ *http.Request) {
	// TODO: check database connectivity, Kafka connectivity, etc.
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ready",
		"service": "lending-service",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errcheck
}
