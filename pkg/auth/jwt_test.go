package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestJWTService() *JWTService {
	return NewJWTService(JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		Issuer:     "bib-test",
		Expiration: 15 * time.Minute,
	})
}

func TestGenerateAndValidateToken(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()
	tenantID := uuid.New()
	roles := []string{RoleAdmin, RoleOperator}

	tokenString, err := svc.GenerateToken(userID, tenantID, roles)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if tokenString == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	claims, err := svc.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", claims.TenantID, tenantID)
	}
	if len(claims.Roles) != 2 {
		t.Fatalf("Roles length = %d, want 2", len(claims.Roles))
	}
	if claims.Roles[0] != RoleAdmin || claims.Roles[1] != RoleOperator {
		t.Errorf("Roles = %v, want [%s, %s]", claims.Roles, RoleAdmin, RoleOperator)
	}
	if claims.Issuer != "bib-test" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "bib-test")
	}
	if claims.Subject != userID.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, userID.String())
	}
}

func TestValidateToken_Expired(t *testing.T) {
	svc := NewJWTService(JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		Issuer:     "bib-test",
		Expiration: -1 * time.Hour, // already expired
	})

	tokenString, err := svc.GenerateToken(uuid.New(), uuid.New(), []string{RoleCustomer})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = svc.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("ValidateToken() expected error for expired token, got nil")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	svc1 := NewJWTService(JWTConfig{
		Secret:     "secret-one",
		Issuer:     "bib-test",
		Expiration: 15 * time.Minute,
	})
	svc2 := NewJWTService(JWTConfig{
		Secret:     "secret-two",
		Issuer:     "bib-test",
		Expiration: 15 * time.Minute,
	})

	tokenString, err := svc1.GenerateToken(uuid.New(), uuid.New(), []string{RoleCustomer})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = svc2.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("ValidateToken() expected error for invalid signature, got nil")
	}
}

func TestHasRole(t *testing.T) {
	claims := Claims{
		Roles: []string{RoleAdmin, RoleAuditor},
	}

	if !claims.HasRole(RoleAdmin) {
		t.Error("HasRole(RoleAdmin) = false, want true")
	}
	if !claims.HasRole(RoleAuditor) {
		t.Error("HasRole(RoleAuditor) = false, want true")
	}
	if claims.HasRole(RoleCustomer) {
		t.Error("HasRole(RoleCustomer) = true, want false")
	}
	if claims.HasRole("nonexistent") {
		t.Error("HasRole(nonexistent) = true, want false")
	}
}

func TestClaimsFromContext(t *testing.T) {
	// Test with no claims in context.
	ctx := context.Background()
	_, ok := ClaimsFromContext(ctx)
	if ok {
		t.Error("ClaimsFromContext() ok = true for empty context, want false")
	}

	// Test with claims in context.
	expected := &Claims{
		UserID: uuid.New(),
		Roles:  []string{RoleOperator},
	}
	ctx = context.WithValue(ctx, claimsContextKey, expected)
	got, ok := ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("ClaimsFromContext() ok = false, want true")
	}
	if got.UserID != expected.UserID {
		t.Errorf("ClaimsFromContext().UserID = %v, want %v", got.UserID, expected.UserID)
	}
	if len(got.Roles) != 1 || got.Roles[0] != RoleOperator {
		t.Errorf("ClaimsFromContext().Roles = %v, want [%s]", got.Roles, RoleOperator)
	}
}
