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
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps a gRPC server with the lending handler registered.
type Server struct {
	gs      *grpc.Server
	handler *LendingHandler
	logger  *slog.Logger
}

// NewServer creates and configures the gRPC server.
func NewServer(handler *LendingHandler, logger *slog.Logger, jwtService *auth.JWTService) *Server {
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

	gs := grpc.NewServer(serverOpts...)

	// Register gRPC health check.
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(gs, healthSrv)
	healthSrv.SetServingStatus("lending-service", healthpb.HealthCheckResponse_SERVING)

	// Only enable reflection when GRPC_REFLECTION=true.
	if os.Getenv("GRPC_REFLECTION") == "true" {
		reflection.Register(gs)
	}

	// Register the LendingService server.
	RegisterLendingServiceServer(gs, handler)

	return &Server{
		gs:      gs,
		handler: handler,
		logger:  logger,
	}
}

// Serve starts the gRPC server on the specified address.
func (s *Server) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s.logger.Info("gRPC server listening", "addr", addr)
	return s.gs.Serve(lis)
}

// GracefulStop stops the server gracefully.
func (s *Server) GracefulStop() {
	s.logger.Info("gRPC server shutting down")
	s.gs.GracefulStop()
}
