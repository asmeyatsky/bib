package grpc

import (
	"fmt"
	"log/slog"
	"net"

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
func NewServer(handler *LendingHandler, logger *slog.Logger) *Server {
	gs := grpc.NewServer()

	// Register gRPC health check.
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(gs, healthSrv)
	healthSrv.SetServingStatus("lending-service", healthpb.HealthCheckResponse_SERVING)

	// Enable reflection for development tooling (grpcurl, etc.).
	reflection.Register(gs)

	// TODO: Register the generated LendingService server once proto is compiled.
	// pb.RegisterLendingServiceServer(gs, handler)

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
