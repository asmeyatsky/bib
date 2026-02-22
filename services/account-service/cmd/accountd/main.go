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
	"github.com/bibbank/bib/services/account-service/internal/application/usecase"
	"github.com/bibbank/bib/services/account-service/internal/infrastructure/config"
	infraKafka "github.com/bibbank/bib/services/account-service/internal/infrastructure/kafka"
	grpcPresentation "github.com/bibbank/bib/services/account-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/account-service/internal/presentation/rest"
	infraPostgres "github.com/bibbank/bib/services/account-service/internal/infrastructure/postgres"
)

func main() {
	// Initialize structured logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting account service")

	// Load configuration.
	cfg := config.Load()

	// Initialize database connection pool.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to parse database config", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connection.
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database", "database", cfg.Database.Database)

	// Initialize infrastructure adapters.
	accountRepo := infraPostgres.NewAccountRepository(pool)
	kafkaProducer := pkgkafka.NewProducer(pkgkafka.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer kafkaProducer.Close()
	eventPublisher := infraKafka.NewPublisher(kafkaProducer, logger)

	// Initialize use cases.
	// LedgerClient is nil for now; will be integrated when ledger service is available.
	openAccountUC := usecase.NewOpenAccountUseCase(accountRepo, eventPublisher, nil, logger)
	getAccountUC := usecase.NewGetAccountUseCase(accountRepo, logger)
	freezeAccountUC := usecase.NewFreezeAccountUseCase(accountRepo, eventPublisher, logger)
	closeAccountUC := usecase.NewCloseAccountUseCase(accountRepo, eventPublisher, logger)
	listAccountsUC := usecase.NewListAccountsUseCase(accountRepo, logger)

	// JWT service.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-prod" // development only
	}
	jwtSvc := auth.NewJWTService(auth.JWTConfig{
		Secret: jwtSecret,
		Issuer: "bib-account",
	})

	// Initialize gRPC handler and server.
	handler := grpcPresentation.NewAccountHandler(
		openAccountUC,
		getAccountUC,
		freezeAccountUC,
		closeAccountUC,
		listAccountsUC,
	)
	grpcServer := grpcPresentation.NewServer(handler, cfg.GRPCPort, logger, jwtSvc)

	// Initialize HTTP health server.
	healthHandler := rest.NewHealthHandler(cfg.ServiceName, logger)
	httpMux := http.NewServeMux()
	healthHandler.RegisterRoutes(httpMux)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: httpMux,
	}

	// Start servers in goroutines.
	errCh := make(chan error, 2)

	go func() {
		if err := grpcServer.Start(); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		logger.Info("HTTP health server starting", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown.
	logger.Info("shutting down servers")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	grpcServer.Stop()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown HTTP server", "error", err)
	}

	logger.Info("account service stopped")
}
