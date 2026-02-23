package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InitiatePaymentRequest is the input DTO for initiating a payment order.
type InitiatePaymentRequest struct {
	Amount                decimal.Decimal
	Currency              string
	RoutingNumber         string
	ExternalAccountNumber string
	DestinationCountry    string
	Reference             string
	Description           string
	TenantID              uuid.UUID
	SourceAccountID       uuid.UUID
	DestinationAccountID  uuid.UUID
}

// InitiatePaymentResponse is the output DTO after a payment order is initiated.
type InitiatePaymentResponse struct {
	CreatedAt time.Time
	Status    string
	Rail      string
	ID        uuid.UUID
}

// GetPaymentRequest is the input DTO for retrieving a single payment order.
type GetPaymentRequest struct {
	PaymentID uuid.UUID
}

// PaymentOrderResponse is the output DTO for a payment order.
type PaymentOrderResponse struct {
	InitiatedAt           time.Time
	UpdatedAt             time.Time
	CreatedAt             time.Time
	SettledAt             *time.Time
	RoutingNumber         string
	Rail                  string
	Status                string
	Currency              string
	ExternalAccountNumber string
	Reference             string
	Description           string
	FailureReason         string
	Amount                decimal.Decimal
	Version               int
	ID                    uuid.UUID
	DestinationAccountID  uuid.UUID
	SourceAccountID       uuid.UUID
	TenantID              uuid.UUID
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
