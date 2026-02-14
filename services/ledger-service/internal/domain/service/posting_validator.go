package service

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// PostingValidator is a domain service that validates posting pairs.
type PostingValidator struct{}

func NewPostingValidator() *PostingValidator {
	return &PostingValidator{}
}

// ValidatePostings ensures all posting pairs are balanced (debits = credits per currency).
func (v *PostingValidator) ValidatePostings(postings []valueobject.PostingPair) error {
	if len(postings) == 0 {
		return fmt.Errorf("at least one posting pair is required")
	}

	// Each PostingPair is balanced by definition (same amount on both sides),
	// so we just validate the overall structure.
	totals := make(map[string]decimal.Decimal)
	for _, p := range postings {
		totals[p.Currency()] = totals[p.Currency()].Add(p.Amount())
	}

	// Ensure no negative totals
	for currency, total := range totals {
		if total.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("total for currency %s must be positive, got %s", currency, total)
		}
	}

	return nil
}

// ValidateNotSelfPosting ensures no posting pair has the same debit and credit.
func (v *PostingValidator) ValidateNotSelfPosting(postings []valueobject.PostingPair) error {
	for _, p := range postings {
		if p.DebitAccount().Equal(p.CreditAccount()) {
			return fmt.Errorf("self-posting not allowed: account %s", p.DebitAccount())
		}
	}
	return nil
}
