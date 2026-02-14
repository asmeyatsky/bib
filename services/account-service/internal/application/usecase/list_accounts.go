package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/port"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListAccountsUseCase handles listing customer accounts with pagination.
type ListAccountsUseCase struct {
	repo   port.AccountRepository
	logger *slog.Logger
}

// NewListAccountsUseCase creates a new ListAccountsUseCase.
func NewListAccountsUseCase(repo port.AccountRepository, logger *slog.Logger) *ListAccountsUseCase {
	return &ListAccountsUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute lists customer accounts with pagination.
// Filters by tenant ID or holder ID (at least one must be provided).
func (uc *ListAccountsUseCase) Execute(ctx context.Context, req dto.ListAccountsRequest) (dto.ListAccountsResponse, error) {
	uc.logger.Info("listing accounts",
		"tenant_id", req.TenantID,
		"holder_id", req.HolderID,
		"limit", req.Limit,
		"offset", req.Offset,
	)

	// Apply defaults.
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	var (
		accounts []model.CustomerAccount
		total    int
		err      error
	)

	switch {
	case req.HolderID != uuid.Nil:
		accounts, total, err = uc.repo.ListByHolder(ctx, req.HolderID, limit, offset)
	case req.TenantID != uuid.Nil:
		accounts, total, err = uc.repo.ListByTenant(ctx, req.TenantID, limit, offset)
	default:
		return dto.ListAccountsResponse{}, fmt.Errorf("tenant_id or holder_id is required")
	}

	if err != nil {
		return dto.ListAccountsResponse{}, fmt.Errorf("failed to list accounts: %w", err)
	}

	responses := make([]dto.AccountResponse, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, dto.AccountResponse{
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
		})
	}

	return dto.ListAccountsResponse{
		Accounts:   responses,
		TotalCount: total,
	}, nil
}
