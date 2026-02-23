package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/pkg/observability"
	pgpkg "github.com/bibbank/bib/pkg/postgres"
	kafkapkg "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/services/deposit-service/internal/application/usecase"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/service"
	"github.com/bibbank/bib/services/deposit-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/deposit-service/internal/infrastructure/kafka"
	infraPG "github.com/bibbank/bib/services/deposit-service/internal/infrastructure/postgres"
	grpcPresentation "github.com/bibbank/bib/services/deposit-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/deposit-service/internal/presentation/rest"
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

	logger.Info("starting deposit-service",
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
	if err := pgpkg.RunMigrations(dsn, "file://internal/infrastructure/persistence/migration"); err != nil {
		logger.Warn("migration warning", "error", err)
	}

	// Initialize Kafka producer
	producer := kafkapkg.NewProducer(kafkapkg.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer producer.Close()

	// Wire dependencies (DI via constructors)
	productRepo := infraPG.NewProductRepo(pool)
	positionRepo := infraPG.NewPositionRepo(pool)
	publisher := kafka.NewPublisher(producer)
	accrualEngine := service.NewAccrualEngine()

	// Use cases
	createProductUC := usecase.NewCreateDepositProduct(productRepo)
	openPositionUC := usecase.NewOpenDepositPosition(productRepo, positionRepo, publisher)
	getPositionUC := usecase.NewGetDepositPosition(positionRepo)
	accrueInterestUC := usecase.NewAccrueInterest(productRepo, positionRepo, publisher, accrualEngine)

	// JWT service (validation-only: public key preferred, secret as fallback).
	jwtCfg := auth.JWTConfig{
		Issuer: "bib-deposit",
	}
	switch {
	case os.Getenv("JWT_PUBLIC_KEY") != "":
		jwtCfg.PublicKeyPEM = os.Getenv("JWT_PUBLIC_KEY")
	case os.Getenv("JWT_PUBLIC_KEY_FILE") != "":
		keyData, err := auth.LoadKeyFromFile(os.Getenv("JWT_PUBLIC_KEY_FILE"))
		if err != nil {
			logger.Error("failed to load JWT public key file", "error", err)
			os.Exit(1)
		}
		jwtCfg.PublicKeyPEM = string(keyData)
	default:
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "dev-secret-change-in-prod" // development only
		}
		jwtCfg.Secret = jwtSecret
	}
	jwtSvc, err := auth.NewJWTService(jwtCfg)
	if err != nil {
		logger.Error("failed to initialize JWT service", "error", err)
		os.Exit(1)
	}

	// gRPC server
	handler := grpcPresentation.NewDepositHandler(createProductUC, openPositionUC, getPositionUC, accrueInterestUC)
	grpcServer := grpcPresentation.NewServer(handler, cfg.GRPCPort, logger, jwtSvc)

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
	logger.Info("deposit-service stopped")
}
