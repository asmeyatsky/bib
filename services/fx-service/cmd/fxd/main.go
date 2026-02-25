package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/pkg/observability"
	"github.com/bibbank/bib/pkg/postgres"

	"github.com/bibbank/bib/services/fx-service/internal/application/usecase"
	"github.com/bibbank/bib/services/fx-service/internal/domain/port"
	"github.com/bibbank/bib/services/fx-service/internal/domain/service"
	"github.com/bibbank/bib/services/fx-service/internal/infrastructure/config"
	infraKafka "github.com/bibbank/bib/services/fx-service/internal/infrastructure/kafka"
	infraPostgres "github.com/bibbank/bib/services/fx-service/internal/infrastructure/postgres"
	"github.com/bibbank/bib/services/fx-service/internal/infrastructure/provider"
	grpcPresentation "github.com/bibbank/bib/services/fx-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/fx-service/internal/presentation/rest"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fx-service: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration.
	cfg := config.Load()

	// Initialize structured logger.
	logger := observability.InitLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	logger.Info("starting fx-service",
		"grpc_port", cfg.GRPCPort,
		"http_port", cfg.HTTPPort,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database pool.
	pool, err := postgres.NewPool(ctx, postgres.Config{
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
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	logger.Info("database pool created")

	// Run database migrations.
	migDSN := postgres.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		Database: cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
	}.DSN()
	if migErr := postgres.RunMigrations(migDSN, "file://internal/infrastructure/postgres/migrations"); migErr != nil {
		logger.Warn("migration warning", "error", migErr)
	}

	// Kafka producer.
	kafkaProducer := kafka.NewProducer(kafka.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer kafkaProducer.Close()
	logger.Info("kafka producer created")

	// Repositories and infrastructure.
	rateRepo := infraPostgres.NewExchangeRateRepo(pool)
	publisher := infraKafka.NewPublisher(kafkaProducer)

	// Domain services.
	revalEngine := service.NewRevaluationEngine()

	// Rate provider: use static rates when FX_RATE_PROVIDER=static (for dev/CI),
	// otherwise nil (production should wire an HTTP-based external API provider).
	var rateProvider port.RateProvider
	if os.Getenv("FX_RATE_PROVIDER") == "static" {
		rateProvider = provider.NewStaticRateProvider()
		logger.Info("using static rate provider")
	}

	// Use cases.
	getExchangeRate := usecase.NewGetExchangeRate(rateRepo, rateProvider, publisher)
	convertAmount := usecase.NewConvertAmount(rateRepo, rateProvider)
	revaluate := usecase.NewRevaluate(rateRepo, publisher, revalEngine)

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
			return fmt.Errorf("load JWT public key file: %w", loadErr)
		}
		jwtCfg.PublicKeyPEM = string(keyData)
	default:
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "test-e2e-secret" // Match gateway default for E2E tests
		}
		logger.Info("using JWT secret for token validation", "issuer", jwtCfg.Issuer)
		jwtCfg.Secret = jwtSecret
	}
	jwtSvc, err := auth.NewJWTService(jwtCfg)
	if err != nil {
		return fmt.Errorf("initialize JWT service: %w", err)
	}

	// gRPC server.
	handler := grpcPresentation.NewHandler(getExchangeRate, convertAmount, revaluate, logger)
	grpcServer := grpcPresentation.NewServer(handler, logger, cfg.GRPCPort, jwtSvc)

	// HTTP health server.
	healthHandler := rest.NewHealthHandler(pool, logger)
	mux := http.NewServeMux()
	healthHandler.RegisterRoutes(mux)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start servers.
	errCh := make(chan error, 2)

	go func() {
		errCh <- grpcServer.Start()
	}()

	go func() {
		logger.Info("HTTP health server starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	// Graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("received shutdown signal", "signal", sig.String())
	case err := <-errCh:
		logger.Error("server error", "error", err)
		return err
	}

	// Shutdown sequence.
	logger.Info("shutting down")

	grpcServer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	cancel()
	logger.Info("fx-service stopped")
	return nil
}
