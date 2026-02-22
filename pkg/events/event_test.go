package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewBaseEvent(t *testing.T) {
	aggregateID := "agg-123"
	tenantID := "tenant-456"

	before := time.Now().UTC()
	event := NewBaseEvent("AccountOpened", aggregateID, "Account", tenantID)
	after := time.Now().UTC()

	if event.EventID() == "" {
		t.Error("expected non-empty event ID")
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

	if event.TenantID() != tenantID {
		t.Errorf("expected tenant ID %q, got %q", tenantID, event.TenantID())
	}

	if event.OccurredAt().Before(before) || event.OccurredAt().After(after) {
		t.Errorf("expected occurredAt between %v and %v, got %v", before, after, event.OccurredAt())
	}
}

func TestBaseEventImplementsDomainEvent(t *testing.T) {
	var _ DomainEvent = BaseEvent{}
}

func TestNewOutboxEntry(t *testing.T) {
	aggregateID := "agg-789"
	tenantID := "tenant-012"
	event := NewBaseEvent("FundsDeposited", aggregateID, "Account", tenantID)

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

	// Payload should be a valid JSON marshalling of the event.
	if len(entry.Payload) == 0 {
		t.Error("expected non-empty payload")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(entry.Payload, &parsed); err != nil {
		t.Errorf("expected valid JSON payload, got error: %v", err)
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
	aggregateID := "agg-test"

	e1 := NewBaseEvent("Event1", aggregateID, "Aggregate", "")
	e2 := NewBaseEvent("Event2", aggregateID, "Aggregate", "")

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
	collector.Record(NewBaseEvent("Event1", "agg", "Aggregate", ""))

	_ = collector.Events()

	if len(collector.Events()) != 1 {
		t.Error("expected Events() to not clear the internal slice")
	}
}

func TestEventCollectorClearEvents(t *testing.T) {
	collector := &EventCollector{}
	aggregateID := "agg-clear"

	collector.Record(NewBaseEvent("Event1", aggregateID, "Aggregate", ""))
	collector.Record(NewBaseEvent("Event2", aggregateID, "Aggregate", ""))

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
