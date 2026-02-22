package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/bibbank/bib/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps a gRPC server for the identity service.
type Server struct {
	server  *grpc.Server
	handler *IdentityHandler
	port    int
	logger  *slog.Logger
}

func NewServer(handler *IdentityHandler, port int, logger *slog.Logger, jwtService *auth.JWTService, opts ...grpc.ServerOption) *Server {
	// Add auth interceptor, skipping health check methods.
	authInterceptor := auth.UnaryAuthInterceptor(jwtService, []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	})
	opts = append(opts, grpc.UnaryInterceptor(authInterceptor))

	srv := grpc.NewServer(opts...)

	// Register health check
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("identity-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Only enable reflection when GRPC_REFLECTION=true.
	if os.Getenv("GRPC_REFLECTION") == "true" {
		reflection.Register(srv)
	}

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
