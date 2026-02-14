package middleware

import (
	"net/http"
	"strings"

	"github.com/bibbank/bib/pkg/auth"
)

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

			claims, err := jwtService.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Add claims to context using the auth package's context helper.
			ctx := auth.ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
