package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/service"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

const TopicLedgerEntries = "bib.ledger.entries"

// PostJournalEntry handles the creation and posting of journal entries.
type PostJournalEntry struct {
	journalRepo port.JournalRepository
	balanceRepo port.BalanceRepository
	publisher   port.EventPublisher
	validator   *service.PostingValidator
}

func NewPostJournalEntry(
	journalRepo port.JournalRepository,
	balanceRepo port.BalanceRepository,
	publisher port.EventPublisher,
	validator *service.PostingValidator,
) *PostJournalEntry {
	return &PostJournalEntry{
		journalRepo: journalRepo,
		balanceRepo: balanceRepo,
		publisher:   publisher,
		validator:   validator,
	}
}

func (uc *PostJournalEntry) Execute(ctx context.Context, req dto.PostJournalEntryRequest) (dto.JournalEntryResponse, error) {
	// Convert DTOs to value objects
	var postings []valueobject.PostingPair
	for _, p := range req.Postings {
		debit, err := valueobject.NewAccountCode(p.DebitAccount)
		if err != nil {
			return dto.JournalEntryResponse{}, fmt.Errorf("invalid debit account: %w", err)
		}
		credit, err := valueobject.NewAccountCode(p.CreditAccount)
		if err != nil {
			return dto.JournalEntryResponse{}, fmt.Errorf("invalid credit account: %w", err)
		}
		pair, err := valueobject.NewPostingPair(debit, credit, p.Amount, p.Currency, p.Description)
		if err != nil {
			return dto.JournalEntryResponse{}, fmt.Errorf("invalid posting pair: %w", err)
		}
		postings = append(postings, pair)
	}

	// Validate postings
	if err := uc.validator.ValidatePostings(postings); err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("posting validation failed: %w", err)
	}

	// Create journal entry
	entry, err := model.NewJournalEntry(req.TenantID, req.EffectiveDate, postings, req.Description, req.Reference)
	if err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to create journal entry: %w", err)
	}

	// Post the entry
	now := time.Now().UTC()
	posted, err := entry.Post(now)
	if err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to post entry: %w", err)
	}

	// Persist
	if err := uc.journalRepo.Save(ctx, posted); err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to save entry: %w", err)
	}

	// Update balances for each posting
	for _, p := range posted.Postings() {
		// Debit increases debit-normal accounts
		if err := uc.balanceRepo.UpdateBalance(ctx, p.DebitAccount(), p.Currency(), p.Amount()); err != nil {
			return dto.JournalEntryResponse{}, fmt.Errorf("failed to update debit balance: %w", err)
		}
		// Credit decreases (negative delta) debit-normal accounts
		if err := uc.balanceRepo.UpdateBalance(ctx, p.CreditAccount(), p.Currency(), p.Amount().Neg()); err != nil {
			return dto.JournalEntryResponse{}, fmt.Errorf("failed to update credit balance: %w", err)
		}
	}

	// Publish domain events
	if events := posted.DomainEvents(); len(events) > 0 {
		if err := uc.publisher.Publish(ctx, TopicLedgerEntries, events...); err != nil {
			return dto.JournalEntryResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	return toJournalEntryResponse(posted), nil
}

func toJournalEntryResponse(entry model.JournalEntry) dto.JournalEntryResponse {
	var postings []dto.PostingPairDTO
	for _, p := range entry.Postings() {
		postings = append(postings, dto.PostingPairDTO{
			DebitAccount:  p.DebitAccount().Code(),
			CreditAccount: p.CreditAccount().Code(),
			Amount:        p.Amount(),
			Currency:      p.Currency(),
			Description:   p.Description(),
		})
	}
	return dto.JournalEntryResponse{
		ID:            entry.ID(),
		TenantID:      entry.TenantID(),
		EffectiveDate: entry.EffectiveDate(),
		Postings:      postings,
		Status:        string(entry.Status()),
		Description:   entry.Description(),
		Reference:     entry.Reference(),
		Version:       entry.Version(),
		CreatedAt:     entry.CreatedAt(),
		UpdatedAt:     entry.UpdatedAt(),
	}
}
