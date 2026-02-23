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
	Name     string
	Currency string
	Tiers    []InterestTierDTO
	TermDays int
	TenantID uuid.UUID
}

// DepositProductResponse is the output DTO for a deposit product.
type DepositProductResponse struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	Currency  string
	Tiers     []InterestTierDTO
	TermDays  int
	Version   int
	ID        uuid.UUID
	TenantID  uuid.UUID
	IsActive  bool
}

// --- Deposit Position DTOs ---

// OpenPositionRequest is the input DTO for opening a deposit position.
type OpenPositionRequest struct {
	Principal decimal.Decimal
	TenantID  uuid.UUID
	AccountID uuid.UUID
	ProductID uuid.UUID
}

// DepositPositionResponse is the output DTO for a deposit position.
type DepositPositionResponse struct {
	OpenedAt        time.Time
	UpdatedAt       time.Time
	CreatedAt       time.Time
	LastAccrualDate time.Time
	MaturityDate    *time.Time
	AccruedInterest decimal.Decimal
	Status          string
	Currency        string
	Principal       decimal.Decimal
	Version         int
	ID              uuid.UUID
	ProductID       uuid.UUID
	AccountID       uuid.UUID
	TenantID        uuid.UUID
}

// --- Accrual DTOs ---

// AccrueInterestRequest is the input DTO for batch interest accrual.
type AccrueInterestRequest struct {
	AsOf     time.Time
	TenantID uuid.UUID
}

// AccrueInterestResponse is the output DTO for batch interest accrual.
type AccrueInterestResponse struct {
	TotalAccrued       decimal.Decimal
	PositionsProcessed int
}

// --- Query DTOs ---

// GetPositionRequest is the input DTO for fetching a deposit position.
type GetPositionRequest struct {
	PositionID uuid.UUID
}
