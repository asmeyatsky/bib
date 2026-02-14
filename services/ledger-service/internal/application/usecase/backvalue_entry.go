package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
)

// BackvalueEntry re-dates a pending journal entry.
type BackvalueEntry struct {
	journalRepo port.JournalRepository
}

func NewBackvalueEntry(journalRepo port.JournalRepository) *BackvalueEntry {
	return &BackvalueEntry{journalRepo: journalRepo}
}

func (uc *BackvalueEntry) Execute(ctx context.Context, req dto.BackvalueEntryRequest) (dto.JournalEntryResponse, error) {
	entry, err := uc.journalRepo.FindByID(ctx, req.EntryID)
	if err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to find entry: %w", err)
	}

	now := time.Now().UTC()
	backvalued, err := entry.Backvalue(req.NewDate, now)
	if err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to backvalue entry: %w", err)
	}

	if err := uc.journalRepo.Save(ctx, backvalued); err != nil {
		return dto.JournalEntryResponse{}, fmt.Errorf("failed to save backvalued entry: %w", err)
	}

	return toJournalEntryResponse(backvalued), nil
}
