package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/services/lending-service/internal/application/usecase"
	"github.com/bibbank/bib/services/lending-service/internal/domain/service"
	"github.com/bibbank/bib/services/lending-service/internal/infrastructure/adapter"
	"github.com/bibbank/bib/services/lending-service/internal/infrastructure/config"
	"github.com/bibbank/bib/services/lending-service/internal/infrastructure/messaging"
	pgRepo "github.com/bibbank/bib/services/lending-service/internal/infrastructure/persistence/postgres"
	grpcPresentation "github.com/bibbank/bib/services/lending-service/internal/presentation/grpc"
	"github.com/bibbank/bib/services/lending-service/internal/presentation/rest"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	logger.Info("starting lending service",
		"grpc_port", cfg.GRPCPort,
		"http_port", cfg.HTTPPort,
	)

	// --- Database -----------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// --- Infrastructure adapters -------------------------------------------
	appRepo := pgRepo.NewLoanApplicationRepo(pool)
	loanRepo := pgRepo.NewLoanRepo(pool)
	publisher := messaging.NewKafkaEventPublisher(cfg.KafkaBroker, "lending.events", logger)
	creditClient := adapter.NewStubCreditBureauClient()
	underwriter := service.NewUnderwritingEngine()

	// --- Use cases ----------------------------------------------------------
	submitAppUC := usecase.NewSubmitLoanApplicationUseCase(appRepo, publisher, creditClient, underwriter)
	disburseUC := usecase.NewDisburseLoanUseCase(appRepo, loanRepo, publisher)
	paymentUC := usecase.NewMakePaymentUseCase(loanRepo, publisher)
	getLoanUC := usecase.NewGetLoanUseCase(loanRepo)
	getAppUC := usecase.NewGetApplicationUseCase(appRepo)

	// --- gRPC server --------------------------------------------------------
	handler := grpcPresentation.NewLendingHandler(submitAppUC, disburseUC, paymentUC, getLoanUC, getAppUC)
	grpcServer := grpcPresentation.NewServer(handler, logger)

	go func() {
		if err := grpcServer.Serve(cfg.GRPCAddr()); err != nil {
			logger.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	// --- HTTP health server -------------------------------------------------
	mux := http.NewServeMux()
	healthHandler := rest.NewHealthHandler(logger)
	healthHandler.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr(),
		Handler: mux,
	}

	go func() {
		logger.Info("HTTP health server listening", "addr", cfg.HTTPAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown --------------------------------------------------
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received shutdown signal", "signal", sig.String())

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("lending service stopped")
}
