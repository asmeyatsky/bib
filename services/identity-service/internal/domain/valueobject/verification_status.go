package valueobject

import "fmt"

// VerificationStatus represents the lifecycle state of a verification or check.
type VerificationStatus struct {
	value string
}

var (
	StatusPending    = VerificationStatus{"PENDING"}
	StatusInProgress = VerificationStatus{"IN_PROGRESS"}
	StatusApproved   = VerificationStatus{"APPROVED"}
	StatusRejected   = VerificationStatus{"REJECTED"}
	StatusExpired    = VerificationStatus{"EXPIRED"}
)

// validStatuses is the set of all known verification statuses.
var validStatuses = map[string]VerificationStatus{
	"PENDING":     StatusPending,
	"IN_PROGRESS": StatusInProgress,
	"APPROVED":    StatusApproved,
	"REJECTED":    StatusRejected,
	"EXPIRED":     StatusExpired,
}

// NewVerificationStatus creates a VerificationStatus from a string, returning an error for unknown values.
func NewVerificationStatus(s string) (VerificationStatus, error) {
	vs, ok := validStatuses[s]
	if !ok {
		return VerificationStatus{}, fmt.Errorf("unknown verification status: %q", s)
	}
	return vs, nil
}

// String returns the string representation of the verification status.
func (s VerificationStatus) String() string {
	return s.value
}

// IsTerminal returns true if this status represents a final state (APPROVED, REJECTED, EXPIRED).
func (s VerificationStatus) IsTerminal() bool {
	return s == StatusApproved || s == StatusRejected || s == StatusExpired
}

// Equal returns true if two statuses are the same.
func (s VerificationStatus) Equal(other VerificationStatus) bool {
	return s.value == other.value
}
