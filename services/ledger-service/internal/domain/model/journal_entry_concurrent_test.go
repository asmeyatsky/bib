package model

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// createTestJournalEntry is a helper that builds a valid PENDING journal entry.
func createTestJournalEntry(t *testing.T) JournalEntry {
	t.Helper()

	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")

	pp, err := valueobject.NewPostingPair(
		debit, credit,
		decimal.NewFromInt(100),
		"USD",
		"test posting",
	)
	if err != nil {
		t.Fatalf("failed to create posting pair: %v", err)
	}

	entry, err := NewJournalEntry(
		uuid.New(),
		time.Now().UTC(),
		[]valueobject.PostingPair{pp},
		"test entry",
		"REF-001",
	)
	if err != nil {
		t.Fatalf("failed to create journal entry: %v", err)
	}

	return entry
}

// TestJournalEntry_ConcurrentEventCollection creates a PENDING journal entry
// and spawns goroutines that each call Post() on their own copy concurrently.
// It verifies that domain events do not leak between copies.
func TestJournalEntry_ConcurrentEventCollection(t *testing.T) {
	entry := createTestJournalEntry(t)
	now := time.Now().UTC()

	const goroutines = 100

	type result struct {
		err    error
		posted JournalEntry
	}

	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			posted, err := entry.Post(now)
			results[idx] = result{posted: posted, err: err}
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Errorf("goroutine %d: Post() returned unexpected error: %v", i, r.err)
			continue
		}

		if r.posted.Status() != EntryStatusPosted {
			t.Errorf("goroutine %d: expected POSTED status, got %s", i, r.posted.Status())
		}

		// Each posted copy should have exactly 1 domain event (EntryPosted).
		events := r.posted.DomainEvents()
		if len(events) != 1 {
			t.Errorf("goroutine %d: expected 1 domain event, got %d", i, len(events))
		}
	}

	// Original entry must remain PENDING with no domain events.
	if entry.Status() != EntryStatusPending {
		t.Errorf("original entry status mutated: got %s, want PENDING", entry.Status())
	}
	if len(entry.DomainEvents()) != 0 {
		t.Errorf("original entry has %d domain events, want 0", len(entry.DomainEvents()))
	}
}

// TestPostingPair_ConcurrentCreation creates posting pairs concurrently and
// verifies all are valid and independent of each other.
func TestPostingPair_ConcurrentCreation(t *testing.T) {
	const goroutines = 100

	type result struct {
		err  error
		pair valueobject.PostingPair
	}

	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			debit := valueobject.MustAccountCode("1000")
			credit := valueobject.MustAccountCode("2000")
			amount := decimal.NewFromInt(int64(idx + 1))

			pp, err := valueobject.NewPostingPair(
				debit, credit,
				amount,
				"USD",
				"concurrent posting",
			)
			results[idx] = result{pair: pp, err: err}
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Errorf("goroutine %d: NewPostingPair failed: %v", i, r.err)
			continue
		}

		expectedAmount := decimal.NewFromInt(int64(i + 1))
		if !r.pair.Amount().Equal(expectedAmount) {
			t.Errorf("goroutine %d: expected amount %s, got %s", i, expectedAmount, r.pair.Amount())
		}
		if r.pair.Currency() != "USD" {
			t.Errorf("goroutine %d: expected currency USD, got %s", i, r.pair.Currency())
		}
		if r.pair.DebitAccount().Code() != "1000" {
			t.Errorf("goroutine %d: expected debit account 1000, got %s", i, r.pair.DebitAccount().Code())
		}
		if r.pair.CreditAccount().Code() != "2000" {
			t.Errorf("goroutine %d: expected credit account 2000, got %s", i, r.pair.CreditAccount().Code())
		}
	}
}
