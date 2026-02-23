package valueobject

import "fmt"

// CardStatus represents the lifecycle status of a card.
// This is an immutable value object.
type CardStatus string

const (
	CardStatusPending  CardStatus = "PENDING"
	CardStatusActive   CardStatus = "ACTIVE"
	CardStatusFrozen   CardStatus = "FROZEN"
	CardStatusCanceled CardStatus = "CANCELED"
	CardStatusExpired  CardStatus = "EXPIRED"
)

// validCardStatuses contains all valid card statuses for validation.
var validCardStatuses = map[CardStatus]bool{
	CardStatusPending:  true,
	CardStatusActive:   true,
	CardStatusFrozen:   true,
	CardStatusCanceled: true,
	CardStatusExpired:  true,
}

// NewCardStatus creates a validated CardStatus from a string.
func NewCardStatus(s string) (CardStatus, error) {
	cs := CardStatus(s)
	if !validCardStatuses[cs] {
		return "", fmt.Errorf("invalid card status: %q", s)
	}
	return cs, nil
}

// String returns the string representation of the CardStatus.
func (cs CardStatus) String() string {
	return string(cs)
}

// IsUsable returns true if the card can be used for transactions.
// Only ACTIVE cards are usable.
func (cs CardStatus) IsUsable() bool {
	return cs == CardStatusActive
}
