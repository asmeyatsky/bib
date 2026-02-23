package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	// Secret is the HMAC-SHA256 symmetric key.
	// Deprecated: Use PrivateKeyPEM/PublicKeyPEM for RSA asymmetric signing.
	Secret string

	// PrivateKeyPEM is a PEM-encoded RSA private key for signing tokens (issuer mode).
	PrivateKeyPEM string

	// PublicKeyPEM is a PEM-encoded RSA public key for validating tokens (validator mode).
	PublicKeyPEM string

	// SigningMethod selects the algorithm: "RS256" (default) or "HS256" (legacy).
	// When PrivateKeyPEM or PublicKeyPEM is set, this defaults to "RS256".
	// When only Secret is set, this defaults to "HS256".
	SigningMethod string

	Issuer     string
	Expiration time.Duration
}

// JWTService handles JWT token operations.
type JWTService struct {
	config     JWTConfig
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	useRSA     bool
}

// NewJWTService creates a new JWTService with the given configuration.
//
// Configuration modes:
//   - PrivateKeyPEM set: full issuer mode (can sign and validate). The public key is derived.
//   - PublicKeyPEM set (no private): validation-only mode. GenerateToken returns an error.
//   - Only Secret set: legacy HMAC-SHA256 mode (backwards compatible).
func NewJWTService(cfg JWTConfig) (*JWTService, error) {
	svc := &JWTService{config: cfg}

	switch {
	case cfg.PrivateKeyPEM != "":
		// Parse RSA private key and derive public key.
		privKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(cfg.PrivateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
		}
		svc.privateKey = privKey
		svc.publicKey = &privKey.PublicKey
		svc.useRSA = true

	case cfg.PublicKeyPEM != "":
		// Validation-only mode: parse RSA public key.
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.PublicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
		}
		svc.publicKey = pubKey
		svc.useRSA = true

	case cfg.Secret != "":
		// Legacy HMAC-SHA256 mode.
		svc.useRSA = false

	default:
		return nil, fmt.Errorf("jwt configuration requires PrivateKeyPEM, PublicKeyPEM, or Secret")
	}

	return svc, nil
}

// GenerateToken creates a new JWT token for the given user.
func (s *JWTService) GenerateToken(userID, tenantID uuid.UUID, roles []string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.Expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:   userID,
		TenantID: tenantID,
		Roles:    roles,
	}

	if s.useRSA {
		if s.privateKey == nil {
			return "", fmt.Errorf("cannot generate token: no private key configured (validation-only mode)")
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signedToken, err := token.SignedString(s.privateKey)
		if err != nil {
			return "", fmt.Errorf("failed to sign token with RSA: %w", err)
		}
		return signedToken, nil
	}

	// Legacy HMAC-SHA256 mode.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken parses and validates a JWT token string.
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if s.useRSA {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v (expected RS256)", token.Header["alg"])
			}
			return s.publicKey, nil
		}

		// Legacy HMAC-SHA256 mode.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Validate issuer if configured.
	if s.config.Issuer != "" {
		if claims.Issuer != s.config.Issuer {
			return nil, fmt.Errorf("invalid issuer: got %q, want %q", claims.Issuer, s.config.Issuer)
		}
	}

	return claims, nil
}

// LoadKeyFromFile reads a PEM-encoded key from a file path.
func LoadKeyFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %q: %w", path, err)
	}
	return data, nil
}

// GenerateKeyPair generates a 2048-bit RSA keypair and returns PEM-encoded bytes.
// Useful for development and testing.
func GenerateKeyPair() (privateKeyPEM, publicKeyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM.
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	// Encode public key to PEM.
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	return privPEM, pubPEM, nil
}
