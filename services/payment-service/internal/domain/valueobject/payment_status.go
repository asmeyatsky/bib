package valueobject

import "fmt"

// PaymentStatus represents the lifecycle state of a payment order.
type PaymentStatus struct {
	value string
}

var (
	PaymentStatusInitiated  = PaymentStatus{"INITIATED"}
	PaymentStatusProcessing = PaymentStatus{"PROCESSING"}
	PaymentStatusSettled    = PaymentStatus{"SETTLED"}
	PaymentStatusFailed     = PaymentStatus{"FAILED"}
	PaymentStatusReversed   = PaymentStatus{"REVERSED"}
)

var validStatuses = map[string]PaymentStatus{
	"INITIATED":  PaymentStatusInitiated,
	"PROCESSING": PaymentStatusProcessing,
	"SETTLED":    PaymentStatusSettled,
	"FAILED":     PaymentStatusFailed,
	"REVERSED":   PaymentStatusReversed,
}

// NewPaymentStatus validates and creates a PaymentStatus from a string.
func NewPaymentStatus(s string) (PaymentStatus, error) {
	if status, ok := validStatuses[s]; ok {
		return status, nil
	}
	return PaymentStatus{}, fmt.Errorf("invalid payment status: %q", s)
}

// String returns the string representation of the payment status.
func (s PaymentStatus) String() string {
	return s.value
}

// IsTerminal returns true if the payment status is a terminal state (SETTLED, FAILED, or REVERSED).
func (s PaymentStatus) IsTerminal() bool {
	return s == PaymentStatusSettled || s == PaymentStatusFailed || s == PaymentStatusReversed
}

// IsZero returns true if the payment status is uninitialized.
func (s PaymentStatus) IsZero() bool {
	return s.value == ""
}
