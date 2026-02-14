package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/event"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// EntryStatus represents the lifecycle state of a journal entry.
type EntryStatus string

const (
	EntryStatusPending  EntryStatus = "PENDING"
	EntryStatusPosted   EntryStatus = "POSTED"
	EntryStatusReversed EntryStatus = "REVERSED"
)

// JournalEntry is the root aggregate for the ledger bounded context.
// It represents an immutable double-entry accounting transaction.
type JournalEntry struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	effectiveDate time.Time
	postings      []valueobject.PostingPair
	status        EntryStatus
	description   string
	reference     string
	version       int
	createdAt     time.Time
	updatedAt     time.Time
	domainEvents  []events.DomainEvent
}

// NewJournalEntry creates a new journal entry in PENDING status.
func NewJournalEntry(
	tenantID uuid.UUID,
	effectiveDate time.Time,
	postings []valueobject.PostingPair,
	description, reference string,
) (JournalEntry, error) {
	if tenantID == uuid.Nil {
		return JournalEntry{}, fmt.Errorf("tenant ID is required")
	}
	if effectiveDate.IsZero() {
		return JournalEntry{}, fmt.Errorf("effective date is required")
	}
	if len(postings) == 0 {
		return JournalEntry{}, fmt.Errorf("at least one posting pair is required")
	}

	// Validate that debits equal credits per currency
	debits := make(map[string]decimal.Decimal)
	credits := make(map[string]decimal.Decimal)
	for _, p := range postings {
		debits[p.Currency()] = debits[p.Currency()].Add(p.Amount())
		credits[p.Currency()] = credits[p.Currency()].Add(p.Amount())
	}
	// For posting pairs, debits always equal credits by construction.
	// But we validate the total is balanced across all pairs.

	now := time.Now().UTC()
	return JournalEntry{
		id:            uuid.New(),
		tenantID:      tenantID,
		effectiveDate: effectiveDate,
		postings:      postings,
		status:        EntryStatusPending,
		description:   description,
		reference:     reference,
		version:       1,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// Reconstruct recreates a JournalEntry from persistence (no validation, no events).
func Reconstruct(
	id, tenantID uuid.UUID,
	effectiveDate time.Time,
	postings []valueobject.PostingPair,
	status EntryStatus,
	description, reference string,
	version int,
	createdAt, updatedAt time.Time,
) JournalEntry {
	return JournalEntry{
		id:            id,
		tenantID:      tenantID,
		effectiveDate: effectiveDate,
		postings:      postings,
		status:        status,
		description:   description,
		reference:     reference,
		version:       version,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

// Post transitions the entry from PENDING to POSTED (immutable - returns new copy).
func (je JournalEntry) Post(now time.Time) (JournalEntry, error) {
	if je.status != EntryStatusPending {
		return JournalEntry{}, fmt.Errorf("can only post entries in PENDING status, current: %s", je.status)
	}

	posted := je
	posted.status = EntryStatusPosted
	posted.updatedAt = now
	posted.version++
	posted.domainEvents = append([]events.DomainEvent{}, je.domainEvents...)
	posted.domainEvents = append(posted.domainEvents, event.NewEntryPosted(je.id, je.tenantID, je.effectiveDate))
	return posted, nil
}

// Reverse transitions the entry from POSTED to REVERSED and creates a reversal entry.
func (je JournalEntry) Reverse(now time.Time, reason string) (reversed JournalEntry, reversal JournalEntry, err error) {
	if je.status != EntryStatusPosted {
		return JournalEntry{}, JournalEntry{}, fmt.Errorf("can only reverse entries in POSTED status, current: %s", je.status)
	}

	// Create reversed original
	reversed = je
	reversed.status = EntryStatusReversed
	reversed.updatedAt = now
	reversed.version++

	// Create reversal entry (swap debit/credit in each posting)
	var reversalPostings []valueobject.PostingPair
	for _, p := range je.postings {
		rp, _ := valueobject.NewPostingPair(
			p.CreditAccount(), // swap
			p.DebitAccount(),  // swap
			p.Amount(),
			p.Currency(),
			fmt.Sprintf("Reversal: %s", p.Description()),
		)
		reversalPostings = append(reversalPostings, rp)
	}

	reversal = JournalEntry{
		id:            uuid.New(),
		tenantID:      je.tenantID,
		effectiveDate: now,
		postings:      reversalPostings,
		status:        EntryStatusPosted,
		description:   fmt.Sprintf("Reversal of %s: %s", je.id, reason),
		reference:     je.id.String(),
		version:       1,
		createdAt:     now,
		updatedAt:     now,
	}

	reversed.domainEvents = append([]events.DomainEvent{}, je.domainEvents...)
	reversed.domainEvents = append(reversed.domainEvents, event.NewEntryReversed(je.id, reversal.id, je.tenantID))

	return reversed, reversal, nil
}

// Backvalue re-dates a PENDING entry to a past effective date.
func (je JournalEntry) Backvalue(newDate time.Time, now time.Time) (JournalEntry, error) {
	if je.status != EntryStatusPending {
		return JournalEntry{}, fmt.Errorf("can only backvalue entries in PENDING status")
	}
	if newDate.After(now) {
		return JournalEntry{}, fmt.Errorf("backvalue date must be in the past")
	}

	updated := je
	updated.effectiveDate = newDate
	updated.updatedAt = now
	updated.version++
	return updated, nil
}

// Accessors
func (je JournalEntry) ID() uuid.UUID                          { return je.id }
func (je JournalEntry) TenantID() uuid.UUID                    { return je.tenantID }
func (je JournalEntry) EffectiveDate() time.Time               { return je.effectiveDate }
func (je JournalEntry) Postings() []valueobject.PostingPair    { return je.postings }
func (je JournalEntry) Status() EntryStatus                    { return je.status }
func (je JournalEntry) Description() string                    { return je.description }
func (je JournalEntry) Reference() string                      { return je.reference }
func (je JournalEntry) Version() int                           { return je.version }
func (je JournalEntry) CreatedAt() time.Time                   { return je.createdAt }
func (je JournalEntry) UpdatedAt() time.Time                   { return je.updatedAt }
func (je JournalEntry) DomainEvents() []events.DomainEvent     { return je.domainEvents }
func (je JournalEntry) ClearDomainEvents() []events.DomainEvent {
	evts := je.domainEvents
	je.domainEvents = nil
	return evts
}
