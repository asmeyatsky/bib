package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps a gRPC server for the deposit service.
type Server struct {
	server  *grpc.Server
	handler *DepositHandler
	port    int
	logger  *slog.Logger
}

func NewServer(handler *DepositHandler, port int, logger *slog.Logger, opts ...grpc.ServerOption) *Server {
	srv := grpc.NewServer(opts...)

	// Register health check
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("deposit-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for development
	reflection.Register(srv)

	return &Server{
		server:  srv,
		handler: handler,
		port:    port,
		logger:  logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	s.logger.Info("gRPC server starting", "port", s.port)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutting down gRPC server")
		s.server.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) Stop() {
	s.server.GracefulStop()
}
