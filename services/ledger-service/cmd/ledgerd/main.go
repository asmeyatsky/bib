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
	kafkapkg "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/pkg/observability"
	pgpkg "github.com/bibbank/bib/pkg/postgres"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/service"
	"github.com/bibbank/bib/services/ledger-service/internal/infrastructure/config"
	infraKafka "github.com/bibbank/bib/services/ledger-service/internal/infrastructure/kafka"
	infraPG "github.com/bibbank/bib/services/ledger-service/internal/infrastructure/postgres"
	grpcPresentation "github.com/bibbank/bib/services/ledger-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/ledger-service/internal/presentation/rest"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := observability.InitLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	slog.SetDefault(logger)

	logger.Info("starting ledger-service",
		"http_port", cfg.HTTPPort,
		"grpc_port", cfg.GRPCPort,
	)

	// Initialize tracing
	shutdown, err := observability.InitTracer(ctx, observability.TracingConfig{
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.OTLPEndpoint,
		Insecure:    true,
	})
	if err != nil {
		logger.Warn("failed to initialize tracer, continuing without tracing", "error", err)
	} else {
		defer func() { _ = shutdown(ctx) }() //nolint:errcheck // best-effort tracer shutdown
	}

	// Initialize database
	pool, err := pgpkg.NewPool(ctx, pgpkg.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		Database: cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
		MaxConns: cfg.DB.MaxConns,
		MinConns: cfg.DB.MinConns,
	})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Run migrations
	dsn := pgpkg.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		Database: cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
	}.DSN()
	if err = pgpkg.RunMigrations(dsn, "internal/infrastructure/postgres/migrations"); err != nil {
		logger.Warn("migration warning", "error", err)
	}

	// Initialize Kafka producer
	producer := kafkapkg.NewProducer(kafkapkg.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer producer.Close()

	// Wire dependencies (DI via constructors)
	journalRepo := infraPG.NewJournalRepo(pool)
	balanceRepo := infraPG.NewBalanceRepo(pool)
	periodRepo := infraPG.NewFiscalPeriodRepo(pool)
	publisher := infraKafka.NewPublisher(producer)
	validator := service.NewPostingValidator()

	// Use cases
	postEntryUC := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)
	getEntryUC := usecase.NewGetJournalEntry(journalRepo)
	getBalanceUC := usecase.NewGetBalance(balanceRepo)
	listEntriesUC := usecase.NewListJournalEntries(journalRepo)
	backvalueUC := usecase.NewBackvalueEntry(journalRepo)
	periodCloseUC := usecase.NewPeriodClose(periodRepo, publisher)

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

	// gRPC server
	handler := grpcPresentation.NewLedgerHandler(postEntryUC, getEntryUC, getBalanceUC, listEntriesUC, backvalueUC, periodCloseUC)
	grpcServer := grpcPresentation.NewServer(handler, cfg.GRPCPort, logger, jwtSvc)

	// HTTP server (health checks + metrics)
	mux := http.NewServeMux()
	healthHandler := rest.NewHealthHandler()
	healthHandler.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start servers
	errCh := make(chan error, 2)

	go func() {
		errCh <- grpcServer.Start(ctx)
	}()

	go func() {
		logger.Info("HTTP server starting", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for shutdown
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown
	if err := httpServer.Shutdown(context.Background()); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
	grpcServer.Stop()
	logger.Info("ledger-service stopped")
}
