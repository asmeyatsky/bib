package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/card-service/internal/application/usecase"
	"github.com/bibbank/bib/services/card-service/internal/domain/service"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/adapter"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/messaging"
	"github.com/bibbank/bib/services/card-service/internal/infrastructure/persistence"
	grpcpresentation "github.com/bibbank/bib/services/card-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/card-service/internal/presentation/rest"
)

func main() {
	// Logger setup.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting card-service")

	// Load configuration.
	cfg := config.Load()

	// Database connection pool.
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Infrastructure adapters.
	cardRepo := persistence.NewPostgresCardRepository(pool)
	eventPublisher := messaging.NewKafkaEventPublisher(cfg.KafkaBroker, "card-events", logger)
	cardProcessor := adapter.NewStubCardProcessor(logger)
	balanceClient := adapter.NewStubAccountBalanceClient(logger, decimal.NewFromInt(100000)) // Default 100k balance.

	// Domain services.
	jitFundingService := service.NewJITFundingService()

	// Application use cases.
	issueCardUC := usecase.NewIssueCardUseCase(cardRepo, eventPublisher, cardProcessor)
	authorizeUC := usecase.NewAuthorizeTransactionUseCase(cardRepo, eventPublisher, balanceClient, jitFundingService)
	getCardUC := usecase.NewGetCardUseCase(cardRepo)
	freezeCardUC := usecase.NewFreezeCardUseCase(cardRepo, eventPublisher)

	// gRPC handler and server.
	grpcHandler := grpcpresentation.NewCardServiceHandler(issueCardUC, authorizeUC, getCardUC, freezeCardUC)
	grpcServer := grpcpresentation.NewServer(grpcHandler, logger)

	// HTTP health server.
	healthHandler := rest.NewHealthHandler(logger)
	httpMux := http.NewServeMux()
	healthHandler.RegisterRoutes(httpMux)

	// Start gRPC server.
	go func() {
		if err := grpcServer.Start(cfg.GRPCAddr()); err != nil {
			logger.Error("gRPC server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Start HTTP health server.
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr(),
		Handler: httpMux,
	}
	go func() {
		logger.Info("HTTP health server starting", slog.String("addr", cfg.HTTPAddr()))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("card-service is running",
		slog.String("grpc_addr", cfg.GRPCAddr()),
		slog.String("http_addr", cfg.HTTPAddr()),
	)

	// Graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	logger.Info("received shutdown signal", slog.String("signal", sig.String()))

	grpcServer.Stop()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("card-service stopped")
}
