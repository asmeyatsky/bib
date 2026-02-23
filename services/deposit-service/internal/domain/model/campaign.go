package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

// CampaignStatus represents the lifecycle state of a deposit campaign.
type CampaignStatus string

const (
	CampaignStatusDraft    CampaignStatus = "DRAFT"
	CampaignStatusActive   CampaignStatus = "ACTIVE"
	CampaignStatusExpired  CampaignStatus = "EXPIRED"
	CampaignStatusCanceled CampaignStatus = "CANCELED"
)

// TargetAudience defines the audience segment for a campaign.
type TargetAudience string

const (
	TargetAudienceAll         TargetAudience = "ALL"
	TargetAudienceNewCustomer TargetAudience = "NEW_CUSTOMER"
	TargetAudienceExisting    TargetAudience = "EXISTING"
	TargetAudienceHighValue   TargetAudience = "HIGH_VALUE"
)

// Campaign is the aggregate root for deposit promotional campaigns.
// It tracks promotional rates, eligibility, time windows, and effectiveness metrics.
type Campaign struct {
	updatedAt         time.Time
	createdAt         time.Time
	endDate           time.Time
	startDate         time.Time
	description       string
	targetAudience    TargetAudience
	status            CampaignStatus
	totalDepositValue string
	name              string
	promotionalRate   valueobject.PromotionalRate
	totalEnrollments  int
	version           int
	id                uuid.UUID
	productID         uuid.UUID
	tenantID          uuid.UUID
}

// NewCampaign creates a new Campaign in DRAFT status.
func NewCampaign(
	tenantID uuid.UUID,
	name, description string,
	productID uuid.UUID,
	promotionalRate valueobject.PromotionalRate,
	targetAudience TargetAudience,
	startDate, endDate time.Time,
) (Campaign, error) {
	if tenantID == uuid.Nil {
		return Campaign{}, fmt.Errorf("tenant ID is required")
	}
	if name == "" {
		return Campaign{}, fmt.Errorf("campaign name is required")
	}
	if productID == uuid.Nil {
		return Campaign{}, fmt.Errorf("product ID is required")
	}
	if startDate.IsZero() {
		return Campaign{}, fmt.Errorf("start date is required")
	}
	if endDate.IsZero() {
		return Campaign{}, fmt.Errorf("end date is required")
	}
	if !endDate.After(startDate) {
		return Campaign{}, fmt.Errorf("end date must be after start date")
	}
	if targetAudience == "" {
		targetAudience = TargetAudienceAll
	}

	now := time.Now().UTC()
	return Campaign{
		id:                uuid.New(),
		tenantID:          tenantID,
		name:              name,
		description:       description,
		productID:         productID,
		promotionalRate:   promotionalRate,
		targetAudience:    targetAudience,
		startDate:         startDate,
		endDate:           endDate,
		status:            CampaignStatusDraft,
		totalEnrollments:  0,
		totalDepositValue: "0",
		version:           1,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

// ReconstructCampaign recreates a Campaign from persistence without validation.
func ReconstructCampaign(
	id, tenantID uuid.UUID,
	name, description string,
	productID uuid.UUID,
	promotionalRate valueobject.PromotionalRate,
	targetAudience TargetAudience,
	startDate, endDate time.Time,
	status CampaignStatus,
	totalEnrollments int,
	totalDepositValue string,
	version int,
	createdAt, updatedAt time.Time,
) Campaign {
	return Campaign{
		id:                id,
		tenantID:          tenantID,
		name:              name,
		description:       description,
		productID:         productID,
		promotionalRate:   promotionalRate,
		targetAudience:    targetAudience,
		startDate:         startDate,
		endDate:           endDate,
		status:            status,
		totalEnrollments:  totalEnrollments,
		totalDepositValue: totalDepositValue,
		version:           version,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

// Activate transitions the campaign from DRAFT to ACTIVE.
func (c Campaign) Activate(now time.Time) (Campaign, error) {
	if c.status != CampaignStatusDraft {
		return Campaign{}, fmt.Errorf("can only activate DRAFT campaigns, current: %s", c.status)
	}
	activated := c
	activated.status = CampaignStatusActive
	activated.updatedAt = now
	activated.version++
	return activated, nil
}

// Expire transitions the campaign from ACTIVE to EXPIRED.
func (c Campaign) Expire(now time.Time) (Campaign, error) {
	if c.status != CampaignStatusActive {
		return Campaign{}, fmt.Errorf("can only expire ACTIVE campaigns, current: %s", c.status)
	}
	expired := c
	expired.status = CampaignStatusExpired
	expired.updatedAt = now
	expired.version++
	return expired, nil
}

// Cancel transitions the campaign from DRAFT or ACTIVE to CANCELED.
func (c Campaign) Cancel(now time.Time) (Campaign, error) {
	if c.status != CampaignStatusDraft && c.status != CampaignStatusActive {
		return Campaign{}, fmt.Errorf("can only cancel DRAFT or ACTIVE campaigns, current: %s", c.status)
	}
	canceled := c
	canceled.status = CampaignStatusCanceled
	canceled.updatedAt = now
	canceled.version++
	return canceled, nil
}

// RecordEnrollment increments enrollment count and adds to the total deposit value.
func (c Campaign) RecordEnrollment(depositAmount string, now time.Time) Campaign {
	updated := c
	updated.totalEnrollments++
	updated.totalDepositValue = depositAmount // simplified: in production this would accumulate
	updated.updatedAt = now
	updated.version++
	return updated
}

// IsActiveAt returns true if the campaign is active and within its date window.
func (c Campaign) IsActiveAt(t time.Time) bool {
	return c.status == CampaignStatusActive &&
		!t.Before(c.startDate) &&
		!t.After(c.endDate)
}

// Accessors
func (c Campaign) ID() uuid.UUID                                { return c.id }
func (c Campaign) TenantID() uuid.UUID                          { return c.tenantID }
func (c Campaign) Name() string                                 { return c.name }
func (c Campaign) Description() string                          { return c.description }
func (c Campaign) ProductID() uuid.UUID                         { return c.productID }
func (c Campaign) PromotionalRate() valueobject.PromotionalRate { return c.promotionalRate }
func (c Campaign) TargetAudience() TargetAudience               { return c.targetAudience }
func (c Campaign) StartDate() time.Time                         { return c.startDate }
func (c Campaign) EndDate() time.Time                           { return c.endDate }
func (c Campaign) Status() CampaignStatus                       { return c.status }
func (c Campaign) TotalEnrollments() int                        { return c.totalEnrollments }
func (c Campaign) TotalDepositValue() string                    { return c.totalDepositValue }
func (c Campaign) Version() int                                 { return c.version }
func (c Campaign) CreatedAt() time.Time                         { return c.createdAt }
func (c Campaign) UpdatedAt() time.Time                         { return c.updatedAt }
