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

	"github.com/bibbank/bib/pkg/auth"
	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/pkg/observability"
	pkgpostgres "github.com/bibbank/bib/pkg/postgres"
	"github.com/bibbank/bib/services/fraud-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/service"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/kafka"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/ml"
	"github.com/bibbank/bib/services/fraud-service/internal/infrastructure/postgres"
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
		defer func() { _ = shutdown(ctx) }() //nolint:errcheck // best-effort tracer shutdown
	}

	// Database connection.
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()

	pool, err := pkgpostgres.NewPool(dbCtx, pkgpostgres.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		Database: cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
	})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	// Run database migrations.
	migDSN := pkgpostgres.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		Database: cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
	}.DSN()
	if migErr := pkgpostgres.RunMigrations(migDSN, "file://internal/infrastructure/postgres/migrations"); migErr != nil {
		logger.Warn("migration warning", "error", migErr)
	}

	// Wire infrastructure adapters.
	assessmentRepo := postgres.NewAssessmentRepository(pool)
	kafkaProducer := pkgkafka.NewProducer(pkgkafka.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer kafkaProducer.Close()
	eventPublisher := kafka.NewPublisher(
		kafkaProducer,
		"fraud-events",
		logger,
	)

	// Wire domain services.
	riskScorer := service.NewRiskScorer()

	var scorer service.Scorer = riskScorer
	if getEnv("FRAUD_ML_ENABLED", "false") == "true" {
		mlClient := ml.NewStubModelClient(logger)
		scorer = service.NewHybridScorer(riskScorer, mlClient, 0.3, logger)
		logger.Info("ML-enhanced hybrid scoring enabled")
	}

	// Wire use cases.
	assessTransactionUC := usecase.NewAssessTransaction(assessmentRepo, eventPublisher, scorer)
	getAssessmentUC := usecase.NewGetAssessment(assessmentRepo)

	// JWT service (validation-only: public key preferred, secret as fallback).
	jwtCfg := auth.JWTConfig{
		Issuer: "bib-gateway",
	}
	switch {
	case os.Getenv("JWT_PUBLIC_KEY") != "":
		jwtCfg.PublicKeyPEM = os.Getenv("JWT_PUBLIC_KEY")
	case os.Getenv("JWT_PUBLIC_KEY_FILE") != "":
		keyData, loadErr := auth.LoadKeyFromFile(os.Getenv("JWT_PUBLIC_KEY_FILE"))
		if loadErr != nil {
			logger.Error("failed to load JWT public key file", "error", loadErr)
			os.Exit(1)
		}
		jwtCfg.PublicKeyPEM = string(keyData)
	default:
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "test-e2e-secret" // Match gateway default for E2E tests
		}
		jwtCfg.Secret = jwtSecret
	}
	jwtSvc, err := auth.NewJWTService(jwtCfg)
	if err != nil {
		logger.Error("failed to initialize JWT service", "error", err)
		os.Exit(1)
	}

	// gRPC server.
	grpcHandler := grpcpresentation.NewFraudServiceHandler(assessTransactionUC, getAssessmentUC, logger)
	grpcServer := grpcpresentation.NewServer(grpcHandler, cfg.GRPCAddr(), logger, jwtSvc)

	// HTTP server (health checks).
	healthHandler := rest.NewHealthHandler(logger)
	httpMux := http.NewServeMux()
	healthHandler.RegisterRoutes(httpMux)

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr(),
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
		logger.Info("HTTP server starting", "address", cfg.HTTPAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	logger.Info("fraud-service started",
		"grpc_address", cfg.GRPCAddr(),
		"http_address", cfg.HTTPAddr(),
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
