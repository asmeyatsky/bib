package grpc

import (
	"fmt"
	"log/slog"
	"net"

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
func NewServer(handler *ReportingHandler, logger *slog.Logger) *Server {
	grpcServer := grpc.NewServer()

	// Register health check.
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("reporting-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection for development.
	reflection.Register(grpcServer)

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
