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

	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/messaging"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/persistence/postgres"
	grpcpresentation "github.com/bibbank/bib/services/fraud-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/fraud-service/internal/presentation/rest"
)

func main() {
	// Initialize structured logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting fraud-service")

	// Load configuration.
	cfg := config.Load()

	// Connect to PostgreSQL.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Initialize infrastructure adapters.
	assessmentRepo := postgres.NewAssessmentRepository(pool)
	eventPublisher := messaging.NewKafkaPublisher(
		[]string{cfg.KafkaBroker},
		"fraud.events",
		logger,
	)

	// Initialize domain services.
	riskScorer := service.NewRiskScorer()

	// Initialize use cases.
	assessTransactionUC := usecase.NewAssessTransaction(assessmentRepo, eventPublisher, riskScorer)
	getAssessmentUC := usecase.NewGetAssessment(assessmentRepo)

	// Initialize gRPC handler and server.
	grpcHandler := grpcpresentation.NewFraudServiceHandler(assessTransactionUC, getAssessmentUC, logger)
	grpcServer := grpcpresentation.NewServer(grpcHandler, cfg.GRPCAddress(), logger)

	// Initialize HTTP health server.
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
		logger.Info("HTTP health server starting", slog.String("address", cfg.HTTPAddress()))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	logger.Info("fraud-service started",
		slog.String("grpc_address", cfg.GRPCAddress()),
		slog.String("http_address", cfg.HTTPAddress()),
		slog.String("environment", cfg.Environment),
	)

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	case err := <-errCh:
		logger.Error("server error", slog.String("error", err.Error()))
	}

	// Graceful shutdown.
	logger.Info("shutting down fraud-service")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	grpcServer.Stop()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("fraud-service stopped")
}
