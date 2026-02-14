package valueobject

import (
	"fmt"
	"regexp"
)

// RoutingInfo holds the external routing details for a payment.
type RoutingInfo struct {
	routingNumber         string
	externalAccountNumber string
}

var routingNumberPattern = regexp.MustCompile(`^\d{9}$`)

// NewRoutingInfo validates and creates a RoutingInfo value object.
// The routing number must be exactly 9 digits (ABA routing number format for ACH).
// The external account number must not be empty.
func NewRoutingInfo(routingNumber, accountNumber string) (RoutingInfo, error) {
	if routingNumber == "" && accountNumber == "" {
		// Empty routing info is valid for internal transfers.
		return RoutingInfo{}, nil
	}
	if routingNumber != "" && !routingNumberPattern.MatchString(routingNumber) {
		return RoutingInfo{}, fmt.Errorf("routing number must be exactly 9 digits, got: %q", routingNumber)
	}
	if routingNumber != "" && accountNumber == "" {
		return RoutingInfo{}, fmt.Errorf("external account number is required when routing number is provided")
	}
	return RoutingInfo{
		routingNumber:         routingNumber,
		externalAccountNumber: accountNumber,
	}, nil
}

// RoutingNumber returns the ABA routing number.
func (r RoutingInfo) RoutingNumber() string {
	return r.routingNumber
}

// ExternalAccountNumber returns the external account number.
func (r RoutingInfo) ExternalAccountNumber() string {
	return r.externalAccountNumber
}

// IsEmpty returns true if the routing info has no routing details.
func (r RoutingInfo) IsEmpty() bool {
	return r.routingNumber == "" && r.externalAccountNumber == ""
}
