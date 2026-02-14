package valueobject

import "fmt"

// AccountType is an immutable value object representing the type of a bank account.
type AccountType struct {
	value string
}

// Known account types.
var (
	AccountTypeChecking = AccountType{"CHECKING"}
	AccountTypeSavings  = AccountType{"SAVINGS"}
	AccountTypeLoan     = AccountType{"LOAN"}
	AccountTypeNominal  = AccountType{"NOMINAL"}
)

var knownAccountTypes = map[string]AccountType{
	"CHECKING": AccountTypeChecking,
	"SAVINGS":  AccountTypeSavings,
	"LOAN":     AccountTypeLoan,
	"NOMINAL":  AccountTypeNominal,
}

// NewAccountType validates and creates an AccountType from a string.
func NewAccountType(s string) (AccountType, error) {
	at, ok := knownAccountTypes[s]
	if !ok {
		return AccountType{}, fmt.Errorf("unknown account type %q: expected CHECKING, SAVINGS, LOAN, or NOMINAL", s)
	}
	return at, nil
}

// String returns the string representation of the account type.
func (t AccountType) String() string {
	return t.value
}

// IsZero returns true if the account type is empty.
func (t AccountType) IsZero() bool {
	return t.value == ""
}

// Equal returns true if two account types are equal.
func (t AccountType) Equal(other AccountType) bool {
	return t.value == other.value
}
