package model

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// createActiveTestAccount is a helper that creates an account and activates it.
func createActiveTestAccount(t *testing.T) CustomerAccount {
	t.Helper()

	holder, err := NewAccountHolder(
		uuid.New(),
		"Jane",
		"Doe",
		"jane.doe@example.com",
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("failed to create holder: %v", err)
	}

	account, err := NewCustomerAccount(
		uuid.New(),
		valueobject.AccountTypeChecking,
		"USD",
		holder,
	)
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	active, err := account.Activate(time.Now())
	if err != nil {
		t.Fatalf("failed to activate account: %v", err)
	}

	return active
}

// TestCustomerAccount_ConcurrentStateTransitions spawns goroutines that each
// attempt to freeze, close, or activate copies of an active account concurrently.
// Because CustomerAccount is a value type (methods receive by value and return
// new copies), there are no shared mutable references. The test verifies that
// the race detector does not flag any data races and that only valid state
// transitions succeed.
func TestCustomerAccount_ConcurrentStateTransitions(t *testing.T) {
	account := createActiveTestAccount(t)
	now := time.Now()

	const goroutines = 10

	type result struct {
		account CustomerAccount
		err     error
		op      string
	}

	results := make([]result, goroutines*3)
	var wg sync.WaitGroup

	// Each goroutine operates on its own copy of the value-type account.
	for i := 0; i < goroutines; i++ {
		idx := i * 3

		wg.Add(3)

		go func(base int) {
			defer wg.Done()
			a, err := account.Freeze("suspicious activity", now)
			results[base] = result{account: a, err: err, op: "freeze"}
		}(idx)

		go func(base int) {
			defer wg.Done()
			a, err := account.Close("customer request", now)
			results[base+1] = result{account: a, err: err, op: "close"}
		}(idx)

		go func(base int) {
			defer wg.Done()
			// Activate should fail since account is already ACTIVE.
			a, err := account.Activate(now)
			results[base+2] = result{account: a, err: err, op: "activate"}
		}(idx)
	}

	wg.Wait()

	// Verify results.
	freezeSuccesses := 0
	closeSuccesses := 0
	activateSuccesses := 0

	for _, r := range results {
		switch r.op {
		case "freeze":
			if r.err == nil {
				freezeSuccesses++
				if r.account.Status() != AccountStatusFrozen {
					t.Errorf("freeze succeeded but status is %s, want FROZEN", r.account.Status())
				}
			}
		case "close":
			if r.err == nil {
				closeSuccesses++
				if r.account.Status() != AccountStatusClosed {
					t.Errorf("close succeeded but status is %s, want CLOSED", r.account.Status())
				}
			}
		case "activate":
			if r.err == nil {
				activateSuccesses++
			}
		}
	}

	// Since the source account is ACTIVE, all freeze and close calls should
	// succeed (they each work on an independent copy) and all activate calls
	// should fail.
	if freezeSuccesses != goroutines {
		t.Errorf("expected %d freeze successes, got %d", goroutines, freezeSuccesses)
	}
	if closeSuccesses != goroutines {
		t.Errorf("expected %d close successes, got %d", goroutines, closeSuccesses)
	}
	if activateSuccesses != 0 {
		t.Errorf("expected 0 activate successes, got %d", activateSuccesses)
	}

	// Verify the original account was not mutated.
	if account.Status() != AccountStatusActive {
		t.Errorf("original account status mutated: got %s, want ACTIVE", account.Status())
	}
}

// TestCustomerAccount_ImmutabilityUnderConcurrency verifies that when an
// account value is shared across many goroutines calling various methods, the
// original value is never mutated.
func TestCustomerAccount_ImmutabilityUnderConcurrency(t *testing.T) {
	account := createActiveTestAccount(t)
	now := time.Now()

	originalID := account.ID()
	originalStatus := account.Status()
	originalVersion := account.Version()
	originalCurrency := account.Currency()
	originalEventsLen := len(account.DomainEvents())

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			switch idx % 5 {
			case 0:
				account.Freeze("reason", now) //nolint:errcheck
			case 1:
				account.Close("reason", now) //nolint:errcheck
			case 2:
				account.AssignLedgerCode("1000-001", now) //nolint:errcheck
			case 3:
				account.DomainEvents()
			case 4:
				account.ClearDomainEvents()
			}
		}(i)
	}

	wg.Wait()

	// Verify the original account is completely untouched.
	if account.ID() != originalID {
		t.Error("account ID was mutated")
	}
	if account.Status() != originalStatus {
		t.Error("account status was mutated")
	}
	if account.Version() != originalVersion {
		t.Error("account version was mutated")
	}
	if account.Currency() != originalCurrency {
		t.Error("account currency was mutated")
	}
	if len(account.DomainEvents()) != originalEventsLen {
		t.Errorf("account domain events count changed: got %d, want %d",
			len(account.DomainEvents()), originalEventsLen)
	}
}
