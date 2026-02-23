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

	"github.com/bibbank/bib/gateway/internal/config"
	"github.com/bibbank/bib/gateway/internal/handler"
	"github.com/bibbank/bib/gateway/internal/middleware"
	"github.com/bibbank/bib/gateway/internal/proxy"
	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/pkg/observability"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.Load()
	cfg.Validate()

	logger := observability.InitLogger(observability.LogConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	slog.SetDefault(logger)

	logger.Info("starting gateway", "port", cfg.HTTPPort)

	// JWT service for token signing (gateway is the issuer).
	jwtCfg := auth.JWTConfig{
		Issuer:     "bib-gateway",
		Expiration: 24 * time.Hour,
	}

	// Prefer RSA private key; fall back to HMAC secret for backwards compat.
	switch {
	case cfg.JWTPrivateKey != "":
		jwtCfg.PrivateKeyPEM = cfg.JWTPrivateKey
	case cfg.JWTPrivateKeyFile != "":
		keyData, err := auth.LoadKeyFromFile(cfg.JWTPrivateKeyFile)
		if err != nil {
			logger.Error("failed to load JWT private key file", "error", err)
			os.Exit(1)
		}
		jwtCfg.PrivateKeyPEM = string(keyData)
	default:
		jwtCfg.Secret = cfg.JWTSecret
	}

	jwtService, err := auth.NewJWTService(jwtCfg)
	if err != nil {
		logger.Error("failed to initialize JWT service", "error", err)
		os.Exit(1)
	}

	// Connect to backend gRPC services.
	proxies, closers, err := dialBackends(cfg, logger)
	if err != nil {
		logger.Error("failed to connect to backend services", "error", err)
		// Continue anyway -- connections are lazy and will retry.
	}
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()

	// Per-client rate limiter.
	rateLimiter := middleware.NewPerClientRateLimiter(cfg.RateLimit)

	// Routes.
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, proxies)

	// Build middleware chain (applied in reverse order).
	var h http.Handler = mux
	h = middleware.LoggingMiddleware(logger)(h)
	h = middleware.PerClientRateLimitMiddleware(rateLimiter)(h)
	h = middleware.AuthMiddleware(jwtService, []string{"/healthz", "/readyz"})(h)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: h,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("gateway stopped")
}

// dialBackends establishes gRPC connections to all backend services.
// Returns the Proxies struct, a slice of connections to close on shutdown,
// and an error if any connection fails (non-fatal, connections are lazy).
func dialBackends(cfg config.Config, logger *slog.Logger) (*handler.Proxies, []*proxy.ServiceConn, error) {
	type svcDef struct {
		name string
		addr string
	}

	defs := []svcDef{
		{"ledger-service", cfg.LedgerAddr},
		{"account-service", cfg.AccountAddr},
		{"fx-service", cfg.FXAddr},
		{"deposit-service", cfg.DepositAddr},
		{"identity-service", cfg.IdentityAddr},
		{"payment-service", cfg.PaymentAddr},
		{"lending-service", cfg.LendingAddr},
		{"fraud-service", cfg.FraudAddr},
		{"card-service", cfg.CardAddr},
		{"reporting-service", cfg.ReportingAddr},
	}

	conns := make(map[string]*proxy.ServiceConn, len(defs))
	var closers []*proxy.ServiceConn
	var firstErr error

	for _, d := range defs {
		conn, err := proxy.Dial(d.name, d.addr, logger)
		if err != nil {
			logger.Error("failed to dial backend", "service", d.name, "addr", d.addr, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		conns[d.name] = conn
		closers = append(closers, conn)
	}

	proxies := &handler.Proxies{
		Ledger:    proxy.NewLedgerProxy(conns["ledger-service"], logger),
		Account:   proxy.NewAccountProxy(conns["account-service"], logger),
		FX:        proxy.NewFXProxy(conns["fx-service"], logger),
		Deposit:   proxy.NewDepositProxy(conns["deposit-service"], logger),
		Identity:  proxy.NewIdentityProxy(conns["identity-service"], logger),
		Payment:   proxy.NewPaymentProxy(conns["payment-service"], logger),
		Lending:   proxy.NewLendingProxy(conns["lending-service"], logger),
		Fraud:     proxy.NewFraudProxy(conns["fraud-service"], logger),
		Card:      proxy.NewCardProxy(conns["card-service"], logger),
		Reporting: proxy.NewReportingProxy(conns["reporting-service"], logger),
	}

	return proxies, closers, firstErr
}
