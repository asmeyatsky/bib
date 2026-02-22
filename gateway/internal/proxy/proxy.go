// Package proxy provides HTTP-to-gRPC proxy clients for backend services.
//
// Each service client holds a gRPC connection and exposes HTTP handler
// functions that translate JSON requests into gRPC calls and return
// JSON responses. The clients use a JSON codec so that proto-generated
// stubs are not required; once generated code is available the raw
// Invoke calls can be replaced with typed client stubs.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// ServiceConn represents a gRPC client connection to a backend service.
type ServiceConn struct {
	Name   string
	Addr   string
	Conn   *grpc.ClientConn
	Health healthpb.HealthClient
	Logger *slog.Logger
}

// Dial establishes a gRPC connection to the backend service.
func Dial(name, addr string, logger *slog.Logger) (*ServiceConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s at %s: %w", name, addr, err)
	}

	logger.Info("connected to backend service", "service", name, "addr", addr)

	return &ServiceConn{
		Name:   name,
		Addr:   addr,
		Conn:   conn,
		Health: healthpb.NewHealthClient(conn),
		Logger: logger,
	}, nil
}

// Close closes the underlying gRPC connection.
func (sc *ServiceConn) Close() error {
	if sc == nil || sc.Conn == nil {
		return nil
	}
	return sc.Conn.Close()
}

// Invoke calls a gRPC method on the backend service using the JSON codec.
// Returns an appropriate error if the connection is not established.
func (sc *ServiceConn) Invoke(ctx context.Context, method string, req, resp interface{}) error {
	if sc == nil || sc.Conn == nil {
		return status.Error(codes.Unavailable, "backend service not connected")
	}
	return sc.Conn.Invoke(ctx, method, req, resp, grpcCallOption())
}

// CheckHealth queries the gRPC health check endpoint of the backend service.
func (sc *ServiceConn) CheckHealth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := sc.Health.Check(ctx, &healthpb.HealthCheckRequest{
		Service: sc.Name,
	})
	if err != nil {
		return fmt.Errorf("health check %s: %w", sc.Name, err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("service %s not serving: %s", sc.Name, resp.Status)
	}
	return nil
}

// readJSON reads and unmarshals a JSON request body into the provided value.
func readJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB max
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		return fmt.Errorf("request body is empty")
	}
	return json.Unmarshal(body, v)
}

// writeJSON marshals the value as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, statusCode int, msg string) {
	writeJSON(w, statusCode, map[string]string{"error": msg})
}

// grpcToHTTPStatus maps a gRPC status code to an HTTP status code.
func grpcToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// handleGRPCError writes an appropriate HTTP error response for a gRPC error.
func handleGRPCError(w http.ResponseWriter, err error, logger *slog.Logger) {
	st, ok := status.FromError(err)
	if !ok {
		logger.Error("backend call failed", "error", err)
		writeError(w, http.StatusBadGateway, "backend service unavailable")
		return
	}
	httpStatus := grpcToHTTPStatus(st.Code())
	logger.Error("backend gRPC error",
		"code", st.Code().String(),
		"message", st.Message(),
		"http_status", httpStatus,
	)
	writeError(w, httpStatus, st.Message())
}

// jsonCodec is a gRPC codec that uses JSON encoding.
// This allows making raw gRPC calls without proto-generated types.
type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return "json"
}
