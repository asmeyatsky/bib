package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/bibbank/bib/gateway/internal/config"
	"github.com/bibbank/bib/gateway/internal/handler"
	"github.com/bibbank/bib/gateway/internal/middleware"
	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/pkg/observability"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.Load()

	logger := observability.InitLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	slog.SetDefault(logger)

	logger.Info("starting gateway", "port", cfg.HTTPPort)

	// JWT service for token validation.
	jwtService := auth.NewJWTService(auth.JWTConfig{
		Secret: cfg.JWTSecret,
		Issuer: "bib-gateway",
	})

	// Rate limiter.
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit)

	// Routes.
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Build middleware chain (applied in reverse order).
	var h http.Handler = mux
	h = middleware.LoggingMiddleware(logger)(h)
	h = middleware.RateLimitMiddleware(rateLimiter)(h)
	h = middleware.AuthMiddleware(jwtService, []string{"/healthz", "/readyz"})(h)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: h,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("gateway stopped")
}
