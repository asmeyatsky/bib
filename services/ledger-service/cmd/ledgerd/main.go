package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bibbank/bib/pkg/observability"
	pgpkg "github.com/bibbank/bib/pkg/postgres"
	kafkapkg "github.com/bibbank/bib/pkg/kafka"
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
		defer shutdown(ctx)
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
	if err := pgpkg.RunMigrations(dsn, "internal/infrastructure/postgres/migrations"); err != nil {
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

	// gRPC server
	handler := grpcPresentation.NewLedgerHandler(postEntryUC, getEntryUC, getBalanceUC, listEntriesUC, backvalueUC, periodCloseUC)
	grpcServer := grpcPresentation.NewServer(handler, cfg.GRPCPort, logger)

	// HTTP server (health checks + metrics)
	mux := http.NewServeMux()
	healthHandler := rest.NewHealthHandler()
	healthHandler.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
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
	httpServer.Shutdown(context.Background())
	grpcServer.Stop()
	logger.Info("ledger-service stopped")
}
