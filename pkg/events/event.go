package events

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the interface all domain events must implement.
type DomainEvent interface {
	EventID() string
	EventType() string
	AggregateID() string
	AggregateType() string
	TenantID() string
	OccurredAt() time.Time
}

// BaseEvent provides a default implementation of DomainEvent.
type BaseEvent struct {
	ID             string    `json:"event_id"`
	Type           string    `json:"event_type"`
	AggregateIDV   string    `json:"aggregate_id"`
	AggregateTypeV string    `json:"aggregate_type"`
	Tenant         string    `json:"tenant_id"`
	Timestamp      time.Time `json:"occurred_at"`
}

// NewBaseEvent creates a new BaseEvent with a generated UUID and the current time.
func NewBaseEvent(eventType string, aggregateID string, aggregateType string, tenantID string) BaseEvent {
	return BaseEvent{
		ID:             uuid.New().String(),
		Type:           eventType,
		AggregateIDV:   aggregateID,
		AggregateTypeV: aggregateType,
		Tenant:         tenantID,
		Timestamp:      time.Now().UTC(),
	}
}

// EventID returns the unique identifier for this event.
func (e BaseEvent) EventID() string {
	return e.ID
}

// EventType returns the type name of this event.
func (e BaseEvent) EventType() string {
	return e.Type
}

// AggregateID returns the identifier of the aggregate that produced this event.
func (e BaseEvent) AggregateID() string {
	return e.AggregateIDV
}

// AggregateType returns the type name of the aggregate that produced this event.
func (e BaseEvent) AggregateType() string {
	return e.AggregateTypeV
}

// TenantID returns the tenant identifier for this event.
func (e BaseEvent) TenantID() string {
	return e.Tenant
}

// OccurredAt returns the time at which this event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}
