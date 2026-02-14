package valueobject

import (
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// LoanApplicationStatus – immutable value object
// ---------------------------------------------------------------------------

// LoanApplicationStatus represents the lifecycle stage of a loan application.
type LoanApplicationStatus struct {
	value string
}

const (
	loanAppStatusSubmitted   = "SUBMITTED"
	loanAppStatusUnderReview = "UNDER_REVIEW"
	loanAppStatusApproved    = "APPROVED"
	loanAppStatusRejected    = "REJECTED"
	loanAppStatusDisbursed   = "DISBURSED"
)

var (
	LoanApplicationStatusSubmitted   = LoanApplicationStatus{value: loanAppStatusSubmitted}
	LoanApplicationStatusUnderReview = LoanApplicationStatus{value: loanAppStatusUnderReview}
	LoanApplicationStatusApproved    = LoanApplicationStatus{value: loanAppStatusApproved}
	LoanApplicationStatusRejected    = LoanApplicationStatus{value: loanAppStatusRejected}
	LoanApplicationStatusDisbursed   = LoanApplicationStatus{value: loanAppStatusDisbursed}
)

var validLoanApplicationStatuses = map[string]LoanApplicationStatus{
	loanAppStatusSubmitted:   LoanApplicationStatusSubmitted,
	loanAppStatusUnderReview: LoanApplicationStatusUnderReview,
	loanAppStatusApproved:    LoanApplicationStatusApproved,
	loanAppStatusRejected:    LoanApplicationStatusRejected,
	loanAppStatusDisbursed:   LoanApplicationStatusDisbursed,
}

// NewLoanApplicationStatus creates a LoanApplicationStatus from a raw string.
func NewLoanApplicationStatus(s string) (LoanApplicationStatus, error) {
	v, ok := validLoanApplicationStatuses[s]
	if !ok {
		return LoanApplicationStatus{}, fmt.Errorf("invalid loan application status: %q", s)
	}
	return v, nil
}

// String returns the string representation of the status.
func (s LoanApplicationStatus) String() string { return s.value }

// IsZero returns true if the status has not been initialised.
func (s LoanApplicationStatus) IsZero() bool { return s.value == "" }

// Equal returns true when both statuses carry the same value.
func (s LoanApplicationStatus) Equal(other LoanApplicationStatus) bool {
	return s.value == other.value
}

// ---------------------------------------------------------------------------
// LoanStatus – immutable value object
// ---------------------------------------------------------------------------

// LoanStatus represents the lifecycle stage of an active loan.
type LoanStatus struct {
	value string
}

const (
	loanStatusActive     = "ACTIVE"
	loanStatusDelinquent = "DELINQUENT"
	loanStatusDefault    = "DEFAULT"
	loanStatusPaidOff    = "PAID_OFF"
	loanStatusWrittenOff = "WRITTEN_OFF"
)

var (
	LoanStatusActive     = LoanStatus{value: loanStatusActive}
	LoanStatusDelinquent = LoanStatus{value: loanStatusDelinquent}
	LoanStatusDefault    = LoanStatus{value: loanStatusDefault}
	LoanStatusPaidOff    = LoanStatus{value: loanStatusPaidOff}
	LoanStatusWrittenOff = LoanStatus{value: loanStatusWrittenOff}
)

var validLoanStatuses = map[string]LoanStatus{
	loanStatusActive:     LoanStatusActive,
	loanStatusDelinquent: LoanStatusDelinquent,
	loanStatusDefault:    LoanStatusDefault,
	loanStatusPaidOff:    LoanStatusPaidOff,
	loanStatusWrittenOff: LoanStatusWrittenOff,
}

// NewLoanStatus creates a LoanStatus from a raw string.
func NewLoanStatus(s string) (LoanStatus, error) {
	v, ok := validLoanStatuses[s]
	if !ok {
		return LoanStatus{}, fmt.Errorf("invalid loan status: %q", s)
	}
	return v, nil
}

// String returns the string representation of the status.
func (s LoanStatus) String() string { return s.value }

// IsZero returns true if the status has not been initialised.
func (s LoanStatus) IsZero() bool { return s.value == "" }

// Equal returns true when both statuses carry the same value.
func (s LoanStatus) Equal(other LoanStatus) bool { return s.value == other.value }

// ---------------------------------------------------------------------------
// CollectionCaseStatus – immutable value object
// ---------------------------------------------------------------------------

// CollectionCaseStatus represents the lifecycle stage of a collection case.
type CollectionCaseStatus struct {
	value string
}

const (
	collectionStatusOpen       = "OPEN"
	collectionStatusInProgress = "IN_PROGRESS"
	collectionStatusResolved   = "RESOLVED"
	collectionStatusClosed     = "CLOSED"
)

var (
	CollectionCaseStatusOpen       = CollectionCaseStatus{value: collectionStatusOpen}
	CollectionCaseStatusInProgress = CollectionCaseStatus{value: collectionStatusInProgress}
	CollectionCaseStatusResolved   = CollectionCaseStatus{value: collectionStatusResolved}
	CollectionCaseStatusClosed     = CollectionCaseStatus{value: collectionStatusClosed}
)

var validCollectionCaseStatuses = map[string]CollectionCaseStatus{
	collectionStatusOpen:       CollectionCaseStatusOpen,
	collectionStatusInProgress: CollectionCaseStatusInProgress,
	collectionStatusResolved:   CollectionCaseStatusResolved,
	collectionStatusClosed:     CollectionCaseStatusClosed,
}

// NewCollectionCaseStatus creates a CollectionCaseStatus from a raw string.
func NewCollectionCaseStatus(s string) (CollectionCaseStatus, error) {
	v, ok := validCollectionCaseStatuses[s]
	if !ok {
		return CollectionCaseStatus{}, fmt.Errorf("invalid collection case status: %q", s)
	}
	return v, nil
}

// String returns the string representation.
func (s CollectionCaseStatus) String() string { return s.value }

// IsZero returns true when not initialised.
func (s CollectionCaseStatus) IsZero() bool { return s.value == "" }

// Equal returns true when both statuses match.
func (s CollectionCaseStatus) Equal(other CollectionCaseStatus) bool {
	return s.value == other.value
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)
