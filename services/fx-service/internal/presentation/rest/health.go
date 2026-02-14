package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler provides HTTP health check endpoints.
type HealthHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(pool *pgxpool.Pool, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{pool: pool, logger: logger}
}

// healthResponse is the JSON body returned by health endpoints.
type healthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// LivenessHandler returns 200 if the process is alive.
func (h *HealthHandler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := healthResponse{
			Status:    "UP",
			Service:   "fx-service",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// ReadinessHandler returns 200 if the service is ready to accept traffic.
// It checks the database connection.
func (h *HealthHandler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.pool.Ping(ctx); err != nil {
			checks["postgres"] = fmt.Sprintf("DOWN: %v", err)
			resp := healthResponse{
				Status:    "DOWN",
				Service:   "fx-service",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Checks:    checks,
			}
			h.logger.Warn("readiness check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, resp)
			return
		}
		checks["postgres"] = "UP"

		resp := healthResponse{
			Status:    "UP",
			Service:   "fx-service",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checks,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// RegisterRoutes registers the health check routes on the provided mux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.LivenessHandler())
	mux.HandleFunc("GET /readyz", h.ReadinessHandler())
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
