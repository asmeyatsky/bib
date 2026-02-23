package valueobject

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// PostingPair represents a double-entry posting: one debit and one credit.
// Immutable value object.
type PostingPair struct {
	debitAccount  AccountCode
	creditAccount AccountCode
	amount        decimal.Decimal
	currency      string
	description   string
}

func NewPostingPair(debit, credit AccountCode, amount decimal.Decimal, currency, description string) (PostingPair, error) {
	if debit.IsZero() {
		return PostingPair{}, fmt.Errorf("debit account code is required")
	}
	if credit.IsZero() {
		return PostingPair{}, fmt.Errorf("credit account code is required")
	}
	if debit.Equal(credit) {
		return PostingPair{}, fmt.Errorf("debit and credit accounts must be different")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return PostingPair{}, fmt.Errorf("posting amount must be positive, got %s", amount.String())
	}
	if currency == "" {
		return PostingPair{}, fmt.Errorf("currency is required")
	}
	return PostingPair{
		debitAccount:  debit,
		creditAccount: credit,
		amount:        amount,
		currency:      currency,
		description:   description,
	}, nil
}

func (p PostingPair) DebitAccount() AccountCode  { return p.debitAccount }
func (p PostingPair) CreditAccount() AccountCode { return p.creditAccount }
func (p PostingPair) Amount() decimal.Decimal    { return p.amount }
func (p PostingPair) Currency() string           { return p.currency }
func (p PostingPair) Description() string        { return p.description }

func (p PostingPair) String() string {
	return fmt.Sprintf("DR %s / CR %s: %s %s", p.debitAccount, p.creditAccount, p.amount, p.currency)
}
