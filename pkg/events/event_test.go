package events

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewBaseEvent(t *testing.T) {
	aggregateID := uuid.New()
	payload := []byte(`{"amount":100}`)

	before := time.Now().UTC()
	event := NewBaseEvent("AccountOpened", aggregateID, "Account", payload)
	after := time.Now().UTC()

	if event.EventID() == uuid.Nil {
		t.Error("expected non-nil event ID")
	}

	if event.EventType() != "AccountOpened" {
		t.Errorf("expected event type %q, got %q", "AccountOpened", event.EventType())
	}

	if event.AggregateID() != aggregateID {
		t.Errorf("expected aggregate ID %v, got %v", aggregateID, event.AggregateID())
	}

	if event.AggregateType() != "Account" {
		t.Errorf("expected aggregate type %q, got %q", "Account", event.AggregateType())
	}

	if event.OccurredAt().Before(before) || event.OccurredAt().After(after) {
		t.Errorf("expected occurredAt between %v and %v, got %v", before, after, event.OccurredAt())
	}

	if string(event.Payload()) != string(payload) {
		t.Errorf("expected payload %q, got %q", string(payload), string(event.Payload()))
	}
}

func TestBaseEventImplementsDomainEvent(t *testing.T) {
	var _ DomainEvent = BaseEvent{}
}

func TestNewOutboxEntry(t *testing.T) {
	aggregateID := uuid.New()
	payload := []byte(`{"currency":"USD"}`)
	event := NewBaseEvent("FundsDeposited", aggregateID, "Account", payload)

	entry := NewOutboxEntry(event)

	if entry.ID != event.EventID() {
		t.Errorf("expected outbox ID %v, got %v", event.EventID(), entry.ID)
	}

	if entry.AggregateID != aggregateID {
		t.Errorf("expected aggregate ID %v, got %v", aggregateID, entry.AggregateID)
	}

	if entry.AggregateType != "Account" {
		t.Errorf("expected aggregate type %q, got %q", "Account", entry.AggregateType)
	}

	if entry.EventType != "FundsDeposited" {
		t.Errorf("expected event type %q, got %q", "FundsDeposited", entry.EventType)
	}

	if string(entry.Payload) != string(payload) {
		t.Errorf("expected payload %q, got %q", string(payload), string(entry.Payload))
	}

	if entry.CreatedAt != event.OccurredAt() {
		t.Errorf("expected created at %v, got %v", event.OccurredAt(), entry.CreatedAt)
	}

	if entry.PublishedAt != nil {
		t.Error("expected published at to be nil")
	}
}

func TestEventCollectorRecord(t *testing.T) {
	collector := &EventCollector{}
	aggregateID := uuid.New()

	e1 := NewBaseEvent("Event1", aggregateID, "Aggregate", nil)
	e2 := NewBaseEvent("Event2", aggregateID, "Aggregate", nil)

	collector.Record(e1)
	collector.Record(e2)

	events := collector.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].EventType() != "Event1" {
		t.Errorf("expected first event type %q, got %q", "Event1", events[0].EventType())
	}

	if events[1].EventType() != "Event2" {
		t.Errorf("expected second event type %q, got %q", "Event2", events[1].EventType())
	}
}

func TestEventCollectorEventsDoesNotClear(t *testing.T) {
	collector := &EventCollector{}
	collector.Record(NewBaseEvent("Event1", uuid.New(), "Aggregate", nil))

	_ = collector.Events()

	if len(collector.Events()) != 1 {
		t.Error("expected Events() to not clear the internal slice")
	}
}

func TestEventCollectorClearEvents(t *testing.T) {
	collector := &EventCollector{}
	aggregateID := uuid.New()

	collector.Record(NewBaseEvent("Event1", aggregateID, "Aggregate", nil))
	collector.Record(NewBaseEvent("Event2", aggregateID, "Aggregate", nil))

	cleared := collector.ClearEvents()

	if len(cleared) != 2 {
		t.Fatalf("expected ClearEvents to return 2 events, got %d", len(cleared))
	}

	if len(collector.Events()) != 0 {
		t.Errorf("expected internal slice to be empty after ClearEvents, got %d events", len(collector.Events()))
	}
}

func TestEventCollectorClearEventsOnEmpty(t *testing.T) {
	collector := &EventCollector{}

	cleared := collector.ClearEvents()

	if cleared != nil {
		t.Errorf("expected nil from ClearEvents on empty collector, got %v", cleared)
	}
}
