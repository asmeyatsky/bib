package residency

import "fmt"

// PolicyViolation describes a single data residency policy violation.
type PolicyViolation struct {
	Rule        string
	Description string
	Severity    string // "ERROR" or "WARNING"
}

// PolicyResult holds the outcome of a policy evaluation.
type PolicyResult struct {
	Violations []PolicyViolation
	Allowed    bool
}

// PolicyEngine evaluates data storage and processing operations against
// jurisdiction requirements. It is the central decision point for data
// residency compliance.
type PolicyEngine struct {
	rules map[Jurisdiction]JurisdictionRule
}

// NewPolicyEngine creates a policy engine with the given jurisdiction rules.
func NewPolicyEngine(rules map[Jurisdiction]JurisdictionRule) *PolicyEngine {
	return &PolicyEngine{rules: rules}
}

// NewDefaultPolicyEngine creates a policy engine with default jurisdiction rules.
func NewDefaultPolicyEngine() *PolicyEngine {
	return NewPolicyEngine(DefaultJurisdictionRules())
}

// ValidateStorage checks if storing data of the given classification in the
// specified region is allowed under the jurisdiction's rules.
func (pe *PolicyEngine) ValidateStorage(
	jurisdiction Jurisdiction,
	classification DataClassification,
	storageRegion Region,
) PolicyResult {
	result := PolicyResult{Allowed: true}

	rule, ok := pe.rules[jurisdiction]
	if !ok {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule:        "KNOWN_JURISDICTION",
			Description: fmt.Sprintf("unknown jurisdiction: %s", jurisdiction),
			Severity:    "ERROR",
		})
		return result
	}

	// Check region is allowed for storage.
	regionAllowed := false
	for _, r := range rule.AllowedRegions {
		if r == storageRegion {
			regionAllowed = true
			break
		}
	}
	if !regionAllowed {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule:        "REGION_ALLOWED",
			Description: fmt.Sprintf("region %s not allowed for jurisdiction %s", storageRegion, jurisdiction),
			Severity:    "ERROR",
		})
	}

	// For PII and financial data, encryption at rest is mandatory in all jurisdictions.
	if (classification == ClassPII || classification == ClassFinancial) && !rule.RequiresEncryptionAtRest {
		result.Violations = append(result.Violations, PolicyViolation{
			Rule:        "ENCRYPTION_AT_REST",
			Description: "PII/financial data should be encrypted at rest",
			Severity:    "WARNING",
		})
	}

	return result
}

// ValidateProcessing checks if processing data in the given region is
// permitted under the jurisdiction's rules.
func (pe *PolicyEngine) ValidateProcessing(
	jurisdiction Jurisdiction,
	_ DataClassification,
	processingRegion Region,
) PolicyResult {
	result := PolicyResult{Allowed: true}

	rule, ok := pe.rules[jurisdiction]
	if !ok {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule:        "KNOWN_JURISDICTION",
			Description: fmt.Sprintf("unknown jurisdiction: %s", jurisdiction),
			Severity:    "ERROR",
		})
		return result
	}

	regionAllowed := false
	for _, r := range rule.AllowedProcessingRegions {
		if r == processingRegion {
			regionAllowed = true
			break
		}
	}
	if !regionAllowed {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule: "PROCESSING_REGION_ALLOWED",
			Description: fmt.Sprintf("processing in region %s not allowed for jurisdiction %s",
				processingRegion, jurisdiction),
			Severity: "ERROR",
		})
	}

	return result
}

// ValidateCrossBorderTransfer checks if transferring data from the source
// jurisdiction to the destination region is allowed.
func (pe *PolicyEngine) ValidateCrossBorderTransfer(
	sourceJurisdiction Jurisdiction,
	destinationRegion Region,
	hasConsent bool,
) PolicyResult {
	result := PolicyResult{Allowed: true}

	rule, ok := pe.rules[sourceJurisdiction]
	if !ok {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule:        "KNOWN_JURISDICTION",
			Description: fmt.Sprintf("unknown jurisdiction: %s", sourceJurisdiction),
			Severity:    "ERROR",
		})
		return result
	}

	// Check if cross-border transfer is allowed at all.
	if !rule.CrossBorderTransferAllowed {
		// Check if destination is within jurisdiction's allowed regions.
		inJurisdiction := false
		for _, r := range rule.AllowedRegions {
			if r == destinationRegion {
				inJurisdiction = true
				break
			}
		}
		if !inJurisdiction {
			result.Allowed = false
			result.Violations = append(result.Violations, PolicyViolation{
				Rule: "CROSS_BORDER_PROHIBITED",
				Description: fmt.Sprintf("cross-border transfer from %s to region %s is prohibited",
					sourceJurisdiction, destinationRegion),
				Severity: "ERROR",
			})
			return result
		}
	}

	// Check if consent is required for cross-border transfer.
	if rule.CrossBorderRequiresConsent && !hasConsent {
		// Only flag if destination is outside jurisdiction.
		inJurisdiction := false
		for _, r := range rule.AllowedRegions {
			if r == destinationRegion {
				inJurisdiction = true
				break
			}
		}
		if !inJurisdiction {
			result.Allowed = false
			result.Violations = append(result.Violations, PolicyViolation{
				Rule: "CROSS_BORDER_CONSENT",
				Description: fmt.Sprintf("cross-border transfer from %s requires data subject consent",
					sourceJurisdiction),
				Severity: "ERROR",
			})
		}
	}

	return result
}

// ValidateRetention checks if the data retention period is within limits.
func (pe *PolicyEngine) ValidateRetention(
	jurisdiction Jurisdiction,
	retentionDays int,
) PolicyResult {
	result := PolicyResult{Allowed: true}

	rule, ok := pe.rules[jurisdiction]
	if !ok {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule:        "KNOWN_JURISDICTION",
			Description: fmt.Sprintf("unknown jurisdiction: %s", jurisdiction),
			Severity:    "ERROR",
		})
		return result
	}

	if rule.MaxRetentionDays > 0 && retentionDays > rule.MaxRetentionDays {
		result.Allowed = false
		result.Violations = append(result.Violations, PolicyViolation{
			Rule: "MAX_RETENTION",
			Description: fmt.Sprintf("retention of %d days exceeds maximum %d days for jurisdiction %s",
				retentionDays, rule.MaxRetentionDays, jurisdiction),
			Severity: "ERROR",
		})
	}

	return result
}
