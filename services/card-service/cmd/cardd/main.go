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

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/auth"
	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/pkg/observability"
	pkgpostgres "github.com/bibbank/bib/pkg/postgres"
	"github.com/bibbank/bib/services/card-service/internal/application/usecase"
	"github.com/bibbank/bib/services/card-service/internal/domain/service"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/adapter"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/kafka"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/postgres"
	grpcpresentation "github.com/bibbank/bib/services/card-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/card-service/internal/presentation/rest"
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

	logger.Info("starting card-service",
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
		defer func() { _ = shutdown(ctx) }() //nolint:errcheck
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
	cardRepo := postgres.NewCardRepository(pool)
	kafkaProducer := pkgkafka.NewProducer(pkgkafka.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer kafkaProducer.Close()
	eventPublisher := kafka.NewEventPublisher(kafkaProducer, "card-events", logger)
	cardProcessor := adapter.NewStubCardProcessor(logger)
	balanceClient := adapter.NewStubAccountBalanceClient(logger, decimal.NewFromInt(100000))

	// Wire domain services.
	jitFundingService := service.NewJITFundingService()

	// Wire use cases.
	issueCardUC := usecase.NewIssueCardUseCase(cardRepo, eventPublisher, cardProcessor)
	authorizeUC := usecase.NewAuthorizeTransactionUseCase(cardRepo, eventPublisher, balanceClient, jitFundingService)
	getCardUC := usecase.NewGetCardUseCase(cardRepo)
	freezeCardUC := usecase.NewFreezeCardUseCase(cardRepo, eventPublisher)

	// JWT service for gRPC auth (validation-only: public key preferred, secret as fallback).
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
	grpcHandler := grpcpresentation.NewCardServiceHandler(issueCardUC, authorizeUC, getCardUC, freezeCardUC)
	grpcServer := grpcpresentation.NewServer(grpcHandler, logger, jwtSvc)

	// HTTP server (health checks).
	healthHandler := rest.NewHealthHandler(logger)
	httpMux := http.NewServeMux()
	healthHandler.RegisterRoutes(httpMux)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           httpMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start servers.
	errCh := make(chan error, 2)

	go func() {
		if err := grpcServer.Start(cfg.GRPCAddr()); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		logger.Info("HTTP server starting", "addr", cfg.HTTPAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	logger.Info("card-service is running",
		"grpc_addr", cfg.GRPCAddr(),
		"http_addr", cfg.HTTPAddr(),
	)

	// Wait for shutdown signal.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown.
	grpcServer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("card-service stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
