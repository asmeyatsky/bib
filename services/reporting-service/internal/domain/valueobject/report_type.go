package valueobject

import "fmt"

// ReportType represents the type of regulatory report.
// It is an immutable value object.
type ReportType struct {
	value string
}

const (
	reportTypeCOREP  = "COREP"
	reportTypeFINREP = "FINREP"
	reportTypeMREL   = "MREL"
	reportTypeCUSTOM = "CUSTOM"
)

var (
	ReportTypeCOREP  = ReportType{value: reportTypeCOREP}
	ReportTypeFINREP = ReportType{value: reportTypeFINREP}
	ReportTypeMREL   = ReportType{value: reportTypeMREL}
	ReportTypeCUSTOM = ReportType{value: reportTypeCUSTOM}
)

var validReportTypes = map[string]ReportType{
	reportTypeCOREP:  ReportTypeCOREP,
	reportTypeFINREP: ReportTypeFINREP,
	reportTypeMREL:   ReportTypeMREL,
	reportTypeCUSTOM: ReportTypeCUSTOM,
}

// NewReportType creates a ReportType from a string, validating it is a known type.
func NewReportType(s string) (ReportType, error) {
	rt, ok := validReportTypes[s]
	if !ok {
		return ReportType{}, fmt.Errorf("invalid report type: %q", s)
	}
	return rt, nil
}

// String returns the string representation of the ReportType.
func (r ReportType) String() string {
	return r.value
}

// IsZero returns true if the ReportType has not been set.
func (r ReportType) IsZero() bool {
	return r.value == ""
}

// Equal returns true if two ReportType values are equal.
func (r ReportType) Equal(other ReportType) bool {
	return r.value == other.value
}
