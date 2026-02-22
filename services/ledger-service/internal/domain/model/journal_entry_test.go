package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func newTestPostings(t *testing.T) []valueobject.PostingPair {
	t.Helper()
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	pp, err := valueobject.NewPostingPair(debit, credit, decimal.NewFromInt(100), "USD", "test posting")
	require.NoError(t, err)
	return []valueobject.PostingPair{pp}
}

func TestNewJournalEntry_Valid(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test entry", "REF-001")
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, entry.ID())
	assert.Equal(t, tenantID, entry.TenantID())
	assert.Equal(t, effectiveDate, entry.EffectiveDate())
	assert.Len(t, entry.Postings(), 1)
	assert.Equal(t, model.EntryStatusPending, entry.Status())
	assert.Equal(t, "Test entry", entry.Description())
	assert.Equal(t, "REF-001", entry.Reference())
	assert.Equal(t, 1, entry.Version())
	assert.False(t, entry.CreatedAt().IsZero())
	assert.False(t, entry.UpdatedAt().IsZero())
	assert.Empty(t, entry.DomainEvents())
}

func TestNewJournalEntry_MissingTenantID(t *testing.T) {
	postings := newTestPostings(t)
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)

	_, err := model.NewJournalEntry(uuid.Nil, effectiveDate, postings, "Test", "REF")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestNewJournalEntry_MissingEffectiveDate(t *testing.T) {
	tenantID := uuid.New()
	postings := newTestPostings(t)

	_, err := model.NewJournalEntry(tenantID, time.Time{}, postings, "Test", "REF")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "effective date is required")
}

func TestNewJournalEntry_EmptyPostings(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)

	_, err := model.NewJournalEntry(tenantID, effectiveDate, nil, "Test", "REF")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one posting pair is required")

	_, err = model.NewJournalEntry(tenantID, effectiveDate, []valueobject.PostingPair{}, "Test", "REF")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one posting pair is required")
}

func TestNewJournalEntry_MultiplePostings(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)

	debit1 := valueobject.MustAccountCode("1000")
	credit1 := valueobject.MustAccountCode("2000")
	pp1, err := valueobject.NewPostingPair(debit1, credit1, decimal.NewFromInt(100), "USD", "posting 1")
	require.NoError(t, err)

	debit2 := valueobject.MustAccountCode("3000")
	credit2 := valueobject.MustAccountCode("4000")
	pp2, err := valueobject.NewPostingPair(debit2, credit2, decimal.NewFromInt(200), "EUR", "posting 2")
	require.NoError(t, err)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, []valueobject.PostingPair{pp1, pp2}, "Multi-posting", "REF-002")
	require.NoError(t, err)
	assert.Len(t, entry.Postings(), 2)
}

func TestJournalEntry_Post_FromPending(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)
	assert.Equal(t, model.EntryStatusPending, entry.Status())

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	assert.Equal(t, model.EntryStatusPosted, posted.Status())
	assert.Equal(t, 2, posted.Version())
	assert.Equal(t, now, posted.UpdatedAt())
	assert.Equal(t, entry.ID(), posted.ID()) // same ID

	// Original should remain unchanged (immutable behavior)
	assert.Equal(t, model.EntryStatusPending, entry.Status())
	assert.Equal(t, 1, entry.Version())
}

func TestJournalEntry_Post_GeneratesDomainEvent(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	events := posted.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "ledger.entry.posted", events[0].EventType())
	assert.Equal(t, entry.ID().String(), events[0].AggregateID())
}

