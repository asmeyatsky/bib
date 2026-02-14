package events

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the interface all domain events must implement.
type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	AggregateID() uuid.UUID
	AggregateType() string
	OccurredAt() time.Time
	Payload() []byte
}

// BaseEvent provides a default implementation of DomainEvent.
type BaseEvent struct {
	id            uuid.UUID
	eventType     string
	aggregateID   uuid.UUID
	aggregateType string
	occurredAt    time.Time
	payload       []byte
}

// NewBaseEvent creates a new BaseEvent with a generated UUID and the current time.
func NewBaseEvent(eventType string, aggregateID uuid.UUID, aggregateType string, payload []byte) BaseEvent {
	return BaseEvent{
		id:            uuid.New(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		occurredAt:    time.Now().UTC(),
		payload:       payload,
	}
}

// EventID returns the unique identifier for this event.
func (e BaseEvent) EventID() uuid.UUID {
	return e.id
}

// EventType returns the type name of this event.
func (e BaseEvent) EventType() string {
	return e.eventType
}

// AggregateID returns the identifier of the aggregate that produced this event.
func (e BaseEvent) AggregateID() uuid.UUID {
	return e.aggregateID
}

// AggregateType returns the type name of the aggregate that produced this event.
func (e BaseEvent) AggregateType() string {
	return e.aggregateType
}

// OccurredAt returns the time at which this event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// Payload returns the serialized event payload.
func (e BaseEvent) Payload() []byte {
	return e.payload
}
