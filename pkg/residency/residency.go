// Package residency provides data residency controls for the Bank-in-a-Box
// platform. It defines jurisdiction rules, data classification levels, and
// geofencing checks to ensure compliance with regional data sovereignty
// requirements (e.g. GDPR, CCPA, data localization laws).
package residency

import "fmt"

// Jurisdiction represents a regulatory jurisdiction with data residency rules.
type Jurisdiction string

const (
	JurisdictionUS Jurisdiction = "US"
	JurisdictionEU Jurisdiction = "EU"
	JurisdictionUK Jurisdiction = "UK"
	JurisdictionSG Jurisdiction = "SG"
	JurisdictionAU Jurisdiction = "AU"
	JurisdictionCA Jurisdiction = "CA"
	JurisdictionIN Jurisdiction = "IN"
	JurisdictionBR Jurisdiction = "BR"
)

// DataClassification represents the sensitivity level of data.
type DataClassification string

const (
	// ClassPII is personally identifiable information (names, addresses, SSN).
	ClassPII DataClassification = "PII"
	// ClassFinancial is financial data (transactions, balances, account numbers).
	ClassFinancial DataClassification = "FINANCIAL"
	// ClassOperational is operational data (logs, metrics, configuration).
	ClassOperational DataClassification = "OPERATIONAL"
	// ClassPublic is public information (product catalogs, rates).
	ClassPublic DataClassification = "PUBLIC"
)

// Region represents a cloud provider region or data center location.
type Region string

const (
	RegionUSEast1     Region = "us-east-1"
	RegionUSWest2     Region = "us-west-2"
	RegionEUWest1     Region = "eu-west-1"
	RegionEUCentral1  Region = "eu-central-1"
	RegionAPSouth1    Region = "ap-south-1"
	RegionAPSoutheast Region = "ap-southeast-1"
	RegionCACentral1  Region = "ca-central-1"
	RegionSAEast1     Region = "sa-east-1"
	RegionUKSouth     Region = "uk-south"
	RegionAUEast      Region = "au-east"
)

// JurisdictionRule defines data residency requirements for a jurisdiction.
type JurisdictionRule struct {
	// Jurisdiction is the regulatory jurisdiction.
	Jurisdiction Jurisdiction
	// AllowedRegions lists the cloud regions where data may be stored.
	AllowedRegions []Region
	// AllowedProcessingRegions lists regions where data may be processed
	// (may differ from storage regions for some jurisdictions).
	AllowedProcessingRegions []Region
	// RequiresEncryptionAtRest indicates if data must be encrypted at rest.
	RequiresEncryptionAtRest bool
	// RequiresEncryptionInTransit indicates if data must be encrypted in transit.
	RequiresEncryptionInTransit bool
	// MaxRetentionDays is the maximum data retention period (0 = no limit).
	MaxRetentionDays int
	// RequiresAuditLog indicates if access to this data must be audit-logged.
	RequiresAuditLog bool
	// CrossBorderTransferAllowed indicates if data can leave the jurisdiction.
	CrossBorderTransferAllowed bool
	// CrossBorderRequiresConsent indicates if cross-border transfer needs consent.
	CrossBorderRequiresConsent bool
}

