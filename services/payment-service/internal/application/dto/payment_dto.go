package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InitiatePaymentRequest is the input DTO for initiating a payment order.
type InitiatePaymentRequest struct {
	TenantID              uuid.UUID
	SourceAccountID       uuid.UUID
	DestinationAccountID  uuid.UUID // uuid.Nil for external payments
	Amount                decimal.Decimal
	Currency              string
	RoutingNumber         string
	ExternalAccountNumber string
	DestinationCountry    string
	Reference             string
	Description           string
}

// InitiatePaymentResponse is the output DTO after a payment order is initiated.
type InitiatePaymentResponse struct {
	ID        uuid.UUID
	Status    string
	Rail      string
	CreatedAt time.Time
}

// GetPaymentRequest is the input DTO for retrieving a single payment order.
type GetPaymentRequest struct {
	PaymentID uuid.UUID
}

// PaymentOrderResponse is the output DTO for a payment order.
type PaymentOrderResponse struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	SourceAccountID       uuid.UUID
	DestinationAccountID  uuid.UUID
	Amount                decimal.Decimal
	Currency              string
	Rail                  string
	Status                string
	RoutingNumber         string
	ExternalAccountNumber string
	Reference             string
	Description           string
	FailureReason         string
	InitiatedAt           time.Time
	SettledAt             *time.Time
	Version               int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ListPaymentsRequest is the input DTO for listing payment orders.
type ListPaymentsRequest struct {
	TenantID  uuid.UUID
	AccountID uuid.UUID // optional; if set, filter by account
	PageSize  int
	Offset    int
}

// ListPaymentsResponse is the output DTO for listing payment orders.
type ListPaymentsResponse struct {
	Payments   []PaymentOrderResponse
	TotalCount int
}
