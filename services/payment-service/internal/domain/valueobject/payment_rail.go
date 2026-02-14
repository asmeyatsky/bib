package valueobject

import "fmt"

// PaymentRail represents the payment network/channel used to process a payment.
type PaymentRail struct {
	value string
}

var (
	RailACH      = PaymentRail{"ACH"}
	RailFedNow   = PaymentRail{"FEDNOW"}
	RailSWIFT    = PaymentRail{"SWIFT"}
	RailSEPA     = PaymentRail{"SEPA"}
	RailCHIPS    = PaymentRail{"CHIPS"}
	RailInternal = PaymentRail{"INTERNAL"}
)

var validRails = map[string]PaymentRail{
	"ACH":      RailACH,
	"FEDNOW":   RailFedNow,
	"SWIFT":    RailSWIFT,
	"SEPA":     RailSEPA,
	"CHIPS":    RailCHIPS,
	"INTERNAL": RailInternal,
}

// NewPaymentRail validates and creates a PaymentRail from a string.
func NewPaymentRail(s string) (PaymentRail, error) {
	if rail, ok := validRails[s]; ok {
		return rail, nil
	}
	return PaymentRail{}, fmt.Errorf("invalid payment rail: %q", s)
}

// String returns the string representation of the payment rail.
func (r PaymentRail) String() string {
	return r.value
}

// IsZero returns true if the payment rail is uninitialized.
func (r PaymentRail) IsZero() bool {
	return r.value == ""
}
