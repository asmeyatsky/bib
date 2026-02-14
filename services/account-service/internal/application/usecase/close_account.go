package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/domain/port"
)

// CloseAccountUseCase handles closing a customer account.
type CloseAccountUseCase struct {
	repo      port.AccountRepository
	publisher port.EventPublisher
	logger    *slog.Logger
}

// NewCloseAccountUseCase creates a new CloseAccountUseCase.
func NewCloseAccountUseCase(
	repo port.AccountRepository,
	publisher port.EventPublisher,
	logger *slog.Logger,
) *CloseAccountUseCase {
	return &CloseAccountUseCase{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// Execute closes a customer account.
func (uc *CloseAccountUseCase) Execute(ctx context.Context, req dto.CloseAccountRequest) (dto.AccountResponse, error) {
	uc.logger.Info("closing account", "account_id", req.AccountID, "reason", req.Reason)

	// Fetch the account.
	account, err := uc.repo.FindByID(ctx, req.AccountID)
	if err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to find account %s: %w", req.AccountID, err)
	}

	// Close the account (state transition).
	now := time.Now()
	closed, err := account.Close(req.Reason, now)
	if err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to close account: %w", err)
	}

	// Persist.
	if err := uc.repo.Save(ctx, closed); err != nil {
		return dto.AccountResponse{}, fmt.Errorf("failed to save closed account: %w", err)
	}

	// Publish domain events.
	events := closed.DomainEvents()
	if len(events) > 0 {
		if err := uc.publisher.Publish(ctx, accountEventsTopic, events...); err != nil {
			uc.logger.Error("failed to publish domain events",
				"error", err,
				"account_id", closed.ID(),
				"event_count", len(events),
			)
		}
	}

	uc.logger.Info("account closed successfully", "account_id", closed.ID())

	return dto.AccountResponse{
		AccountID:         closed.ID(),
		TenantID:          closed.TenantID(),
		AccountNumber:     closed.AccountNumber().String(),
		AccountType:       closed.AccountType().String(),
		Status:            string(closed.Status()),
		Currency:          closed.Currency(),
		LedgerAccountCode: closed.LedgerAccountCode(),
		HolderID:          closed.Holder().ID(),
		HolderFirstName:   closed.Holder().FirstName(),
		HolderLastName:    closed.Holder().LastName(),
		HolderEmail:       closed.Holder().Email(),
		Version:           closed.Version(),
		CreatedAt:         closed.CreatedAt(),
		UpdatedAt:         closed.UpdatedAt(),
	}, nil
}