// DefaultJurisdictionRules returns the default set of jurisdiction rules.
func DefaultJurisdictionRules() map[Jurisdiction]JurisdictionRule {
	return map[Jurisdiction]JurisdictionRule{
		JurisdictionUS: {
			Jurisdiction:               JurisdictionUS,
			AllowedRegions:             []Region{RegionUSEast1, RegionUSWest2},
			AllowedProcessingRegions:   []Region{RegionUSEast1, RegionUSWest2},
			RequiresEncryptionAtRest:   true,
			RequiresEncryptionInTransit: true,
			MaxRetentionDays:           0, // per-regulation basis
			RequiresAuditLog:           true,
			CrossBorderTransferAllowed: true,
			CrossBorderRequiresConsent: false,
		},
		JurisdictionEU: {
			Jurisdiction:               JurisdictionEU,
			AllowedRegions:             []Region{RegionEUWest1, RegionEUCentral1},
			AllowedProcessingRegions:   []Region{RegionEUWest1, RegionEUCentral1},
			RequiresEncryptionAtRest:   true,
			RequiresEncryptionInTransit: true,
			MaxRetentionDays:           0,
			RequiresAuditLog:           true,
			CrossBorderTransferAllowed: false, // GDPR: requires adequacy decision
			CrossBorderRequiresConsent: true,
		},
		JurisdictionUK: {
			Jurisdiction:               JurisdictionUK,
			AllowedRegions:             []Region{RegionUKSouth, RegionEUWest1},
			AllowedProcessingRegions:   []Region{RegionUKSouth, RegionEUWest1},
			RequiresEncryptionAtRest:   true,
			RequiresEncryptionInTransit: true,
			MaxRetentionDays:           0,
			RequiresAuditLog:           true,
			CrossBorderTransferAllowed: true,
			CrossBorderRequiresConsent: true,
		},
		JurisdictionSG: {
			Jurisdiction:               JurisdictionSG,
			AllowedRegions:             []Region{RegionAPSoutheast},
			AllowedProcessingRegions:   []Region{RegionAPSoutheast},
			RequiresEncryptionAtRest:   true,
			RequiresEncryptionInTransit: true,
			MaxRetentionDays:           0,
			RequiresAuditLog:           true,
			CrossBorderTransferAllowed: true,
			CrossBorderRequiresConsent: true,
		},
		JurisdictionIN: {
			Jurisdiction:               JurisdictionIN,
			AllowedRegions:             []Region{RegionAPSouth1},
			AllowedProcessingRegions:   []Region{RegionAPSouth1},
			RequiresEncryptionAtRest:   true,
			RequiresEncryptionInTransit: true,
			MaxRetentionDays:           0,
			RequiresAuditLog:           true,
			CrossBorderTransferAllowed: false, // RBI data localization
			CrossBorderRequiresConsent: false,
		},
	}
}

// IsRegionAllowed checks if a specific region is allowed for data storage
// under the given jurisdiction.
func IsRegionAllowed(jurisdiction Jurisdiction, region Region) bool {
	rules := DefaultJurisdictionRules()
	rule, ok := rules[jurisdiction]
	if !ok {
		return false
	}
	for _, r := range rule.AllowedRegions {
		if r == region {
			return true
		}
	}
	return false
}

// IsProcessingAllowed checks if data processing is allowed in the given region
// under the jurisdiction.
func IsProcessingAllowed(jurisdiction Jurisdiction, region Region) bool {
	rules := DefaultJurisdictionRules()
	rule, ok := rules[jurisdiction]
	if !ok {
		return false
	}
	for _, r := range rule.AllowedProcessingRegions {
		if r == region {
			return true
		}
	}
	return false
}

// JurisdictionMetadata represents jurisdiction information attached to domain entities.
type JurisdictionMetadata struct {
	// PrimaryJurisdiction is the governing jurisdiction for this entity.
	PrimaryJurisdiction Jurisdiction
	// DataClassification is the sensitivity level of the entity's data.
	DataClassification DataClassification
	// StorageRegion is the region where this entity's data is stored.
	StorageRegion Region
	// ConsentForCrossBorder indicates if the data subject consented to cross-border transfer.
	ConsentForCrossBorder bool
}

// Validate checks that the metadata is consistent with jurisdiction rules.
func (m JurisdictionMetadata) Validate() error {
	if m.PrimaryJurisdiction == "" {
		return fmt.Errorf("primary jurisdiction is required")
	}
	if m.DataClassification == "" {
		return fmt.Errorf("data classification is required")
	}
	if m.StorageRegion == "" {
		return fmt.Errorf("storage region is required")
	}
	if !IsRegionAllowed(m.PrimaryJurisdiction, m.StorageRegion) {
		return fmt.Errorf("region %s is not allowed for jurisdiction %s",
			m.StorageRegion, m.PrimaryJurisdiction)
	}
	return nil
}
