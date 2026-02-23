package money

import (
	"sync"
	"testing"

	"github.com/shopspring/decimal"
)

// TestMoney_ConcurrentArithmetic performs Add, Subtract, and Multiply
// operations on shared Money values across goroutines. Because Money is an
// immutable value type, operations return new values without modifying the
// original. The test verifies that the original Money value is never changed.
func TestMoney_ConcurrentArithmetic(t *testing.T) {
	base := New(decimal.NewFromInt(1000), USD)
	addend := New(decimal.NewFromInt(50), USD)
	subtrahend := New(decimal.NewFromInt(25), USD)
	factor := decimal.NewFromFloat(1.5)

	originalAmount := base.Amount()

	const goroutines = 100

	type result struct {
		addResult      Money
		addErr         error
		subtractResult Money
		subtractErr    error
		multiplyResult Money
		negateResult   Money
		absResult      Money
	}

	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			r := &results[idx]

			r.addResult, r.addErr = base.Add(addend)
			r.subtractResult, r.subtractErr = base.Subtract(subtrahend)
			r.multiplyResult = base.Multiply(factor)
			r.negateResult = base.Negate()
			r.absResult = base.Abs()
		}(i)
	}

	wg.Wait()

	// Verify original is unchanged.
	if !base.Amount().Equal(originalAmount) {
		t.Errorf("original base amount mutated: got %s, want %s", base.Amount(), originalAmount)
	}
	if base.Currency() != USD {
		t.Errorf("original base currency mutated: got %s, want USD", base.Currency())
	}

	expectedAdd := decimal.NewFromInt(1050)
	expectedSub := decimal.NewFromInt(975)
	expectedMul := decimal.NewFromFloat(1500)
	expectedNeg := decimal.NewFromInt(-1000)
	expectedAbs := decimal.NewFromInt(1000)

	for i, r := range results {
		if r.addErr != nil {
			t.Errorf("goroutine %d: Add returned error: %v", i, r.addErr)
		} else if !r.addResult.Amount().Equal(expectedAdd) {
			t.Errorf("goroutine %d: Add = %s, want %s", i, r.addResult.Amount(), expectedAdd)
		}

		if r.subtractErr != nil {
			t.Errorf("goroutine %d: Subtract returned error: %v", i, r.subtractErr)
		} else if !r.subtractResult.Amount().Equal(expectedSub) {
			t.Errorf("goroutine %d: Subtract = %s, want %s", i, r.subtractResult.Amount(), expectedSub)
		}

		if !r.multiplyResult.Amount().Equal(expectedMul) {
			t.Errorf("goroutine %d: Multiply = %s, want %s", i, r.multiplyResult.Amount(), expectedMul)
		}

		if !r.negateResult.Amount().Equal(expectedNeg) {
			t.Errorf("goroutine %d: Negate = %s, want %s", i, r.negateResult.Amount(), expectedNeg)
		}

		if !r.absResult.Amount().Equal(expectedAbs) {
			t.Errorf("goroutine %d: Abs = %s, want %s", i, r.absResult.Amount(), expectedAbs)
		}

		// Verify currency is preserved on all results.
		for _, m := range []Money{r.addResult, r.subtractResult, r.multiplyResult, r.negateResult, r.absResult} {
			if m.Currency() != USD {
				t.Errorf("goroutine %d: result currency is %s, want USD", i, m.Currency())
			}
		}
	}

	// Test currency mismatch is detected under concurrency.
	eurMoney := New(decimal.NewFromInt(100), EUR)

	var wg2 sync.WaitGroup
	mismatchErrors := make([]error, goroutines)
	wg2.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg2.Done()
			_, err := base.Add(eurMoney)
			mismatchErrors[idx] = err
		}(i)
	}

	wg2.Wait()

	for i, err := range mismatchErrors {
		if err == nil {
			t.Errorf("goroutine %d: expected currency mismatch error, got nil", i)
		}
	}
}
