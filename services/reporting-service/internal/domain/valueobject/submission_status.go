package valueobject

import "fmt"

// SubmissionStatus represents the status of a report submission.
// It is an immutable value object.
type SubmissionStatus struct {
	value string
}

const (
	statusDraft      = "DRAFT"
	statusGenerating = "GENERATING"
	statusReady      = "READY"
	statusSubmitted  = "SUBMITTED"
	statusAccepted   = "ACCEPTED"
	statusRejected   = "REJECTED"
)

var (
	SubmissionStatusDraft      = SubmissionStatus{value: statusDraft}
	SubmissionStatusGenerating = SubmissionStatus{value: statusGenerating}
	SubmissionStatusReady      = SubmissionStatus{value: statusReady}
	SubmissionStatusSubmitted  = SubmissionStatus{value: statusSubmitted}
	SubmissionStatusAccepted   = SubmissionStatus{value: statusAccepted}
	SubmissionStatusRejected   = SubmissionStatus{value: statusRejected}
)

var validSubmissionStatuses = map[string]SubmissionStatus{
	statusDraft:      SubmissionStatusDraft,
	statusGenerating: SubmissionStatusGenerating,
	statusReady:      SubmissionStatusReady,
	statusSubmitted:  SubmissionStatusSubmitted,
	statusAccepted:   SubmissionStatusAccepted,
	statusRejected:   SubmissionStatusRejected,
}

// NewSubmissionStatus creates a SubmissionStatus from a string, validating it is known.
func NewSubmissionStatus(s string) (SubmissionStatus, error) {
	ss, ok := validSubmissionStatuses[s]
	if !ok {
		return SubmissionStatus{}, fmt.Errorf("invalid submission status: %q", s)
	}
	return ss, nil
}

// String returns the string representation of the SubmissionStatus.
func (s SubmissionStatus) String() string {
	return s.value
}

// IsZero returns true if the SubmissionStatus has not been set.
func (s SubmissionStatus) IsZero() bool {
	return s.value == ""
}

// Equal returns true if two SubmissionStatus values are equal.
func (s SubmissionStatus) Equal(other SubmissionStatus) bool {
	return s.value == other.value
}
