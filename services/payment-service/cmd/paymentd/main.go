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
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
	"github.com/bibbank/bib/services/payment-service/internal/domain/service"
	"github.com/bibbank/bib/services/payment-service/internal/infrastructure/adapter/ach"
	"github.com/bibbank/bib/services/payment-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/payment-service/internal/infrastructure/messaging"
	infraPG "github.com/bibbank/bib/services/payment-service/internal/infrastructure/persistence/postgres"
	grpcPresentation "github.com/bibbank/bib/services/payment-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/payment-service/internal/presentation/rest"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration.
	cfg := config.Load()

	// Initialize logger.
	logger := observability.InitLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	slog.SetDefault(logger)

	logger.Info("starting payment-service",
		"http_port", cfg.HTTPPort,
		"grpc_port", cfg.GRPCPort,
	)

	// Initialize tracing.
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

	// Initialize database.
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

	// Run migrations.
	dsn := pgpkg.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		Database: cfg.DB.Name,
		SSLMode:  cfg.DB.SSLMode,
	}.DSN()
	if err := pgpkg.RunMigrations(dsn, "internal/infrastructure/persistence/postgres/migrations"); err != nil {
		logger.Warn("migration warning", "error", err)
	}

	// Initialize Kafka producer.
	producer := kafkapkg.NewProducer(kafkapkg.Config{
		Brokers: cfg.Kafka.Brokers,
	})
	defer producer.Close()

	// Wire dependencies (DI via constructors).
	paymentRepo := infraPG.NewPaymentOrderRepo(pool)
	publisher := messaging.NewPublisher(producer)
	routingEngine := service.NewRoutingEngine()
	achAdapter := ach.NewAdapter(logger)

	// Use cases.
	initiatePaymentUC := usecase.NewInitiatePayment(paymentRepo, publisher, routingEngine, nil)
	getPaymentUC := usecase.NewGetPayment(paymentRepo)
	listPaymentsUC := usecase.NewListPayments(paymentRepo)
	_ = usecase.NewProcessPayment(paymentRepo, achAdapter, publisher)

	// gRPC server.
	handler := grpcPresentation.NewPaymentHandler(initiatePaymentUC, getPaymentUC, listPaymentsUC)
	grpcServer := grpcPresentation.NewServer(handler, cfg.GRPCPort, logger)

	// HTTP server (health checks + metrics).
	mux := http.NewServeMux()
	healthHandler := rest.NewHealthHandler()
	healthHandler.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	// Start servers.
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

	// Wait for shutdown.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown.
	httpServer.Shutdown(context.Background())
	grpcServer.Stop()
	logger.Info("payment-service stopped")
}
