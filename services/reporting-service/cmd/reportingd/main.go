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

	"github.com/bibbank/bib/pkg/auth"
	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/pkg/observability"
	"github.com/bibbank/bib/services/reporting-service/internal/application/usecase"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
	"github.com/bibbank/bib/services/reporting-service/internal/infrastructure/client"
	"github.com/bibbank/bib/services/reporting-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/reporting-service/internal/infrastructure/messaging"
	pgRepo "github.com/bibbank/bib/services/reporting-service/internal/infrastructure/persistence/postgres"
	grpcpresentation "github.com/bibbank/bib/services/reporting-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/reporting-service/internal/presentation/rest"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration.
	cfg := config.Load()

	// Initialize structured logger via shared observability package.
	logger := observability.InitLogger(observability.LogConfig{
		Level:  "info",
		Format: "json",
	})
	slog.SetDefault(logger)

	logger.Info("starting reporting-service",
		"http_port", cfg.HTTPPort,
		"grpc_port", cfg.GRPCPort,
	)

	// Initialize tracing.
	shutdown, err := observability.InitTracer(ctx, observability.TracingConfig{
		ServiceName: cfg.ServiceName,
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
	reportRepo := pgRepo.NewReportSubmissionRepo(pool)
	kafkaProducer := pkgkafka.NewProducer(pkgkafka.Config{
		Brokers: []string{cfg.KafkaBroker},
	})
	defer kafkaProducer.Close()
	eventPublisher := messaging.NewKafkaPublisher(kafkaProducer, logger)
	ledgerClient := client.NewStubLedgerDataClient()
	xbrlGenerator := service.NewXBRLGenerator()

	// Wire use cases.
	generateReportUC := usecase.NewGenerateReportUseCase(reportRepo, eventPublisher, ledgerClient, xbrlGenerator)
	getReportUC := usecase.NewGetReportUseCase(reportRepo)
	submitReportUC := usecase.NewSubmitReportUseCase(reportRepo, eventPublisher)

	// JWT service.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-prod" // development only
	}
	jwtSvc := auth.NewJWTService(auth.JWTConfig{
		Secret: jwtSecret,
		Issuer: "bib-reporting",
	})

	// gRPC server.
	handler := grpcpresentation.NewReportingHandler(generateReportUC, getReportUC, submitReportUC)
	grpcServer := grpcpresentation.NewServer(handler, logger, jwtSvc)

	// HTTP server (health checks).
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
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown.
	logger.Info("shutting down servers")

	grpcServer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("reporting-service stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
