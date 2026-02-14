package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/pkg/observability"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/messaging"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/persistence/postgres"
	grpcpresentation "github.com/bibbank/bib/services/fraud-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/fraud-service/internal/presentation/rest"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration.
	cfg := config.Load()

	// Initialize structured logger via shared observability package.
	logger := observability.InitLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: "json",
	})
	slog.SetDefault(logger)

	logger.Info("starting fraud-service",
		"http_port", cfg.HTTPPort,
		"grpc_port", cfg.GRPCPort,
	)

	// Initialize tracing.
	shutdown, err := observability.InitTracer(ctx, observability.TracingConfig{
		ServiceName: "fraud-service",
		Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		Insecure:    true,
	})
	if err != nil {
		logger.Warn("failed to initialize tracer, continuing without tracing", "error", err)
	} else {
		defer shutdown(ctx)
	}

	// Database connection.
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()

	pool, err := pgxpool.New(dbCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(dbCtx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Wire infrastructure adapters.
	assessmentRepo := postgres.NewAssessmentRepository(pool)
	eventPublisher := messaging.NewKafkaPublisher(
		[]string{cfg.KafkaBroker},
		"fraud.events",
		logger,
	)

	// Wire domain services.
	riskScorer := service.NewRiskScorer()

	// Wire use cases.
	assessTransactionUC := usecase.NewAssessTransaction(assessmentRepo, eventPublisher, riskScorer)
	getAssessmentUC := usecase.NewGetAssessment(assessmentRepo)

	// gRPC server.
	grpcHandler := grpcpresentation.NewFraudServiceHandler(assessTransactionUC, getAssessmentUC, logger)
	grpcServer := grpcpresentation.NewServer(grpcHandler, cfg.GRPCAddress(), logger)

	// HTTP server (health checks).
	healthHandler := rest.NewHealthHandler(logger)
	httpMux := http.NewServeMux()
	healthHandler.RegisterRoutes(httpMux)

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddress(),
		Handler:      httpMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start servers.
	errCh := make(chan error, 2)

	go func() {
		if err := grpcServer.Start(); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		logger.Info("HTTP server starting", "address", cfg.HTTPAddress())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	logger.Info("fraud-service started",
		"grpc_address", cfg.GRPCAddress(),
		"http_address", cfg.HTTPAddress(),
		"environment", cfg.Environment,
	)

	// Wait for shutdown signal.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown.
	logger.Info("shutting down fraud-service")

	grpcServer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("fraud-service stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
