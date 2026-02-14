package valueobject

import (
	"fmt"
	"regexp"
)

// AccountCode represents a general ledger account code (e.g., "1000-001").
// Immutable value object with unexported fields.
type AccountCode struct {
	code string
}

var accountCodeRegex = regexp.MustCompile(`^[0-9]{4}(-[0-9]{3})?$`)

func NewAccountCode(code string) (AccountCode, error) {
	if !accountCodeRegex.MatchString(code) {
		return AccountCode{}, fmt.Errorf("invalid account code %q: must match pattern NNNN or NNNN-NNN", code)
	}
	return AccountCode{code: code}, nil
}

func MustAccountCode(code string) AccountCode {
	ac, err := NewAccountCode(code)
	if err != nil {
		panic(err)
	}
	return ac
}

func (a AccountCode) String() string { return a.code }
func (a AccountCode) Code() string   { return a.code }
func (a AccountCode) IsZero() bool   { return a.code == "" }

func (a AccountCode) Equal(other AccountCode) bool {
	return a.code == other.code
}
