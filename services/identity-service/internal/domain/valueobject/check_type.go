package valueobject

import "fmt"

// CheckType represents the type of verification check to perform.
type CheckType struct {
	value string
}

var (
	CheckTypeDocument  = CheckType{"DOCUMENT"}
	CheckTypeSelfie    = CheckType{"SELFIE"}
	CheckTypeWatchlist = CheckType{"WATCHLIST"}
	CheckTypeAddress   = CheckType{"ADDRESS"}
)

// validCheckTypes is the set of all known check types.
var validCheckTypes = map[string]CheckType{
	"DOCUMENT":  CheckTypeDocument,
	"SELFIE":    CheckTypeSelfie,
	"WATCHLIST": CheckTypeWatchlist,
	"ADDRESS":   CheckTypeAddress,
}

// NewCheckType creates a CheckType from a string, returning an error for unknown types.
func NewCheckType(s string) (CheckType, error) {
	ct, ok := validCheckTypes[s]
	if !ok {
		return CheckType{}, fmt.Errorf("unknown check type: %q", s)
	}
	return ct, nil
}

// String returns the string representation of the check type.
func (ct CheckType) String() string {
	return ct.value
}

// Equal returns true if two check types are the same.
func (ct CheckType) Equal(other CheckType) bool {
	return ct.value == other.value
}

// DefaultCheckTypes returns the standard set of checks applied to a new verification.
func DefaultCheckTypes() []CheckType {
	return []CheckType{
		CheckTypeDocument,
		CheckTypeSelfie,
		CheckTypeWatchlist,
	}
}
