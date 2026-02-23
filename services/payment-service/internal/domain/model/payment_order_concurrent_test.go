package model

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// createTestPaymentOrder is a helper that builds a valid INITIATED payment order.
func createTestPaymentOrder(t *testing.T) PaymentOrder {
	t.Helper()

	routing, err := valueobject.NewRoutingInfo("", "")
	if err != nil {
		t.Fatalf("failed to create routing info: %v", err)
	}

	order, err := NewPaymentOrder(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		decimal.NewFromInt(500),
		"USD",
		valueobject.RailInternal,
		routing,
		"REF-PAY-001",
		"test payment",
	)
	if err != nil {
		t.Fatalf("failed to create payment order: %v", err)
	}

	return order
}

// TestPaymentOrder_ConcurrentStatusTransitions spawns goroutines that attempt
// various state transitions on the same payment order concurrently.
// Since the model is immutable (value receiver, returns new copies), each
// goroutine works on its own copy. The test verifies no data races occur and
// that the state machine is respected.
func TestPaymentOrder_ConcurrentStatusTransitions(t *testing.T) {
	order := createTestPaymentOrder(t)
	now := time.Now().UTC()

	// First, move to PROCESSING so we can test multiple transitions from there.
	processing, err := order.MarkProcessing(now)
	if err != nil {
		t.Fatalf("failed to mark processing: %v", err)
	}

	const goroutines = 10

	type result struct {
		order PaymentOrder
		err   error
		op    string
	}

	results := make([]result, goroutines*4)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		idx := i * 4

		wg.Add(4)

		// Try to settle from PROCESSING -- should succeed.
		go func(base int) {
			defer wg.Done()
			o, err := processing.Settle(now)
			results[base] = result{order: o, err: err, op: "settle"}
		}(idx)

		// Try to fail from PROCESSING -- should succeed.
		go func(base int) {
			defer wg.Done()
			o, err := processing.Fail("timeout", now)
			results[base+1] = result{order: o, err: err, op: "fail"}
		}(idx)

		// Try to reverse from PROCESSING -- should fail (needs SETTLED).
		go func(base int) {
			defer wg.Done()
			o, err := processing.Reverse("fraud", now)
			results[base+2] = result{order: o, err: err, op: "reverse"}
		}(idx)

		// Try to mark processing again from PROCESSING -- should fail.
		go func(base int) {
			defer wg.Done()
			o, err := processing.MarkProcessing(now)
			results[base+3] = result{order: o, err: err, op: "mark_processing"}
		}(idx)
	}

	wg.Wait()

	settleSuccesses := 0
	failSuccesses := 0
	reverseSuccesses := 0
	markProcessingSuccesses := 0

	for _, r := range results {
		switch r.op {
		case "settle":
			if r.err == nil {
				settleSuccesses++
				if r.order.Status() != valueobject.PaymentStatusSettled {
					t.Errorf("settle succeeded but status is %s, want SETTLED", r.order.Status())
				}
			}
		case "fail":
			if r.err == nil {
				failSuccesses++
				if r.order.Status() != valueobject.PaymentStatusFailed {
					t.Errorf("fail succeeded but status is %s, want FAILED", r.order.Status())
				}
			}
		case "reverse":
			if r.err == nil {
				reverseSuccesses++
			}
		case "mark_processing":
			if r.err == nil {
				markProcessingSuccesses++
			}
		}
	}

	// From PROCESSING: settle and fail should succeed, reverse and re-process should fail.
	if settleSuccesses != goroutines {
		t.Errorf("expected %d settle successes, got %d", goroutines, settleSuccesses)
	}
	if failSuccesses != goroutines {
		t.Errorf("expected %d fail successes, got %d", goroutines, failSuccesses)
	}
	if reverseSuccesses != 0 {
		t.Errorf("expected 0 reverse successes from PROCESSING, got %d", reverseSuccesses)
	}
	if markProcessingSuccesses != 0 {
		t.Errorf("expected 0 mark_processing successes from PROCESSING, got %d", markProcessingSuccesses)
	}

	// Original processing order must be unchanged.
	if processing.Status() != valueobject.PaymentStatusProcessing {
		t.Errorf("original processing order status mutated: got %s, want PROCESSING", processing.Status())
	}
}

// TestPaymentOrder_ConcurrentClearEvents verifies that calling ClearDomainEvents
// concurrently is safe and produces correct results.
func TestPaymentOrder_ConcurrentClearEvents(t *testing.T) {
	order := createTestPaymentOrder(t)

	originalEventsLen := len(order.DomainEvents())
	if originalEventsLen == 0 {
		t.Fatal("expected order to have domain events after creation")
	}

	const goroutines = 100

	type result struct {
		cleared PaymentOrder
		events  int
	}

	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			evts, cleared := order.ClearDomainEvents()
			results[idx] = result{cleared: cleared, events: len(evts)}
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		// Each goroutine should get the same events from the original.
		if r.events != originalEventsLen {
			t.Errorf("goroutine %d: expected %d events, got %d", i, originalEventsLen, r.events)
		}
		// The cleared copy should have no domain events.
		if len(r.cleared.DomainEvents()) != 0 {
			t.Errorf("goroutine %d: cleared order still has %d events", i, len(r.cleared.DomainEvents()))
		}
	}

	// Original order must still have its events.
	if len(order.DomainEvents()) != originalEventsLen {
		t.Errorf("original order events changed: got %d, want %d",
			len(order.DomainEvents()), originalEventsLen)
	}
}
