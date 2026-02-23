package grpc

import (
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

// Server wraps the gRPC server for card-service.
type Server struct {
	grpcServer *grpc.Server
	handler    *CardServiceHandler
	logger     *slog.Logger
}

// NewServer creates a new gRPC server with the given handler.
func NewServer(handler *CardServiceHandler, logger *slog.Logger, jwtService *auth.JWTService) *Server {
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

	// Register gRPC health check.
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("card-service", healthpb.HealthCheckResponse_SERVING)

	// Register the CardService handler.
	RegisterCardServiceServer(grpcServer, handler)

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