func TestJournalEntry_Post_FromPosted_Error(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	// Attempting to post again should fail
	_, err = posted.Post(now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only post entries in PENDING status")
}

func TestJournalEntry_Post_FromReversed_Error(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	reversed, _, err := posted.Reverse(now, "reason")
	require.NoError(t, err)

	// Attempting to post a reversed entry should fail
	_, err = reversed.Post(now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only post entries in PENDING status")
}

func TestJournalEntry_Reverse_FromPosted(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	reversalTime := now.Add(time.Hour)
	reversed, reversal, err := posted.Reverse(reversalTime, "error correction")
	require.NoError(t, err)

	// Reversed original
	assert.Equal(t, model.EntryStatusReversed, reversed.Status())
	assert.Equal(t, posted.ID(), reversed.ID())
	assert.Equal(t, 3, reversed.Version()) // was 2 (posted), now 3

	// Reversal entry
	assert.NotEqual(t, uuid.Nil, reversal.ID())
	assert.NotEqual(t, posted.ID(), reversal.ID())
	assert.Equal(t, model.EntryStatusPosted, reversal.Status())
	assert.Equal(t, tenantID, reversal.TenantID())
	assert.Equal(t, 1, reversal.Version())
	assert.Equal(t, posted.ID().String(), reversal.Reference())
	assert.Contains(t, reversal.Description(), "Reversal of")
	assert.Contains(t, reversal.Description(), "error correction")

	// Reversal postings should have swapped accounts
	require.Len(t, reversal.Postings(), 1)
	originalPosting := posted.Postings()[0]
	reversalPosting := reversal.Postings()[0]

	assert.True(t, reversalPosting.DebitAccount().Equal(originalPosting.CreditAccount()))
	assert.True(t, reversalPosting.CreditAccount().Equal(originalPosting.DebitAccount()))
	assert.True(t, reversalPosting.Amount().Equal(originalPosting.Amount()))
	assert.Equal(t, originalPosting.Currency(), reversalPosting.Currency())
}

func TestJournalEntry_Reverse_GeneratesDomainEvent(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	reversed, _, err := posted.Reverse(now, "reason")
	require.NoError(t, err)

	events := reversed.DomainEvents()
	// Should have the original post event plus the reversal event
	require.Len(t, events, 2)
	assert.Equal(t, "ledger.entry.posted", events[0].EventType())
	assert.Equal(t, "ledger.entry.reversed", events[1].EventType())
}

func TestJournalEntry_Reverse_FromPending_Error(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	_, _, err = entry.Reverse(now, "reason")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only reverse entries in POSTED status")
}

func TestJournalEntry_Reverse_FromReversed_Error(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	reversed, _, err := posted.Reverse(now, "first reversal")
	require.NoError(t, err)

	_, _, err = reversed.Reverse(now, "second reversal")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only reverse entries in POSTED status")
}

func TestJournalEntry_Backvalue_FromPending(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	pastDate := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	backvalued, err := entry.Backvalue(pastDate, now)
	require.NoError(t, err)

	assert.Equal(t, pastDate, backvalued.EffectiveDate())
	assert.Equal(t, 2, backvalued.Version())
	assert.Equal(t, now, backvalued.UpdatedAt())
	assert.Equal(t, model.EntryStatusPending, backvalued.Status())

	// Original remains unchanged
	assert.Equal(t, effectiveDate, entry.EffectiveDate())
	assert.Equal(t, 1, entry.Version())
}

func TestJournalEntry_Backvalue_FromPosted_Error(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)

	pastDate := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	_, err = posted.Backvalue(pastDate, now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only backvalue entries in PENDING status")
}

func TestJournalEntry_Backvalue_FutureDate_Error(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	futureDate := time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC)
	_, err = entry.Backvalue(futureDate, now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backvalue date must be in the past")
}

func TestJournalEntry_Reconstruct(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)
	createdAt := time.Date(2024, time.March, 14, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.March, 14, 11, 0, 0, 0, time.UTC)

	entry := model.Reconstruct(
		id, tenantID, effectiveDate, postings,
		model.EntryStatusPosted, "Reconstructed", "REF-R",
		3, createdAt, updatedAt,
	)

	assert.Equal(t, id, entry.ID())
	assert.Equal(t, tenantID, entry.TenantID())
	assert.Equal(t, effectiveDate, entry.EffectiveDate())
	assert.Len(t, entry.Postings(), 1)
	assert.Equal(t, model.EntryStatusPosted, entry.Status())
	assert.Equal(t, "Reconstructed", entry.Description())
	assert.Equal(t, "REF-R", entry.Reference())
	assert.Equal(t, 3, entry.Version())
	assert.Equal(t, createdAt, entry.CreatedAt())
	assert.Equal(t, updatedAt, entry.UpdatedAt())
	assert.Empty(t, entry.DomainEvents())
}

func TestJournalEntry_ClearDomainEvents(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	now := time.Now().UTC()
	posted, err := entry.Post(now)
	require.NoError(t, err)
	require.Len(t, posted.DomainEvents(), 1)

	cleared, updated := posted.ClearDomainEvents()
	assert.Len(t, cleared, 1)
	assert.Equal(t, "ledger.entry.posted", cleared[0].EventType())
	assert.Empty(t, updated.DomainEvents())
}

func TestJournalEntry_Immutability_PostDoesNotMutateOriginal(t *testing.T) {
	tenantID := uuid.New()
	effectiveDate := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC)
	postings := newTestPostings(t)

	entry, err := model.NewJournalEntry(tenantID, effectiveDate, postings, "Test", "REF")
	require.NoError(t, err)

	originalVersion := entry.Version()
	originalStatus := entry.Status()

	now := time.Now().UTC()
	_, err = entry.Post(now)
	require.NoError(t, err)

	// Original must not have changed
	assert.Equal(t, originalVersion, entry.Version())
	assert.Equal(t, originalStatus, entry.Status())
}

func TestEntryStatus_Constants(t *testing.T) {
	assert.Equal(t, model.EntryStatus("PENDING"), model.EntryStatusPending)
	assert.Equal(t, model.EntryStatus("POSTED"), model.EntryStatusPosted)
	assert.Equal(t, model.EntryStatus("REVERSED"), model.EntryStatusReversed)
}
