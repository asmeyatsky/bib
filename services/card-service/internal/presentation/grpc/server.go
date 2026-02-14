package grpc

import (
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server for card-service.
type Server struct {
	grpcServer *grpc.Server
	handler    *CardServiceHandler
	logger     *slog.Logger
}

// NewServer creates a new gRPC server with the given handler.
func NewServer(handler *CardServiceHandler, logger *slog.Logger) *Server {
	grpcServer := grpc.NewServer()

	// Register gRPC health check.
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("card-service", healthpb.HealthCheckResponse_SERVING)

	// Enable reflection for development tooling.
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		handler:    handler,
		logger:     logger,
	}
}

// Start begins listening on the specified address.
func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.logger.Info("gRPC server starting", slog.String("addr", addr))
	return s.grpcServer.Serve(listener)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}
