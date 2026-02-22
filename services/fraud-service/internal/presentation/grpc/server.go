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

// Server wraps the gRPC server with fraud service handlers.
type Server struct {
	grpcServer *grpc.Server
	handler    *FraudServiceHandler
	logger     *slog.Logger
	address    string
}

// NewServer creates a new gRPC server for the fraud service.
func NewServer(handler *FraudServiceHandler, address string, logger *slog.Logger, jwtService *auth.JWTService) *Server {
	// Add auth interceptor, skipping health check methods.
	authInterceptor := auth.UnaryAuthInterceptor(jwtService, []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	})
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))

	// Register health check service.
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("fraud-service", healthpb.HealthCheckResponse_SERVING)

	// Register the FraudService handler.
	RegisterFraudServiceServer(grpcServer, handler)

	// Only enable reflection when GRPC_REFLECTION=true.
	if os.Getenv("GRPC_REFLECTION") == "true" {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpcServer: grpcServer,
		handler:    handler,
		logger:     logger,
		address:    address,
	}
}

// Start begins listening and serving gRPC requests.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	s.logger.Info("gRPC server starting",
		slog.String("address", s.address),
	)

	return s.grpcServer.Serve(listener)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("gRPC server shutting down")
	s.grpcServer.GracefulStop()
}
