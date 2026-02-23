package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestJWTService() *JWTService {
	svc, err := NewJWTService(JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		Issuer:     "bib-test",
		Expiration: 15 * time.Minute,
	})
	if err != nil {
		panic("newTestJWTService: " + err.Error())
	}
	return svc
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
	svc, err := NewJWTService(JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		Issuer:     "bib-test",
		Expiration: -1 * time.Hour, // already expired
	})
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

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
	svc1, err := NewJWTService(JWTConfig{
		Secret:     "secret-one",
		Issuer:     "bib-test",
		Expiration: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}
	svc2, err := NewJWTService(JWTConfig{
		Secret:     "secret-two",
		Issuer:     "bib-test",
		Expiration: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

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

// --- RSA asymmetric signing tests ---

func TestRSA_GenerateAndValidateToken(t *testing.T) {
	privPEM, pubPEM, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Create issuer service with private key.
	issuer, err := NewJWTService(JWTConfig{
		PrivateKeyPEM: string(privPEM),
		Issuer:        "bib-test",
		Expiration:    15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTService(private key) error = %v", err)
	}

	userID := uuid.New()
	tenantID := uuid.New()
	roles := []string{RoleAdmin, RoleOperator}

	tokenString, err := issuer.GenerateToken(userID, tenantID, roles)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if tokenString == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	// Validate with public key only (simulates backend service).
	validator, err := NewJWTService(JWTConfig{
		PublicKeyPEM: string(pubPEM),
		Issuer:       "bib-test",
	})
	if err != nil {
		t.Fatalf("NewJWTService(public key) error = %v", err)
	}

	claims, err := validator.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("TenantID = %v, want %v", claims.TenantID, tenantID)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != RoleAdmin || claims.Roles[1] != RoleOperator {
		t.Errorf("Roles = %v, want [%s, %s]", claims.Roles, RoleAdmin, RoleOperator)
	}
	if claims.Issuer != "bib-test" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "bib-test")
	}
}

func TestRSA_ValidationOnlyMode_CannotGenerate(t *testing.T) {
	_, pubPEM, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	validator, err := NewJWTService(JWTConfig{
		PublicKeyPEM: string(pubPEM),
		Issuer:       "bib-test",
		Expiration:   15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTService(public key) error = %v", err)
	}

	_, err = validator.GenerateToken(uuid.New(), uuid.New(), []string{RoleCustomer})
	if err == nil {
		t.Fatal("GenerateToken() expected error in validation-only mode, got nil")
	}
}

func TestRSA_InvalidSignature_DifferentKeys(t *testing.T) {
	// Generate two different keypairs.
	privPEM1, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	_, pubPEM2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuer, err := NewJWTService(JWTConfig{
		PrivateKeyPEM: string(privPEM1),
		Issuer:        "bib-test",
		Expiration:    15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	tokenString, err := issuer.GenerateToken(uuid.New(), uuid.New(), []string{RoleCustomer})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Validate with different public key -- should fail.
	validator, err := NewJWTService(JWTConfig{
		PublicKeyPEM: string(pubPEM2),
		Issuer:       "bib-test",
	})
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	_, err = validator.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("ValidateToken() expected error for mismatched RSA keys, got nil")
	}
}

func TestRSA_IssuerCanAlsoValidate(t *testing.T) {
	privPEM, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	svc, err := NewJWTService(JWTConfig{
		PrivateKeyPEM: string(privPEM),
		Issuer:        "bib-test",
		Expiration:    15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}

	userID := uuid.New()
	tokenString, err := svc.GenerateToken(userID, uuid.New(), []string{RoleAdmin})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := svc.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
}

func TestGenerateKeyPair(t *testing.T) {
	privPEM, pubPEM, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	if len(privPEM) == 0 {
		t.Fatal("private key PEM is empty")
	}
	if len(pubPEM) == 0 {
		t.Fatal("public key PEM is empty")
	}
}

func TestNewJWTService_NoConfig(t *testing.T) {
	_, err := NewJWTService(JWTConfig{
		Issuer: "bib-test",
	})
	if err == nil {
		t.Fatal("NewJWTService() expected error with no key configuration, got nil")
	}
}
