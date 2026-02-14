package service

import (
	"fmt"
	"time"

	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
)

// AccrualEngine is a domain service responsible for calculating interest accruals
// on deposit positions based on their associated product tier configuration.
type AccrualEngine struct{}

// NewAccrualEngine creates a new AccrualEngine.
func NewAccrualEngine() *AccrualEngine {
	return &AccrualEngine{}
}

// AccrueForPosition calculates interest accrual for a single position based on its product tiers.
// It finds the applicable tier for the position's total balance (principal + accrued),
// derives the daily rate from that tier, and delegates to the position's AccrueInterest method.
func (e *AccrualEngine) AccrueForPosition(
	position model.DepositPosition,
	product model.DepositProduct,
	asOf time.Time,
) (model.DepositPosition, error) {
	if position.Status() != model.PositionStatusActive {
		return model.DepositPosition{}, fmt.Errorf("position %s is not active", position.ID())
	}

	// Use total balance (principal + accrued interest) to determine the applicable tier
	totalBalance := position.TotalBalance()

	tier, err := product.FindApplicableTier(totalBalance)
	if err != nil {
		return model.DepositPosition{}, fmt.Errorf("find tier for position %s: %w", position.ID(), err)
	}

	dailyRate := tier.DailyRate()

	accrued, err := position.AccrueInterest(dailyRate, asOf)
	if err != nil {
		return model.DepositPosition{}, fmt.Errorf("accrue interest for position %s: %w", position.ID(), err)
	}

	return accrued, nil
}
