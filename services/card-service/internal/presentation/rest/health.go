package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthHandler provides HTTP health check endpoints for the card-service.
type HealthHandler struct {
	logger *slog.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

// healthResponse represents the health check response body.
type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// RegisterRoutes registers the health check routes on the given mux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.Health)
	mux.HandleFunc("/readyz", h.Ready)
}

// Health is the liveness probe endpoint.
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := healthResponse{
		Status:  "UP",
		Service: "card-service",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode health response", slog.String("error", err.Error()))
	}
}

// Ready is the readiness probe endpoint.
func (h *HealthHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := healthResponse{
		Status:  "READY",
		Service: "card-service",
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode ready response", slog.String("error", err.Error()))
	}
}
