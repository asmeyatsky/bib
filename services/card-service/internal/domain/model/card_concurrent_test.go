package model

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

// createActiveTestCard is a helper that creates a card and activates it.
func createActiveTestCard(t *testing.T) Card {
	t.Helper()

	card, err := NewCard(
		uuid.New(),
		uuid.New(),
		valueobject.CardTypeVirtual,
		"USD",
		decimal.NewFromInt(1000),  // daily limit
		decimal.NewFromInt(10000), // monthly limit
	)
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	active, err := card.Activate(time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to activate card: %v", err)
	}

	return active
}

// TestCard_ConcurrentAuthorizeTransaction spawns goroutines that each
// authorize a transaction on their own copy of an active card. Since the Card
// model uses value receivers, each goroutine works independently. The test
// verifies that the race detector finds no data races and that spending limits
// are enforced per-copy.
func TestCard_ConcurrentAuthorizeTransaction(t *testing.T) {
	card := createActiveTestCard(t)
	now := time.Now().UTC()

	const goroutines = 100
	txnAmount := decimal.NewFromInt(50) // Each $50; daily limit is $1000

	type result struct {
		card     Card
		authCode string
		err      error
	}

	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			c, code, err := card.AuthorizeTransaction(
				txnAmount,
				"Test Merchant",
				"RETAIL",
				now,
			)
			results[idx] = result{card: c, authCode: code, err: err}
		}(i)
	}

	wg.Wait()

	successCount := 0
	for i, r := range results {
		if r.err != nil {
			t.Errorf("goroutine %d: unexpected authorization error: %v", i, r.err)
			continue
		}

		successCount++

		// Each copy should have the amount added independently.
		expectedDaily := card.DailySpent().Add(txnAmount)
		if !r.card.DailySpent().Equal(expectedDaily) {
			t.Errorf("goroutine %d: daily spent = %s, want %s",
				i, r.card.DailySpent(), expectedDaily)
		}

		if r.authCode == "" {
			t.Errorf("goroutine %d: empty auth code", i)
		}
	}

	// All goroutines should succeed since each gets its own copy with zero spent.
	if successCount != goroutines {
		t.Errorf("expected %d successes, got %d", goroutines, successCount)
	}

	// Verify original card was not mutated.
	if !card.DailySpent().IsZero() {
		t.Errorf("original card daily spent was mutated: %s", card.DailySpent())
	}
	if !card.MonthlySpent().IsZero() {
		t.Errorf("original card monthly spent was mutated: %s", card.MonthlySpent())
	}

	// Test spending limit enforcement: create a card that has already spent
	// close to its daily limit, then try to authorize transactions that would
	// exceed it.
	cardNearLimit := createActiveTestCard(t)
	// Spend up to $950 on the card (limit is $1000).
	spentCard, _, err := cardNearLimit.AuthorizeTransaction(
		decimal.NewFromInt(950), "Big Store", "RETAIL", now,
	)
	if err != nil {
		t.Fatalf("failed to spend on card: %v", err)
	}

	// Now try concurrent authorizations of $100 each -- all should fail.
	failResults := make([]result, goroutines)
	var wg2 sync.WaitGroup
	wg2.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg2.Done()
			c, code, err := spentCard.AuthorizeTransaction(
				decimal.NewFromInt(100),
				"Another Store",
				"RETAIL",
				now,
			)
			failResults[idx] = result{card: c, authCode: code, err: err}
		}(i)
	}

	wg2.Wait()

	for i, r := range failResults {
		if r.err == nil {
			t.Errorf("goroutine %d: expected authorization to fail (limit exceeded), but it succeeded", i)
		}
	}
}

// TestCard_ConcurrentFreezeAndAuthorize tests the scenario where one goroutine
// freezes the card while others try to authorize. Since Card is a value type,
// the freeze and authorize operate on independent copies. The test verifies
// that a frozen card correctly rejects transactions.
func TestCard_ConcurrentFreezeAndAuthorize(t *testing.T) {
	card := createActiveTestCard(t)
	now := time.Now().UTC()

	// First, create the frozen version.
	frozenCard, err := card.Freeze(now)
	if err != nil {
		t.Fatalf("failed to freeze card: %v", err)
	}

	const goroutines = 100

	type authResult struct {
		err error
	}

	// Launch goroutines that try to authorize on the frozen card.
	authResults := make([]authResult, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, _, err := frozenCard.AuthorizeTransaction(
				decimal.NewFromInt(10),
				"Test Merchant",
				"RETAIL",
				now,
			)
			authResults[idx] = authResult{err: err}
		}(i)
	}

	wg.Wait()

	// All authorization attempts on a frozen card must fail.
	for i, r := range authResults {
		if r.err == nil {
			t.Errorf("goroutine %d: authorization on frozen card should have failed", i)
		}
	}

	// Now test concurrent freeze and authorize on the active card.
	// These operate on independent copies, so freeze always succeeds
	// and authorize always succeeds on the active copy.
	type mixedResult struct {
		freezeErr error
		authErr   error
	}

	mixedResults := make([]mixedResult, goroutines)
	var wg2 sync.WaitGroup
	wg2.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg2.Done()
			_, err := card.Freeze(now)
			mixedResults[idx].freezeErr = err
		}(i)

		go func(idx int) {
			defer wg2.Done()
			_, _, err := card.AuthorizeTransaction(
				decimal.NewFromInt(10),
				"Test Merchant",
				"RETAIL",
				now,
			)
			mixedResults[idx].authErr = err
		}(i)
	}

	wg2.Wait()

	for i, r := range mixedResults {
		if r.freezeErr != nil {
			t.Errorf("goroutine %d: freeze on active card failed: %v", i, r.freezeErr)
		}
		if r.authErr != nil {
			t.Errorf("goroutine %d: authorize on active card failed: %v", i, r.authErr)
		}
	}

	// Verify original card unchanged.
	if card.Status() != valueobject.CardStatusActive {
		t.Errorf("original card status mutated: got %s, want ACTIVE", card.Status())
	}
}
