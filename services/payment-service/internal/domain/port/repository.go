package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// PaymentOrderRepository defines persistence operations for payment orders.
type PaymentOrderRepository interface {
	// Save persists a payment order (insert or update).
	Save(ctx context.Context, order model.PaymentOrder) error
	// FindByID retrieves a payment order by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.PaymentOrder, error)
	// ListByAccount returns payment orders for a given account with pagination.
	ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error)
	// ListByTenant returns payment orders for a given tenant with pagination.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error)
}

// RailAdapter is the port for payment rail adapters (ACH, SWIFT, etc.).
type RailAdapter interface {
	// Submit sends a payment order to the external payment rail for processing.
	Submit(ctx context.Context, order model.PaymentOrder) error
	// GetStatus queries the external payment rail for the current status of a payment.
	GetStatus(ctx context.Context, orderID uuid.UUID) (valueobject.PaymentStatus, string, error)
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...events.DomainEvent) error
}

// FraudClient is the port for fraud assessment services.
type FraudClient interface {
	// AssessTransaction evaluates a transaction for fraud risk.
	// Returns true if the transaction is approved, false if it is flagged/rejected.
	AssessTransaction(ctx context.Context, tenantID, accountID uuid.UUID, amount decimal.Decimal, currency string) (bool, error)
}
