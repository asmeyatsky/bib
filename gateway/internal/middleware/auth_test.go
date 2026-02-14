package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/google/uuid"
)

func newTestJWTService() *auth.JWTService {
	return auth.NewJWTService(auth.JWTConfig{
		Secret:     "test-secret-key",
		Issuer:     "test",
		Expiration: 1 * time.Hour,
	})
}

func TestAuthMiddleware_SkipPaths(t *testing.T) {
	jwtSvc := newTestJWTService()
	mw := AuthMiddleware(jwtSvc, []string{"/healthz", "/readyz"})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request to skip path should succeed without auth.
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for skipped path, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	jwtSvc := newTestJWTService()
	mw := AuthMiddleware(jwtSvc, nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing header, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	jwtSvc := newTestJWTService()
	mw := AuthMiddleware(jwtSvc, nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid format, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtSvc := newTestJWTService()
	mw := AuthMiddleware(jwtSvc, nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtSvc := newTestJWTService()
	mw := AuthMiddleware(jwtSvc, nil)

	var gotClaims bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if ok && claims != nil {
			gotClaims = true
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Generate a valid token.
	token, err := jwtSvc.GenerateToken(uuid.New(), uuid.New(), []string{"admin"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d", rec.Code)
	}
	if !gotClaims {
		t.Fatal("expected claims to be present in context")
	}
}
