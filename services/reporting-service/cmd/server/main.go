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

	"github.com/bibbank/bib/services/reporting-service/internal/application/usecase"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
	"github.com/bibbank/bib/services/reporting-service/internal/infrastructure/client"
	"github.com/bibbank/bib/services/reporting-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/reporting-service/internal/infrastructure/messaging"
	grpcpresentation "github.com/bibbank/bib/services/reporting-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/reporting-service/internal/presentation/rest"
)

func main() {
	// Initialize logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration.
	cfg := config.Load()
	logger.Info("starting reporting-service",
		"grpc_port", cfg.GRPCPort,
		"http_port", cfg.HTTPPort,
	)

	// Initialize infrastructure adapters.
	// In production, this would connect to a real PostgreSQL database.
	// repo := postgres.NewReportSubmissionRepo(pool)
	eventPublisher := messaging.NewKafkaPublisher(cfg.KafkaBroker, logger)
	ledgerClient := client.NewStubLedgerDataClient()
	xbrlGenerator := service.NewXBRLGenerator()

	// For development, use an in-memory stub repository.
	// In production, replace with postgres.NewReportSubmissionRepo(pool).
	_ = eventPublisher
	_ = ledgerClient
	_ = xbrlGenerator

	// Initialize use cases.
	// These would be wired to real repositories in production.
	_ = usecase.NewGenerateReportUseCase
	_ = usecase.NewGetReportUseCase
	_ = usecase.NewSubmitReportUseCase

	// Initialize gRPC server.
	handler := grpcpresentation.NewReportingHandler(nil, nil, nil)
	grpcServer := grpcpresentation.NewServer(handler, logger)

	// Initialize HTTP server for health checks.
	httpMux := http.NewServeMux()
	healthHandler := rest.NewHealthHandler(logger)
	healthHandler.RegisterRoutes(httpMux)

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr(),
		Handler:      httpMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start servers.
	errCh := make(chan error, 2)

	go func() {
		if err := grpcServer.Start(cfg.GRPCAddr()); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		logger.Info("HTTP server starting", "address", cfg.HTTPAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown.
	logger.Info("shutting down servers")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grpcServer.Stop()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("reporting-service stopped")
}
