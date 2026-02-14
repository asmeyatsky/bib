package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/domain/port"
)

// FreezeAccountUseCase handles freezing a customer account.
type FreezeAccountUseCase struct {
	repo      port.AccountRepository
	publisher port.EventPublisher
	logger    *slog.Logger
}

// NewFreezeAccountUseCase creates a new FreezeAccountUseCase.
func NewFreezeAccountUseCase(
	repo port.AccountRepository,
	publisher port.EventPublisher,
	logger *slog.Logger,
) *FreezeAccountUseCase {
	return &FreezeAccountUseCase{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// Execute freezes a customer account.
func (uc *FreezeAccountUseCase) Execute(ctx context.Context, req dto.FreezeAccountRequest) (dto.AccountResponse, error) {
	uc.logger.Info("freezing account", "account_id", req.AccountID, "reason", req.Reason)

	// Fetch the account.
	account, err := uc.repo.FindByID(ctx, req.AccountID)
	if err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to find account %s: %w", req.AccountID, err)
	}

	// Freeze the account (state transition).
	now := time.Now()
	frozen, err := account.Freeze(req.Reason, now)
	if err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to freeze account: %w", err)
	}

	// Persist.
	if err := uc.repo.Save(ctx, frozen); err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to save frozen account: %w", err)
	}

	// Publish domain events.
	events := frozen.DomainEvents()
	if len(events) > 0 {
		if err := uc.publisher.Publish(ctx, accountEventsTopic, events...); err != nil {
			uc.logger.Error("failed to publish domain events",
				"error", err,
				"account_id", frozen.ID(),
				"event_count", len(events),
			)
		}
	}

	uc.logger.Info("account frozen successfully", "account_id", frozen.ID())

	return dto.AccountResponse{
		AccountID:         frozen.ID(),
		TenantID:          frozen.TenantID(),
		AccountNumber:     frozen.AccountNumber().String(),
		AccountType:       frozen.AccountType().String(),
		Status:            string(frozen.Status()),
		Currency:          frozen.Currency(),
		LedgerAccountCode: frozen.LedgerAccountCode(),
		HolderID:          frozen.Holder().ID(),
		HolderFirstName:   frozen.Holder().FirstName(),
		HolderLastName:    frozen.Holder().LastName(),
		HolderEmail:       frozen.Holder().Email(),
		Version:           frozen.Version(),
		CreatedAt:         frozen.CreatedAt(),
		UpdatedAt:         frozen.UpdatedAt(),
	}, nil
}
