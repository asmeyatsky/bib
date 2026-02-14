package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/domain/port"
)

// GetAccountUseCase handles retrieving a customer account by ID.
type GetAccountUseCase struct {
	repo   port.AccountRepository
	logger *slog.Logger
}

// NewGetAccountUseCase creates a new GetAccountUseCase.
func NewGetAccountUseCase(repo port.AccountRepository, logger *slog.Logger) *GetAccountUseCase {
	return &GetAccountUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a customer account by its ID.
func (uc *GetAccountUseCase) Execute(ctx context.Context, req dto.GetAccountRequest) (dto.AccountResponse, error) {
	uc.logger.Info("getting account", "account_id", req.AccountID)

	account, err := uc.repo.FindByID(ctx, req.AccountID)
	if err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to find account %s: %w", req.AccountID, err)
	}

	return dto.AccountResponse{
		AccountID:         account.ID(),
		TenantID:          account.TenantID(),
		AccountNumber:     account.AccountNumber().String(),
		AccountType:       account.AccountType().String(),
		Status:            string(account.Status()),
		Currency:          account.Currency(),
		LedgerAccountCode: account.LedgerAccountCode(),
		HolderID:          account.Holder().ID(),
		HolderFirstName:   account.Holder().FirstName(),
		HolderLastName:    account.Holder().LastName(),
		HolderEmail:       account.Holder().Email(),
		Version:           account.Version(),
		CreatedAt:         account.CreatedAt(),
		UpdatedAt:         account.UpdatedAt(),
	}, nil
}
