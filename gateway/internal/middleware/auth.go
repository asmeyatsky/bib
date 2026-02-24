package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/bibbank/bib/pkg/auth"
)

type bearerTokenKey struct{}

// BearerTokenFromContext retrieves the raw Bearer token stored by AuthMiddleware.
func BearerTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(bearerTokenKey{}).(string)
	return token, ok
}

// AuthMiddleware validates JWT tokens on incoming requests.
// Requests to paths listed in skipPaths bypass authentication.
func AuthMiddleware(jwtService *auth.JWTService, skipPaths []string) func(http.Handler) http.Handler {
	skipSet := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for certain paths.
			if _, skip := skipSet[r.URL.Path]; skip {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			rawToken := parts[1]
			claims, err := jwtService.ValidateToken(rawToken)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Add claims and raw token to context for downstream use.
			ctx := auth.ContextWithClaims(r.Context(), claims)
			ctx = context.WithValue(ctx, bearerTokenKey{}, rawToken)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
