package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/card-service/internal/domain/event"
	"github.com/bibbank/bib/services/card-service/internal/domain/model"
)

// CardRepository defines the persistence port for card aggregates.
type CardRepository interface {
	// Save persists a new card aggregate.
	Save(ctx context.Context, card model.Card) error

	// Update persists changes to an existing card aggregate.
	// Must enforce optimistic concurrency via the version field.
	Update(ctx context.Context, card model.Card) error

	// FindByID retrieves a card by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.Card, error)

	// FindByAccountID retrieves all cards belonging to an account.
	FindByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.Card, error)

	// FindByTenantID retrieves all cards belonging to a tenant.
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.Card, error)

	// SaveTransaction records a card transaction.
	SaveTransaction(ctx context.Context, cardID uuid.UUID, amount decimal.Decimal, currency, merchantName, merchantCategory, authCode, status string) error
}

// EventPublisher defines the port for publishing domain events.
type EventPublisher interface {
	// Publish sends domain events to the event bus.
	Publish(ctx context.Context, events []event.DomainEvent) error
}

// CardProcessorAdapter defines the port for interacting with external
// card processors such as Marqeta or Adyen.
type CardProcessorAdapter interface {
	// IssuePhysicalCard requests the external processor to issue a physical card.
	IssuePhysicalCard(ctx context.Context, card model.Card) error

	// GetCardDetails retrieves card details from the external processor.
	GetCardDetails(ctx context.Context, cardID uuid.UUID) error
}

// AccountBalanceClient defines the port for querying account balances.
// This is used by JIT funding to verify available funds before authorization.
type AccountBalanceClient interface {
	// GetAvailableBalance returns the available balance for the given account.
	GetAvailableBalance(ctx context.Context, accountID uuid.UUID) (decimal.Decimal, error)
}
