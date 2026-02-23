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

// Server wraps a gRPC server with health checks and the FX handler.
type Server struct {
	grpcServer *grpc.Server
	handler    *Handler
	logger     *slog.Logger
	port       int
}

// NewServer creates a new gRPC Server with health checking and reflection enabled.
func NewServer(handler *Handler, logger *slog.Logger, port int, jwtService *auth.JWTService) *Server {
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

	// Register health service.
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("fx-service", healthpb.HealthCheckResponse_SERVING)

	// Register the FXService handler.
	RegisterFXServiceServer(grpcServer, handler)

	// Only enable reflection when GRPC_REFLECTION=true.
	if os.Getenv("GRPC_REFLECTION") == "true" {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpcServer: grpcServer,
		handler:    handler,
		logger:     logger,
		port:       port,
	}
}

// Start begins listening for gRPC connections.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc listen on %s: %w", addr, err)
	}

	s.logger.Info("gRPC server starting", "addr", addr)
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}

// Handler returns the registered FX handler.
func (s *Server) Handler() *Handler {
	return s.handler
}
