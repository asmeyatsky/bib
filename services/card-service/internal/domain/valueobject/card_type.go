package valueobject

import "fmt"

// CardType represents the type of card issued.
// This is an immutable value object.
type CardType string

const (
	CardTypeVirtual  CardType = "VIRTUAL"
	CardTypePhysical CardType = "PHYSICAL"
)

// validCardTypes contains all valid card types for validation.
var validCardTypes = map[CardType]bool{
	CardTypeVirtual:  true,
	CardTypePhysical: true,
}

// NewCardType creates a validated CardType from a string.
func NewCardType(s string) (CardType, error) {
	ct := CardType(s)
	if !validCardTypes[ct] {
		return "", fmt.Errorf("invalid card type: %q, must be VIRTUAL or PHYSICAL", s)
	}
	return ct, nil
}

// String returns the string representation of the CardType.
func (ct CardType) String() string {
	return string(ct)
}

// IsVirtual returns true if this is a virtual card.
func (ct CardType) IsVirtual() bool {
	return ct == CardTypeVirtual
}

// IsPhysical returns true if this is a physical card.
func (ct CardType) IsPhysical() bool {
	return ct == CardTypePhysical
}
