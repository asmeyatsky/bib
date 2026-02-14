package money

import (
	"testing"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Currency
// ---------------------------------------------------------------------------

func TestNewCurrency_Valid(t *testing.T) {
	tests := []string{"USD", "EUR", "GBP", "JPY", "CHF"}
	for _, code := range tests {
		c, err := NewCurrency(code)
		if err != nil {
			t.Errorf("NewCurrency(%q) unexpected error: %v", code, err)
		}
		if c.Code() != code {
			t.Errorf("NewCurrency(%q).Code() = %q, want %q", code, c.Code(), code)
		}
		if c.String() != code {
			t.Errorf("NewCurrency(%q).String() = %q, want %q", code, c.String(), code)
		}
	}
}

func TestNewCurrency_Invalid(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"empty", ""},
		{"lowercase", "usd"},
		{"mixed case", "Usd"},
		{"too short", "US"},
		{"too long", "USDD"},
		{"digits", "US1"},
		{"special chars", "U$D"},
		{"spaces", "U S"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCurrency(tt.code)
			if err == nil {
				t.Errorf("NewCurrency(%q) expected error, got nil", tt.code)
			}
		})
	}
}

func TestMustCurrency_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustCurrency(\"bad\") did not panic")
		}
	}()
	MustCurrency("bad")
}

// ---------------------------------------------------------------------------
// NewFromString
// ---------------------------------------------------------------------------

func TestNewFromString_Valid(t *testing.T) {
	tests := []struct {
		amount   string
		currency string
		want     string
	}{
		{"100", "USD", "100.0000 USD"},
		{"0", "EUR", "0.0000 EUR"},
		{"-50.5", "GBP", "-50.5000 GBP"},
		{"99.9999", "USD", "99.9999 USD"},
		{"0.001", "JPY", "0.0010 JPY"},
	}
	for _, tt := range tests {
		m, err := NewFromString(tt.amount, tt.currency)
		if err != nil {
			t.Errorf("NewFromString(%q, %q) unexpected error: %v", tt.amount, tt.currency, err)
			continue
		}
		if got := m.String(); got != tt.want {
			t.Errorf("NewFromString(%q, %q).String() = %q, want %q", tt.amount, tt.currency, got, tt.want)
		}
	}
}

func TestNewFromString_InvalidAmount(t *testing.T) {
	_, err := NewFromString("not-a-number", "USD")
	if err == nil {
		t.Error("NewFromString with invalid amount expected error, got nil")
	}
}

