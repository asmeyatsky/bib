package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/account-service/internal/domain/event"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// AccountRepository defines the persistence port for CustomerAccount aggregates.
type AccountRepository interface {
	// Save persists a CustomerAccount. If the account already exists, it updates it
	// using optimistic concurrency control via the version field.
	Save(ctx context.Context, account model.CustomerAccount) error

	// FindByID retrieves a CustomerAccount by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error)

	// FindByAccountNumber retrieves a CustomerAccount by its account number.
	FindByAccountNumber(ctx context.Context, number valueobject.AccountNumber) (model.CustomerAccount, error)

	// ListByTenant retrieves all accounts for a given tenant with pagination.
	// Returns the accounts, total count, and any error.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error)

	// ListByHolder retrieves all accounts for a given holder with pagination.
	// Returns the accounts, total count, and any error.
	ListByHolder(ctx context.Context, holderID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error)
}

// EventPublisher defines the port for publishing domain events.
type EventPublisher interface {
	// Publish sends domain events to the specified topic.
	Publish(ctx context.Context, topic string, events ...event.DomainEvent) error
}

// LedgerClient is a port for communicating with the ledger service.
type LedgerClient interface {
	// CreateLedgerAccount requests the creation of a ledger account in the ledger service.
	CreateLedgerAccount(ctx context.Context, tenantID uuid.UUID, accountCode string, currency string) error
}
