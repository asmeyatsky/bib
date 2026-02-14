package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// --- Deposit Product DTOs ---

// InterestTierDTO transfers interest tier data between layers.
type InterestTierDTO struct {
	MinBalance decimal.Decimal
	MaxBalance decimal.Decimal
	RateBps    int
}

// CreateDepositProductRequest is the input DTO for creating a deposit product.
type CreateDepositProductRequest struct {
	TenantID uuid.UUID
	Name     string
	Currency string
	Tiers    []InterestTierDTO
	TermDays int
}

// DepositProductResponse is the output DTO for a deposit product.
type DepositProductResponse struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Currency  string
	Tiers     []InterestTierDTO
	TermDays  int
	IsActive  bool
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// --- Deposit Position DTOs ---

// OpenPositionRequest is the input DTO for opening a deposit position.
type OpenPositionRequest struct {
	TenantID  uuid.UUID
	AccountID uuid.UUID
	ProductID uuid.UUID
	Principal decimal.Decimal
}

// DepositPositionResponse is the output DTO for a deposit position.
type DepositPositionResponse struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	AccountID       uuid.UUID
	ProductID       uuid.UUID
	Principal       decimal.Decimal
	Currency        string
	AccruedInterest decimal.Decimal
	Status          string
	OpenedAt        time.Time
	MaturityDate    *time.Time
	LastAccrualDate time.Time
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// --- Accrual DTOs ---

// AccrueInterestRequest is the input DTO for batch interest accrual.
type AccrueInterestRequest struct {
	TenantID uuid.UUID
	AsOf     time.Time
}

// AccrueInterestResponse is the output DTO for batch interest accrual.
type AccrueInterestResponse struct {
	PositionsProcessed int
	TotalAccrued       decimal.Decimal
}

// --- Query DTOs ---

// GetPositionRequest is the input DTO for fetching a deposit position.
type GetPositionRequest struct {
	PositionID uuid.UUID
}
