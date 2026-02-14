package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims for BIB platform.
type Claims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Roles    []string  `json:"roles"`
}

// HasRole checks if the claims include the specified role.
func (c Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Role constants
const (
	RoleAdmin     = "admin"
	RoleOperator  = "operator"
	RoleAuditor   = "auditor"
	RoleCustomer  = "customer"
	RoleAPIClient = "api_client"
)
