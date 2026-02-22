package grpc

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/bibbank/bib/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server with account service handlers.
type Server struct {
	grpcServer   *grpc.Server
	healthServer *health.Server
	handler      *AccountHandler
	port         int
	logger       *slog.Logger
}

// NewServer creates a new gRPC server with the provided handler.
func NewServer(handler *AccountHandler, port int, logger *slog.Logger, jwtService *auth.JWTService) *Server {
	// Add auth interceptor, skipping health check methods.
	authInterceptor := auth.UnaryAuthInterceptor(jwtService, []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	})
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
	healthServer := health.NewServer()

	// Register health check service.
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Register the AccountService handler.
	RegisterAccountServiceServer(grpcServer, handler)

	// Only enable reflection when GRPC_REFLECTION=true.
	if os.Getenv("GRPC_REFLECTION") == "true" {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpcServer:   grpcServer,
		healthServer: healthServer,
		handler:      handler,
		port:         port,
		logger:       logger,
	}
}

// Start begins listening for gRPC connections.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	s.logger.Info("gRPC server starting", "port", s.port)

	// Mark the service as healthy.
	s.healthServer.SetServingStatus("account-service", healthpb.HealthCheckResponse_SERVING)

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.healthServer.SetServingStatus("account-service", healthpb.HealthCheckResponse_NOT_SERVING)
	s.grpcServer.GracefulStop()
}

// GRPCServer returns the underlying grpc.Server for additional registration.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}
