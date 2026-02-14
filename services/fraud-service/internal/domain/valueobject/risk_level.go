package valueobject

import "fmt"

// RiskLevel is an immutable value object representing the risk classification.
type RiskLevel struct {
	value string
}

var (
	RiskLevelLow      = RiskLevel{value: "LOW"}
	RiskLevelMedium   = RiskLevel{value: "MEDIUM"}
	RiskLevelHigh     = RiskLevel{value: "HIGH"}
	RiskLevelCritical = RiskLevel{value: "CRITICAL"}
)

// RiskLevelFromString reconstructs a RiskLevel from its string representation.
func RiskLevelFromString(s string) (RiskLevel, error) {
	switch s {
	case "LOW":
		return RiskLevelLow, nil
	case "MEDIUM":
		return RiskLevelMedium, nil
	case "HIGH":
		return RiskLevelHigh, nil
	case "CRITICAL":
		return RiskLevelCritical, nil
	default:
		return RiskLevel{}, fmt.Errorf("invalid risk level: %s", s)
	}
}

// RiskLevelFromScore derives the appropriate RiskLevel from a numeric score (0-100).
func RiskLevelFromScore(score int) RiskLevel {
	switch {
	case score >= 80:
		return RiskLevelCritical
	case score >= 60:
		return RiskLevelHigh
	case score >= 35:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// String returns the string representation.
func (r RiskLevel) String() string {
	return r.value
}

// Score returns the canonical numeric score for this risk level.
// LOW=25, MEDIUM=50, HIGH=75, CRITICAL=100.
func (r RiskLevel) Score() int {
	switch r.value {
	case "LOW":
		return 25
	case "MEDIUM":
		return 50
	case "HIGH":
		return 75
	case "CRITICAL":
		return 100
	default:
		return 0
	}
}

// IsZero returns true if the RiskLevel has not been set.
func (r RiskLevel) IsZero() bool {
	return r.value == ""
}

// Equal checks equality with another RiskLevel.
func (r RiskLevel) Equal(other RiskLevel) bool {
	return r.value == other.value
}
