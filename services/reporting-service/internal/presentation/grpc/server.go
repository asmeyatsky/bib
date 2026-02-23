package grpc

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/pkg/tlsutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server with reporting service handlers.
type Server struct {
	grpcServer *grpc.Server
	handler    *ReportingHandler
	logger     *slog.Logger
}

// NewServer creates a new gRPC server.
func NewServer(handler *ReportingHandler, logger *slog.Logger, jwtService *auth.JWTService) *Server {
	// Add auth interceptor, skipping health check methods.
	authInterceptor := auth.UnaryAuthInterceptor(jwtService, []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	})

	var serverOpts []grpc.ServerOption
	serverOpts = append(serverOpts, grpc.UnaryInterceptor(authInterceptor))

	// Optional TLS: set GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE to enable.
	if certFile, keyFile := os.Getenv("GRPC_TLS_CERT_FILE"), os.Getenv("GRPC_TLS_KEY_FILE"); certFile != "" && keyFile != "" {
		creds, err := tlsutil.ServerTLSConfig(certFile, keyFile)
		if err != nil {
			logger.Error("failed to load TLS credentials, starting without TLS", "error", err)
		} else {
			serverOpts = append(serverOpts, grpc.Creds(creds))
			logger.Info("gRPC TLS enabled", "cert", certFile, "key", keyFile)
		}
	} else {
		logger.Info("gRPC TLS not configured, running without TLS")
	}

	grpcServer := grpc.NewServer(serverOpts...)

	// Register health check.
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("reporting-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Only enable reflection when GRPC_REFLECTION=true.
	if os.Getenv("GRPC_REFLECTION") == "true" {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpcServer: grpcServer,
		handler:    handler,
		logger:     logger,
	}
}

// Start begins listening on the given address.
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Info("gRPC server starting", "address", addr)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}
