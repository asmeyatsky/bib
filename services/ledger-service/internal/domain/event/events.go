package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
)

const AggregateTypeJournalEntry = "JournalEntry"

// EntryPosted is emitted when a journal entry is posted.
type EntryPosted struct {
	events.BaseEvent
	EntryID       uuid.UUID `json:"entry_id"`
	EffectiveDate time.Time `json:"effective_date"`
	TenantID      uuid.UUID `json:"tenant_id"`
}

func NewEntryPosted(entryID, tenantID uuid.UUID, effectiveDate time.Time) EntryPosted {
	payload, _ := json.Marshal(struct {
		EntryID       uuid.UUID `json:"entry_id"`
		EffectiveDate time.Time `json:"effective_date"`
		TenantID      uuid.UUID `json:"tenant_id"`
	}{entryID, effectiveDate, tenantID})

	return EntryPosted{
		BaseEvent:     events.NewBaseEvent("ledger.entry.posted", entryID, AggregateTypeJournalEntry, payload),
		EntryID:       entryID,
		EffectiveDate: effectiveDate,
		TenantID:      tenantID,
	}
}

// EntryReversed is emitted when a journal entry is reversed.
type EntryReversed struct {
	events.BaseEvent
	EntryID         uuid.UUID `json:"entry_id"`
	ReversalEntryID uuid.UUID `json:"reversal_entry_id"`
	TenantID        uuid.UUID `json:"tenant_id"`
}

func NewEntryReversed(entryID, reversalEntryID, tenantID uuid.UUID) EntryReversed {
	payload, _ := json.Marshal(struct {
		EntryID         uuid.UUID `json:"entry_id"`
		ReversalEntryID uuid.UUID `json:"reversal_entry_id"`
		TenantID        uuid.UUID `json:"tenant_id"`
	}{entryID, reversalEntryID, tenantID})

	return EntryReversed{
		BaseEvent:       events.NewBaseEvent("ledger.entry.reversed", entryID, AggregateTypeJournalEntry, payload),
		EntryID:         entryID,
		ReversalEntryID: reversalEntryID,
		TenantID:        tenantID,
	}
}

// PeriodClosed is emitted when a fiscal period is closed.
type PeriodClosed struct {
	events.BaseEvent
	TenantID uuid.UUID `json:"tenant_id"`
	Period   string    `json:"period"`
}

func NewPeriodClosed(tenantID uuid.UUID, period string) PeriodClosed {
	id := uuid.New()
	payload, _ := json.Marshal(struct {
		TenantID uuid.UUID `json:"tenant_id"`
		Period   string    `json:"period"`
	}{tenantID, period})

	return PeriodClosed{
		BaseEvent: events.NewBaseEvent("ledger.period.closed", id, "FiscalPeriod", payload),
		TenantID:  tenantID,
		Period:    period,
	}
}
