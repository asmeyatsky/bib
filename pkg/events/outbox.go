package events

import (
	"context"
	"encoding/json"
	"time"
)

// OutboxEntry represents a domain event stored in the outbox table.
type OutboxEntry struct {
	ID            string
	AggregateID   string
	AggregateType string
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

// NewOutboxEntry creates an OutboxEntry from a DomainEvent.
// The payload is produced by JSON-marshalling the event itself.
func NewOutboxEntry(event DomainEvent) OutboxEntry {
	payload, _ := json.Marshal(event)
	return OutboxEntry{
		ID:            event.EventID(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Payload:       payload,
		CreatedAt:     event.OccurredAt(),
		PublishedAt:   nil,
	}
}

// OutboxRepository is the port for outbox persistence.
type OutboxRepository interface {
	Store(ctx context.Context, entries []OutboxEntry) error
	FetchUnpublished(ctx context.Context, batchSize int) ([]OutboxEntry, error)
	MarkPublished(ctx context.Context, ids []string) error
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...DomainEvent) error
}