func TestNewFromString_InvalidCurrency(t *testing.T) {
	_, err := NewFromString("100", "bad")
	if err == nil {
		t.Error("NewFromString with invalid currency expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Zero / New
// ---------------------------------------------------------------------------

func TestZero(t *testing.T) {
	z := Zero(USD)
	if !z.IsZero() {
		t.Error("Zero(USD).IsZero() = false, want true")
	}
	if z.Currency().Code() != "USD" {
		t.Errorf("Zero(USD).Currency().Code() = %q, want %q", z.Currency().Code(), "USD")
	}
}

func TestNew(t *testing.T) {
	amt := decimal.NewFromInt(42)
	m := New(amt, EUR)
	if !m.Amount().Equal(amt) {
		t.Errorf("New amount = %s, want %s", m.Amount(), amt)
	}
	if m.Currency().Code() != "EUR" {
		t.Errorf("New currency = %q, want %q", m.Currency().Code(), "EUR")
	}
}

// ---------------------------------------------------------------------------
// Predicates: IsZero, IsPositive, IsNegative
// ---------------------------------------------------------------------------

func TestIsZero(t *testing.T) {
	z := Zero(USD)
	if !z.IsZero() {
		t.Error("expected IsZero true")
	}
	p := New(decimal.NewFromInt(1), USD)
	if p.IsZero() {
		t.Error("expected IsZero false for 1")
	}
}

func TestIsPositive(t *testing.T) {
	p := New(decimal.NewFromInt(10), USD)
	if !p.IsPositive() {
		t.Error("expected IsPositive true for 10")
	}
	z := Zero(USD)
	if z.IsPositive() {
		t.Error("expected IsPositive false for 0")
	}
	n := New(decimal.NewFromInt(-1), USD)
	if n.IsPositive() {
		t.Error("expected IsPositive false for -1")
	}
}

func TestIsNegative(t *testing.T) {
	n := New(decimal.NewFromInt(-5), USD)
	if !n.IsNegative() {
		t.Error("expected IsNegative true for -5")
	}
	z := Zero(USD)
	if z.IsNegative() {
		t.Error("expected IsNegative false for 0")
	}
	p := New(decimal.NewFromInt(3), USD)
	if p.IsNegative() {
		t.Error("expected IsNegative false for 3")
	}
}

// ---------------------------------------------------------------------------
// Arithmetic: Add, Subtract, Multiply, Negate, Abs
// ---------------------------------------------------------------------------

func TestAdd_SameCurrency(t *testing.T) {
	a := New(decimal.NewFromInt(10), USD)
	b := New(decimal.NewFromInt(20), USD)
	got, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add unexpected error: %v", err)
	}
	want := decimal.NewFromInt(30)
	if !got.Amount().Equal(want) {
		t.Errorf("Add amount = %s, want %s", got.Amount(), want)
	}
	if got.Currency().Code() != "USD" {
		t.Errorf("Add currency = %q, want USD", got.Currency().Code())
	}
}

func TestAdd_CurrencyMismatch(t *testing.T) {
	a := New(decimal.NewFromInt(10), USD)
	b := New(decimal.NewFromInt(20), EUR)
	_, err := a.Add(b)
	if err == nil {
		t.Error("Add with mismatched currencies expected error, got nil")
	}
}

func TestSubtract_SameCurrency(t *testing.T) {
	a := New(decimal.NewFromInt(30), GBP)
	b := New(decimal.NewFromInt(10), GBP)
	got, err := a.Subtract(b)
	if err != nil {
		t.Fatalf("Subtract unexpected error: %v", err)
	}
	want := decimal.NewFromInt(20)
	if !got.Amount().Equal(want) {
		t.Errorf("Subtract amount = %s, want %s", got.Amount(), want)
	}
}

func TestSubtract_CurrencyMismatch(t *testing.T) {
	a := New(decimal.NewFromInt(30), GBP)
	b := New(decimal.NewFromInt(10), USD)
	_, err := a.Subtract(b)
	if err == nil {
		t.Error("Subtract with mismatched currencies expected error, got nil")
	}
}

func TestMultiply(t *testing.T) {
	m := New(decimal.NewFromInt(50), USD)
	factor := decimal.NewFromFloat(1.5)
	got := m.Multiply(factor)
	want := decimal.NewFromFloat(75)
	if !got.Amount().Equal(want) {
		t.Errorf("Multiply amount = %s, want %s", got.Amount(), want)
	}
	if got.Currency().Code() != "USD" {
		t.Errorf("Multiply currency = %q, want USD", got.Currency().Code())
	}
}

func TestMultiply_ByZero(t *testing.T) {
	m := New(decimal.NewFromInt(100), EUR)
	got := m.Multiply(decimal.Zero)
	if !got.IsZero() {
		t.Errorf("Multiply by zero = %s, want 0", got.Amount())
	}
}

func TestNegate(t *testing.T) {
	m := New(decimal.NewFromInt(42), USD)
	neg := m.Negate()
	if !neg.Amount().Equal(decimal.NewFromInt(-42)) {
		t.Errorf("Negate amount = %s, want -42", neg.Amount())
	}

	// Double negate returns original value.
	doubleNeg := neg.Negate()
	if !doubleNeg.Amount().Equal(decimal.NewFromInt(42)) {
		t.Errorf("double Negate amount = %s, want 42", doubleNeg.Amount())
	}
}

func TestNegate_Zero(t *testing.T) {
	z := Zero(USD)
	neg := z.Negate()
	if !neg.IsZero() {
		t.Errorf("Negate zero = %s, want 0", neg.Amount())
	}
}

func TestAbs_Positive(t *testing.T) {
	m := New(decimal.NewFromInt(10), USD)
	got := m.Abs()
	if !got.Amount().Equal(decimal.NewFromInt(10)) {
		t.Errorf("Abs of positive = %s, want 10", got.Amount())
	}
}

func TestAbs_Negative(t *testing.T) {
	m := New(decimal.NewFromInt(-10), USD)
	got := m.Abs()
	if !got.Amount().Equal(decimal.NewFromInt(10)) {
		t.Errorf("Abs of negative = %s, want 10", got.Amount())
	}
}

func TestAbs_Zero(t *testing.T) {
	z := Zero(EUR)
	got := z.Abs()
	if !got.IsZero() {
		t.Errorf("Abs of zero = %s, want 0", got.Amount())
	}
}

// ---------------------------------------------------------------------------
// Equal
// ---------------------------------------------------------------------------

func TestEqual_SameAmountAndCurrency(t *testing.T) {
	a := New(decimal.NewFromInt(100), USD)
	b := New(decimal.NewFromInt(100), USD)
	if !a.Equal(b) {
		t.Error("expected Equal true for same amount and currency")
	}
}

func TestEqual_DifferentAmount(t *testing.T) {
	a := New(decimal.NewFromInt(100), USD)
	b := New(decimal.NewFromInt(200), USD)
	if a.Equal(b) {
		t.Error("expected Equal false for different amounts")
	}
}

func TestEqual_DifferentCurrency(t *testing.T) {
	a := New(decimal.NewFromInt(100), USD)
	b := New(decimal.NewFromInt(100), EUR)
	if a.Equal(b) {
		t.Error("expected Equal false for different currencies")
	}
}

func TestEqual_DecimalEquivalence(t *testing.T) {
	// 10 and 10.00 should be equal via decimal.Equal.
	a := New(decimal.NewFromInt(10), USD)
	b, err := NewFromString("10.00", "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Equal(b) {
		t.Error("expected Equal true for decimal-equivalent amounts (10 vs 10.00)")
	}
}

// ---------------------------------------------------------------------------
// String
// ---------------------------------------------------------------------------

func TestString(t *testing.T) {
	tests := []struct {
		amount   decimal.Decimal
		currency Currency
		want     string
	}{
		{decimal.NewFromInt(100), USD, "100.0000 USD"},
		{decimal.NewFromFloat(0.5), EUR, "0.5000 EUR"},
		{decimal.NewFromInt(-25), GBP, "-25.0000 GBP"},
		{decimal.Zero, USD, "0.0000 USD"},
		{decimal.NewFromFloat(99.9999), USD, "99.9999 USD"},
	}
	for _, tt := range tests {
		m := New(tt.amount, tt.currency)
		if got := m.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Immutability: operations must not mutate the original
// ---------------------------------------------------------------------------

func TestImmutability_Add(t *testing.T) {
	original := New(decimal.NewFromInt(10), USD)
	other := New(decimal.NewFromInt(5), USD)
	_, _ = original.Add(other)
	if !original.Amount().Equal(decimal.NewFromInt(10)) {
		t.Error("Add mutated the original Money value")
	}
}

func TestImmutability_Negate(t *testing.T) {
	original := New(decimal.NewFromInt(10), USD)
	_ = original.Negate()
	if !original.Amount().Equal(decimal.NewFromInt(10)) {
		t.Error("Negate mutated the original Money value")
	}
}

func TestImmutability_Multiply(t *testing.T) {
	original := New(decimal.NewFromInt(10), USD)
	_ = original.Multiply(decimal.NewFromInt(3))
	if !original.Amount().Equal(decimal.NewFromInt(10)) {
		t.Error("Multiply mutated the original Money value")
	}
}

// ---------------------------------------------------------------------------
// Package-level currency vars
// ---------------------------------------------------------------------------

func TestPackageCurrencies(t *testing.T) {
	if USD.Code() != "USD" {
		t.Errorf("USD.Code() = %q, want USD", USD.Code())
	}
	if EUR.Code() != "EUR" {
		t.Errorf("EUR.Code() = %q, want EUR", EUR.Code())
	}
	if GBP.Code() != "GBP" {
		t.Errorf("GBP.Code() = %q, want GBP", GBP.Code())
	}
}
