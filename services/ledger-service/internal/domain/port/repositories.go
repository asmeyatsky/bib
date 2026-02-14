package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// JournalRepository defines persistence operations for journal entries.
type JournalRepository interface {
	// Save persists a journal entry (insert or update).
	Save(ctx context.Context, entry model.JournalEntry) error
	// FindByID retrieves a journal entry by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.JournalEntry, error)
	// ListByAccount returns journal entries filtered by account code within a date range.
	ListByAccount(ctx context.Context, tenantID uuid.UUID, account valueobject.AccountCode, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error)
	// ListByTenant returns journal entries for a tenant within a date range.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time, limit, offset int) ([]model.JournalEntry, int, error)
}

// BalanceRepository defines persistence operations for account balances.
type BalanceRepository interface {
	// UpdateBalance atomically adjusts the balance for an account/currency by delta.
	UpdateBalance(ctx context.Context, account valueobject.AccountCode, currency string, delta decimal.Decimal) error
	// GetBalance retrieves the balance for an account/currency as of a given time.
	GetBalance(ctx context.Context, account valueobject.AccountCode, currency string, asOf time.Time) (decimal.Decimal, error)
}

// FiscalPeriodRepository defines persistence operations for fiscal periods.
type FiscalPeriodRepository interface {
	// GetPeriodStatus returns the current status of a fiscal period.
	GetPeriodStatus(ctx context.Context, tenantID uuid.UUID, period valueobject.FiscalPeriod) (valueobject.PeriodStatus, error)
	// ClosePeriod marks a fiscal period as closed.
	ClosePeriod(ctx context.Context, tenantID uuid.UUID, period valueobject.FiscalPeriod) error
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...events.DomainEvent) error
}
