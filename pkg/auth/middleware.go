package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// ContextWithClaims returns a new context with the given Claims attached.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext extracts Claims from the context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

// UnaryAuthInterceptor returns a gRPC unary server interceptor for JWT auth.
func UnaryAuthInterceptor(jwtService *JWTService, skipMethods []string) grpc.UnaryServerInterceptor {
	skipSet := make(map[string]struct{}, len(skipMethods))
	for _, m := range skipMethods {
		skipSet[m] = struct{}{}
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication for whitelisted methods.
		if _, skip := skipSet[info.FullMethod]; skip {
			return handler(ctx, req)
		}

		// Extract the authorization token from metadata.
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		tokenString := authHeader[0]
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}

		// Validate the token.
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Attach claims to the context.
		newCtx := context.WithValue(ctx, claimsContextKey, claims)
		return handler(newCtx, req)
	}
}

// RequireRole returns a gRPC unary server interceptor that checks for a required role.
func RequireRole(roles ...string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "no claims in context")
		}

		for _, required := range roles {
			if claims.HasRole(required) {
				return handler(ctx, req)
			}
		}

		return nil, status.Errorf(codes.PermissionDenied, "required role(s): %v", roles)
	}
}
