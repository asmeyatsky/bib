package usecase

import (
	"context"
	"fmt"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// ListJournalEntries retrieves journal entries with filtering and pagination.
type ListJournalEntries struct {
	journalRepo port.JournalRepository
}

func NewListJournalEntries(journalRepo port.JournalRepository) *ListJournalEntries {
	return &ListJournalEntries{journalRepo: journalRepo}
}

func (uc *ListJournalEntries) Execute(ctx context.Context, req dto.ListEntriesRequest) (dto.ListEntriesResponse, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	if req.AccountCode != "" {
		accountCode, err := valueobject.NewAccountCode(req.AccountCode)
		if err != nil {
			return dto.ListEntriesResponse{}, fmt.Errorf("invalid account code: %w", err)
		}
		entries, total, err := uc.journalRepo.ListByAccount(ctx, req.TenantID, accountCode, req.FromDate, req.ToDate, pageSize, req.Offset)
		if err != nil {
			return dto.ListEntriesResponse{}, fmt.Errorf("failed to list entries: %w", err)
		}
		return toListResponse(entries, total), nil
	}

	entries, total, err := uc.journalRepo.ListByTenant(ctx, req.TenantID, req.FromDate, req.ToDate, pageSize, req.Offset)
	if err != nil {
		return dto.ListEntriesResponse{}, fmt.Errorf("failed to list entries: %w", err)
	}
	return toListResponse(entries, total), nil
}

func toListResponse(entries []model.JournalEntry, total int) dto.ListEntriesResponse {
	var responses []dto.JournalEntryResponse
	for _, e := range entries {
		responses = append(responses, toJournalEntryResponse(e))
	}
	return dto.ListEntriesResponse{
		Entries:    responses,
		TotalCount: total,
	}
}
