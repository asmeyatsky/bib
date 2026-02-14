package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// GetBalance retrieves the current or historical balance of an account.
type GetBalance struct {
	balanceRepo port.BalanceRepository
}

func NewGetBalance(balanceRepo port.BalanceRepository) *GetBalance {
	return &GetBalance{balanceRepo: balanceRepo}
}

func (uc *GetBalance) Execute(ctx context.Context, req dto.GetBalanceRequest) (dto.BalanceResponse, error) {
	accountCode, err := valueobject.NewAccountCode(req.AccountCode)
	if err != nil {
		return dto.BalanceResponse{}, fmt.Errorf("invalid account code: %w", err)
	}

	asOf := req.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	balance, err := uc.balanceRepo.GetBalance(ctx, accountCode, req.Currency, asOf)
	if err != nil {
		return dto.BalanceResponse{}, fmt.Errorf("failed to get balance: %w", err)
	}

	return dto.BalanceResponse{
		AccountCode: accountCode.Code(),
		Amount:      balance,
		Currency:    req.Currency,
		AsOf:        asOf,
	}, nil
}
