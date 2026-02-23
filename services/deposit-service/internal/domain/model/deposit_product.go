package model

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/valueobject"
)

// DepositProduct is the aggregate root for deposit product definitions.
// It contains tiered interest configuration and term/demand classification.
type DepositProduct struct {
	createdAt time.Time
	updatedAt time.Time
	name      string
	currency  string
	tiers     []valueobject.InterestTier
	termDays  int
	version   int
	id        uuid.UUID
	tenantID  uuid.UUID
	isActive  bool
}

// NewDepositProduct creates a new DepositProduct with validation.
func NewDepositProduct(
	tenantID uuid.UUID,
	name string,
	currency string,
	tiers []valueobject.InterestTier,
	termDays int,
) (DepositProduct, error) {
	if tenantID == uuid.Nil {
		return DepositProduct{}, fmt.Errorf("tenant ID is required")
	}
	if name == "" {
		return DepositProduct{}, fmt.Errorf("product name is required")
	}
	if currency == "" {
		return DepositProduct{}, fmt.Errorf("currency is required")
	}
	if len(currency) != 3 {
		return DepositProduct{}, fmt.Errorf("currency must be a 3-letter ISO code")
	}
	if len(tiers) == 0 {
		return DepositProduct{}, fmt.Errorf("at least one interest tier is required")
	}
	if termDays < 0 {
		return DepositProduct{}, fmt.Errorf("term days must not be negative")
	}
	if err := validateNoTierOverlap(tiers); err != nil {
		return DepositProduct{}, err
	}

	now := time.Now().UTC()
	return DepositProduct{
		id:        uuid.New(),
		tenantID:  tenantID,
		name:      name,
		currency:  currency,
		tiers:     copyTiers(tiers),
		termDays:  termDays,
		isActive:  true,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ReconstructProduct recreates a DepositProduct from persistence (no validation, no events).
func ReconstructProduct(
	id, tenantID uuid.UUID,
	name, currency string,
	tiers []valueobject.InterestTier,
	termDays int,
	isActive bool,
	version int,
	createdAt, updatedAt time.Time,
) DepositProduct {
	return DepositProduct{
		id:        id,
		tenantID:  tenantID,
		name:      name,
		currency:  currency,
		tiers:     copyTiers(tiers),
		termDays:  termDays,
		isActive:  isActive,
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// UpdateTiers replaces the interest tiers (immutable - returns new copy).
func (p DepositProduct) UpdateTiers(tiers []valueobject.InterestTier, now time.Time) (DepositProduct, error) {
	if !p.isActive {
		return DepositProduct{}, fmt.Errorf("cannot update tiers on an inactive product")
	}
	if len(tiers) == 0 {
		return DepositProduct{}, fmt.Errorf("at least one interest tier is required")
	}
	if err := validateNoTierOverlap(tiers); err != nil {
		return DepositProduct{}, err
	}

	updated := p
	updated.tiers = copyTiers(tiers)
	updated.updatedAt = now
	updated.version++
	return updated, nil
}

// Deactivate marks the product as inactive (immutable - returns new copy).
func (p DepositProduct) Deactivate(now time.Time) (DepositProduct, error) {
	if !p.isActive {
		return DepositProduct{}, fmt.Errorf("product is already inactive")
	}

	deactivated := p
	deactivated.isActive = false
	deactivated.updatedAt = now
	deactivated.version++
	return deactivated, nil
}

// FindApplicableTier returns the tier that applies to the given balance.
func (p DepositProduct) FindApplicableTier(balance decimal.Decimal) (valueobject.InterestTier, error) {
	for _, tier := range p.tiers {
		if tier.Applies(balance) {
			return tier, nil
		}
	}
	return valueobject.InterestTier{}, fmt.Errorf("no applicable tier for balance %s", balance)
}

// IsTermDeposit returns true if the product has a fixed term.
func (p DepositProduct) IsTermDeposit() bool {
	return p.termDays > 0
}

// Accessors
func (p DepositProduct) ID() uuid.UUID                     { return p.id }
func (p DepositProduct) TenantID() uuid.UUID               { return p.tenantID }
func (p DepositProduct) Name() string                      { return p.name }
func (p DepositProduct) Currency() string                  { return p.currency }
func (p DepositProduct) Tiers() []valueobject.InterestTier { return copyTiers(p.tiers) }
func (p DepositProduct) TermDays() int                     { return p.termDays }
func (p DepositProduct) IsActive() bool                    { return p.isActive }
func (p DepositProduct) Version() int                      { return p.version }
func (p DepositProduct) CreatedAt() time.Time              { return p.createdAt }
func (p DepositProduct) UpdatedAt() time.Time              { return p.updatedAt }

// validateNoTierOverlap ensures no two tiers have overlapping balance ranges.
func validateNoTierOverlap(tiers []valueobject.InterestTier) error {
	if len(tiers) <= 1 {
		return nil
	}

	// Sort by min balance for overlap detection
	sorted := make([]valueobject.InterestTier, len(tiers))
	copy(sorted, tiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MinBalance().LessThan(sorted[j].MinBalance())
	})

	for i := 1; i < len(sorted); i++ {
		prevMax := sorted[i-1].MaxBalance()
		currMin := sorted[i].MinBalance()
		if currMin.LessThanOrEqual(prevMax) {
			return fmt.Errorf("interest tiers overlap: tier ending at %s overlaps with tier starting at %s",
				prevMax, currMin)
		}
	}
	return nil
}

// copyTiers creates a defensive copy of a tier slice.
func copyTiers(tiers []valueobject.InterestTier) []valueobject.InterestTier {
	if tiers == nil {
		return nil
	}
	c := make([]valueobject.InterestTier, len(tiers))
	copy(c, tiers)
	return c
}
