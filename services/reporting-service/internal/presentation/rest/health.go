package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthHandler handles HTTP health check endpoints.
type HealthHandler struct {
	logger *slog.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

// RegisterRoutes registers health check routes on the given mux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
}

func (h *HealthHandler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]string{
		"status":  "healthy",
		"service": "reporting-service",
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to write health response", "error", err)
	}
}

func (h *HealthHandler) readyz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]string{
		"status":  "ready",
		"service": "reporting-service",
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to write readiness response", "error", err)
	}
}
