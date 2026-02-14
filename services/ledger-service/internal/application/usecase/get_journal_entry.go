package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
)

// GetJournalEntry retrieves a journal entry by ID.
type GetJournalEntry struct {
	journalRepo port.JournalRepository
}

func NewGetJournalEntry(journalRepo port.JournalRepository) *GetJournalEntry {
	return &GetJournalEntry{journalRepo: journalRepo}
}

func (uc *GetJournalEntry) Execute(ctx context.Context, id uuid.UUID) (dto.JournalEntryResponse, error) {
	entry, err := uc.journalRepo.FindByID(ctx, id)
	if err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to find entry: %w", err)
	}
	return toJournalEntryResponse(entry), nil
}
