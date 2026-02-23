package service

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Nostro Reconciliation Domain Service
// ---------------------------------------------------------------------------

// ReconciliationStatus represents the outcome of reconciling a single entry.
type ReconciliationStatus string

const (
	ReconciliationMatched      ReconciliationStatus = "MATCHED"
	ReconciliationUnmatched    ReconciliationStatus = "UNMATCHED"
	ReconciliationAmountDiff   ReconciliationStatus = "AMOUNT_MISMATCH"
	ReconciliationMissingLocal ReconciliationStatus = "MISSING_LOCAL"
)

// ExternalStatementEntry represents a single entry from an external bank
// statement (e.g. parsed from an MT950 message).
type ExternalStatementEntry struct {
	Reference   string
	ValueDate   time.Time
	DebitCredit string // "D" or "C"
	Amount      decimal.Decimal
	Details     string
}

// InternalLedgerEntry represents an internal ledger entry used for
// reconciliation matching.
type InternalLedgerEntry struct {
	EntryID     string
	Reference   string
	ValueDate   time.Time
	DebitCredit string // "D" or "C"
	Amount      decimal.Decimal
	Description string
}

// ReconciliationResult represents the outcome of comparing a single external
// statement entry against internal records.
type ReconciliationResult struct {
	ExternalEntry ExternalStatementEntry
	InternalEntry *InternalLedgerEntry // nil if no match found
	Status        ReconciliationStatus
	AmountDelta   decimal.Decimal // non-zero for AMOUNT_MISMATCH
	Remarks       string
}

// ReconciliationSummary aggregates the results of a full reconciliation run.
type ReconciliationSummary struct {
	StatementDate    time.Time
	AccountID        string
	Results          []ReconciliationResult
	TotalExternal    int
	TotalInternal    int
	Matched          int
	AmountMismatches int
	MissingLocal     int
	UnmatchedLocal   int
}

// NostroReconciliation is a domain service that compares internal ledger
// entries against external bank statement entries to identify matches and
// discrepancies. It is used for nostro account reconciliation.
type NostroReconciliation struct{}

// NewNostroReconciliation creates a new reconciliation service instance.
func NewNostroReconciliation() *NostroReconciliation {
	return &NostroReconciliation{}
}

// Reconcile compares external statement entries against internal ledger entries
// and produces a reconciliation summary.
//
// Matching strategy:
//  1. Exact match on reference, debit/credit direction, and amount.
//  2. Reference match with amount mismatch (flagged for investigation).
//  3. Unmatched external entries are reported as MISSING_LOCAL.
//  4. Unmatched internal entries are counted as UnmatchedLocal.
func (r *NostroReconciliation) Reconcile(
	accountID string,
	statementDate time.Time,
	externalEntries []ExternalStatementEntry,
	internalEntries []InternalLedgerEntry,
) (ReconciliationSummary, error) {
	if accountID == "" {
		return ReconciliationSummary{}, fmt.Errorf("account ID is required")
	}

	// Build a map of internal entries by reference for O(1) lookup.
	internalByRef := make(map[string][]InternalLedgerEntry)
	for _, ie := range internalEntries {
		internalByRef[ie.Reference] = append(internalByRef[ie.Reference], ie)
	}

	// Track which internal entries have been matched.
	matchedInternalIDs := make(map[string]bool)

	summary := ReconciliationSummary{
		AccountID:     accountID,
		StatementDate: statementDate,
		TotalExternal: len(externalEntries),
		TotalInternal: len(internalEntries),
	}

	for _, ext := range externalEntries {
		result := ReconciliationResult{
			ExternalEntry: ext,
		}

		candidates, found := internalByRef[ext.Reference]
		if !found || len(candidates) == 0 {
			result.Status = ReconciliationMissingLocal
			result.Remarks = fmt.Sprintf("no internal entry found for reference %s", ext.Reference)
			summary.MissingLocal++
			summary.Results = append(summary.Results, result)
			continue
		}

		// Find the best matching candidate.
		matched := false
		for i, candidate := range candidates {
			if matchedInternalIDs[candidate.EntryID] {
				continue
			}

			// Check direction match.
			if candidate.DebitCredit != ext.DebitCredit {
				continue
			}

			ie := candidates[i]
			result.InternalEntry = &ie

			// Check amount match.
			if candidate.Amount.Equal(ext.Amount) {
				result.Status = ReconciliationMatched
				result.Remarks = "exact match"
				matchedInternalIDs[candidate.EntryID] = true
				summary.Matched++
				matched = true
				break
			}

			// Amount mismatch on same reference.
			result.Status = ReconciliationAmountDiff
			result.AmountDelta = ext.Amount.Sub(candidate.Amount)
			result.Remarks = fmt.Sprintf("amount differs by %s", result.AmountDelta)
			matchedInternalIDs[candidate.EntryID] = true
			summary.AmountMismatches++
			matched = true
			break
		}

		if !matched {
			result.Status = ReconciliationMissingLocal
			result.Remarks = fmt.Sprintf("no unmatched internal entry for reference %s with direction %s",
				ext.Reference, ext.DebitCredit)
			summary.MissingLocal++
		}

		summary.Results = append(summary.Results, result)
	}

	// Count unmatched internal entries.
	for _, ie := range internalEntries {
		if !matchedInternalIDs[ie.EntryID] {
			summary.UnmatchedLocal++
		}
	}

	return summary, nil
}
