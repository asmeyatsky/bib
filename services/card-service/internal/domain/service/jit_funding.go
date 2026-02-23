package service

import "github.com/shopspring/decimal"

// JITFundingService implements Just-In-Time funding verification.
// JIT funding checks whether the source account has sufficient funds
// at the exact moment a card transaction is being authorized.
type JITFundingService struct{}

// NewJITFundingService creates a new JIT funding service.
func NewJITFundingService() *JITFundingService {
	return &JITFundingService{}
}

// FundingResult contains the outcome of a JIT funding check.
type FundingResult struct {
	DeclineReason string `json:"decline_reason,omitempty"`
	Approved      bool   `json:"approved"`
}

// CheckFunding verifies the source account has sufficient funds for the transaction.
// The available balance must be greater than or equal to the transaction amount.
func (s *JITFundingService) CheckFunding(availableBalance, transactionAmount decimal.Decimal) FundingResult {
	if transactionAmount.IsNegative() || transactionAmount.IsZero() {
		return FundingResult{
			Approved:      false,
			DeclineReason: "transaction amount must be positive",
		}
	}

	if availableBalance.LessThan(transactionAmount) {
		return FundingResult{
			Approved:      false,
			DeclineReason: "insufficient funds",
		}
	}

	return FundingResult{
		Approved: true,
	}
}
