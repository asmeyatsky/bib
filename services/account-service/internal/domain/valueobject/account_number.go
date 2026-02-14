package valueobject

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const (
	accountNumberPrefix  = "BIB"
	accountNumberCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	accountNumberPattern = `^BIB-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`
)

var accountNumberRegex = regexp.MustCompile(accountNumberPattern)

// AccountNumber is an immutable value object representing a unique account identifier.
// Format: BIB-XXXX-XXXX-XXXX where X is alphanumeric (uppercase).
type AccountNumber struct {
	value string
}

// NewAccountNumber generates a new random AccountNumber.
func NewAccountNumber() AccountNumber {
	segments := make([]string, 3)
	for i := range segments {
		segments[i] = randomSegment(4)
	}
	return AccountNumber{
		value: fmt.Sprintf("%s-%s-%s-%s", accountNumberPrefix, segments[0], segments[1], segments[2]),
	}
}

// AccountNumberFromString validates and creates an AccountNumber from a string.
func AccountNumberFromString(s string) (AccountNumber, error) {
	s = strings.TrimSpace(s)
	if !accountNumberRegex.MatchString(s) {
		return AccountNumber{}, fmt.Errorf("invalid account number format %q: expected BIB-XXXX-XXXX-XXXX", s)
	}
	return AccountNumber{value: s}, nil
}

// String returns the string representation of the account number.
func (n AccountNumber) String() string {
	return n.value
}

// IsZero returns true if the account number is empty.
func (n AccountNumber) IsZero() bool {
	return n.value == ""
}

// Equal returns true if two account numbers are equal.
func (n AccountNumber) Equal(other AccountNumber) bool {
	return n.value == other.value
}

func randomSegment(length int) string {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(accountNumberCharset)))
	for i := range result {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			panic(fmt.Sprintf("failed to generate random number: %v", err))
		}
		result[i] = accountNumberCharset[n.Int64()]
	}
	return string(result)
}
