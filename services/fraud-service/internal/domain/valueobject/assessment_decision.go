package valueobject

import "fmt"

// AssessmentDecision is an immutable value object representing the outcome of a risk assessment.
type AssessmentDecision struct {
	value string
}

var (
	DecisionApprove = AssessmentDecision{value: "APPROVE"}
	DecisionReview  = AssessmentDecision{value: "REVIEW"}
	DecisionDecline = AssessmentDecision{value: "DECLINE"}
)

// NewApproveDecision creates an APPROVE decision.
func NewApproveDecision() AssessmentDecision {
	return DecisionApprove
}

// NewReviewDecision creates a REVIEW decision.
func NewReviewDecision() AssessmentDecision {
	return DecisionReview
}

// NewDeclineDecision creates a DECLINE decision.
func NewDeclineDecision() AssessmentDecision {
	return DecisionDecline
}

// AssessmentDecisionFromString reconstructs a decision from its string representation.
func AssessmentDecisionFromString(s string) (AssessmentDecision, error) {
	switch s {
	case "APPROVE":
		return DecisionApprove, nil
	case "REVIEW":
		return DecisionReview, nil
	case "DECLINE":
		return DecisionDecline, nil
	default:
		return AssessmentDecision{}, fmt.Errorf("invalid assessment decision: %s", s)
	}
}

// DecisionFromScore determines the decision based on a risk score.
func DecisionFromScore(score int) AssessmentDecision {
	switch {
	case score > 70:
		return DecisionDecline
	case score >= 30:
		return DecisionReview
	default:
		return DecisionApprove
	}
}

// String returns the string representation.
func (d AssessmentDecision) String() string {
	return d.value
}

// IsZero returns true if the decision has not been set.
func (d AssessmentDecision) IsZero() bool {
	return d.value == ""
}

// Equal checks equality with another AssessmentDecision.
func (d AssessmentDecision) Equal(other AssessmentDecision) bool {
	return d.value == other.value
}

// IsApproved returns true if the decision is APPROVE.
func (d AssessmentDecision) IsApproved() bool {
	return d.value == "APPROVE"
}

// IsReview returns true if the decision is REVIEW.
func (d AssessmentDecision) IsReview() bool {
	return d.value == "REVIEW"
}

// IsDeclined returns true if the decision is DECLINE.
func (d AssessmentDecision) IsDeclined() bool {
	return d.value == "DECLINE"
}
