package event

import (
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
}

func NewEntryPosted(entryID, tenantID uuid.UUID, effectiveDate time.Time) EntryPosted {
	return EntryPosted{
		BaseEvent:     events.NewBaseEvent("ledger.entry.posted", entryID.String(), AggregateTypeJournalEntry, tenantID.String()),
		EntryID:       entryID,
		EffectiveDate: effectiveDate,
	}
}

// EntryReversed is emitted when a journal entry is reversed.
type EntryReversed struct {
	events.BaseEvent
	EntryID         uuid.UUID `json:"entry_id"`
	ReversalEntryID uuid.UUID `json:"reversal_entry_id"`
}

func NewEntryReversed(entryID, reversalEntryID, tenantID uuid.UUID) EntryReversed {
	return EntryReversed{
		BaseEvent:       events.NewBaseEvent("ledger.entry.reversed", entryID.String(), AggregateTypeJournalEntry, tenantID.String()),
		EntryID:         entryID,
		ReversalEntryID: reversalEntryID,
	}
}

// PeriodClosed is emitted when a fiscal period is closed.
type PeriodClosed struct {
	events.BaseEvent
	Period string `json:"period"`
}

func NewPeriodClosed(tenantID uuid.UUID, period string) PeriodClosed {
	id := uuid.New()
	return PeriodClosed{
		BaseEvent: events.NewBaseEvent("ledger.period.closed", id.String(), "FiscalPeriod", tenantID.String()),
		Period:    period,
	}
}
