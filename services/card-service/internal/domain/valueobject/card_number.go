package valueobject

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var fourDigitsRegex = regexp.MustCompile(`^\d{4}$`)

// CardNumber holds the masked card information.
// We never store the full PAN -- only the last four digits plus expiry.
// This is an immutable value object.
type CardNumber struct {
	lastFour    string
	expiryMonth string
	expiryYear  string
}

// NewCardNumber creates a validated CardNumber.
// lastFour must be exactly 4 digits. expiryMonth must be 01-12. expiryYear must be 4 digits.
func NewCardNumber(lastFour, expiryMonth, expiryYear string) (CardNumber, error) {
	if !fourDigitsRegex.MatchString(lastFour) {
		return CardNumber{}, fmt.Errorf("last four must be exactly 4 digits, got: %q", lastFour)
	}

	month, err := strconv.Atoi(expiryMonth)
	if err != nil || month < 1 || month > 12 {
		return CardNumber{}, fmt.Errorf("expiry month must be 01-12, got: %q", expiryMonth)
	}

	if !fourDigitsRegex.MatchString(expiryYear) {
		return CardNumber{}, fmt.Errorf("expiry year must be exactly 4 digits, got: %q", expiryYear)
	}

	return CardNumber{
		lastFour:    lastFour,
		expiryMonth: expiryMonth,
		expiryYear:  expiryYear,
	}, nil
}

// LastFour returns the last four digits of the card number.
func (cn CardNumber) LastFour() string {
	return cn.lastFour
}

// ExpiryMonth returns the expiry month (e.g., "01" through "12").
func (cn CardNumber) ExpiryMonth() string {
	return cn.expiryMonth
}

// ExpiryYear returns the expiry year (e.g., "2027").
func (cn CardNumber) ExpiryYear() string {
	return cn.expiryYear
}

// IsExpired returns true if the card has expired relative to the given time.
func (cn CardNumber) IsExpired(now time.Time) bool {
	year, err := strconv.Atoi(cn.expiryYear)
	if err != nil {
		return true
	}
	month, err := strconv.Atoi(cn.expiryMonth)
	if err != nil {
		return true
	}

	// Card is valid through the last day of the expiry month.
	expiryEnd := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	return now.UTC().After(expiryEnd) || now.UTC().Equal(expiryEnd)
}

// Masked returns a masked card representation like **** **** **** 1234.
func (cn CardNumber) Masked() string {
	return fmt.Sprintf("**** **** **** %s", cn.lastFour)
}

// String returns the masked representation.
func (cn CardNumber) String() string {
	return cn.Masked()
}
