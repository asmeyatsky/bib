package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OutboxEntry represents a domain event stored in the outbox table.
type OutboxEntry struct {
	ID            uuid.UUID
	AggregateID   uuid.UUID
	AggregateType string
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

// NewOutboxEntry creates an OutboxEntry from a DomainEvent.
func NewOutboxEntry(event DomainEvent) OutboxEntry {
	return OutboxEntry{
		ID:            event.EventID(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Payload:       event.Payload(),
		CreatedAt:     event.OccurredAt(),
		PublishedAt:   nil,
	}
}

// OutboxRepository is the port for outbox persistence.
type OutboxRepository interface {
	Store(ctx context.Context, entries []OutboxEntry) error
	FetchUnpublished(ctx context.Context, batchSize int) ([]OutboxEntry, error)
	MarkPublished(ctx context.Context, ids []uuid.UUID) error
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...DomainEvent) error
}
