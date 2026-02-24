package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/account-service/internal/application/dto"
	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/port"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

const (
	accountEventsTopic = "account-events"
)

// ledger code prefixes by account type
var ledgerCodePrefixes = map[string]string{
	"CHECKING": "2000",
	"SAVINGS":  "2100",
	"LOAN":     "2200",
	"NOMINAL":  "2900",
}

// OpenAccountUseCase handles the creation of new customer accounts.
type OpenAccountUseCase struct {
	repo         port.AccountRepository
	publisher    port.EventPublisher
	ledgerClient port.LedgerClient
	logger       *slog.Logger
}

// NewOpenAccountUseCase creates a new OpenAccountUseCase.
func NewOpenAccountUseCase(
	repo port.AccountRepository,
	publisher port.EventPublisher,
	ledgerClient port.LedgerClient,
	logger *slog.Logger,
) *OpenAccountUseCase {
	return &OpenAccountUseCase{
		repo:         repo,
		publisher:    publisher,
		ledgerClient: ledgerClient,
		logger:       logger,
	}
}

// Execute creates a new customer account, generates a ledger account code,
// creates the matching ledger account, persists the account, and publishes events.
func (uc *OpenAccountUseCase) Execute(ctx context.Context, req dto.OpenAccountRequest) (dto.OpenAccountResponse, error) {
	uc.logger.Info("opening new account",
		"tenant_id", req.TenantID,
		"account_type", req.AccountType,
		"currency", req.Currency,
	)

	// Validate and create account type value object.
	accountType, err := valueobject.NewAccountType(req.AccountType)
	if err != nil {
		return dto.OpenAccountResponse{}, fmt.Errorf("invalid account type: %w", err)
	}

	// Create account holder entity.
	holder, err := model.NewAccountHolder(
		uuid.Nil, // Will generate a new ID
		req.HolderFirstName,
		req.HolderLastName,
		req.HolderEmail,
		req.IdentityVerificationID,
	)
	if err != nil {
		return dto.OpenAccountResponse{}, fmt.Errorf("invalid holder data: %w", err)
	}

	// Create the account aggregate (emits AccountOpened event).
	account, err := model.NewCustomerAccount(
		req.TenantID,
		accountType,
		req.Currency,
		holder,
	)
	if err != nil {
		return dto.OpenAccountResponse{}, fmt.Errorf("failed to create account: %w", err)
	}

	// Generate ledger account code.
	ledgerCode := generateLedgerCode(req.AccountType)

	// Assign ledger code to account.
	account, err = account.AssignLedgerCode(ledgerCode, time.Now())
	if err != nil {
		return dto.OpenAccountResponse{}, fmt.Errorf("failed to assign ledger code: %w", err)
	}

	// Activate the account now that setup is complete.
	account, err = account.Activate(time.Now())
	if err != nil {
		return dto.OpenAccountResponse{}, fmt.Errorf("failed to activate account: %w", err)
	}

	// Create matching ledger account in ledger service.
	if uc.ledgerClient != nil {
		if err := uc.ledgerClient.CreateLedgerAccount(ctx, req.TenantID, ledgerCode, req.Currency); err != nil {
			uc.logger.Error("failed to create ledger account", "error", err, "ledger_code", ledgerCode)
			return dto.OpenAccountResponse{}, fmt.Errorf("failed to create ledger account: %w", err)
		}
	}

	// Persist the account.
	if err := uc.repo.Save(ctx, account); err != nil {
		return dto.OpenAccountResponse{}, fmt.Errorf("failed to save account: %w", err)
	}

	// Publish domain events.
	events := account.DomainEvents()
	if len(events) > 0 {
		if err := uc.publisher.Publish(ctx, accountEventsTopic, events...); err != nil {
			uc.logger.Error("failed to publish domain events",
				"error", err,
				"account_id", account.ID(),
				"event_count", len(events),
			)
			// Do not fail the operation if event publishing fails;
			// the outbox pattern will handle retries.
		}
	}

	uc.logger.Info("account opened successfully",
		"account_id", account.ID(),
		"account_number", account.AccountNumber().String(),
		"ledger_code", ledgerCode,
	)

	return dto.OpenAccountResponse{
		AccountID:         account.ID(),
		AccountNumber:     account.AccountNumber().String(),
		Status:            string(account.Status()),
		LedgerAccountCode: ledgerCode,
		CreatedAt:         account.CreatedAt(),
	}, nil
}

// generateLedgerCode generates a ledger account code based on the account type.
// Format: "2000-NNN" for CHECKING, "2100-NNN" for SAVINGS, etc.
func generateLedgerCode(accountType string) string {
	prefix, ok := ledgerCodePrefixes[accountType]
	if !ok {
		prefix = "2900"
	}
	suffix := rand.Intn(900) + 100 //nolint:gosec // account suffix doesn't need crypto random
	return fmt.Sprintf("%s-%03d", prefix, suffix)
}
