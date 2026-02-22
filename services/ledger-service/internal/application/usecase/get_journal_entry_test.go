package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func sampleJournalEntry() model.JournalEntry {
	tenantID := uuid.New()
	id := uuid.New()
	now := time.Now().UTC()

	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	posting, _ := valueobject.NewPostingPair(debit, credit, decimal.NewFromInt(500), "USD", "test posting")

	return model.Reconstruct(
		id, tenantID, now,
		[]valueobject.PostingPair{posting},
		model.EntryStatusPosted, "Test entry", "REF-001",
		1, now, now,
	)
}

func TestGetJournalEntry_Execute(t *testing.T) {
	t.Run("successfully retrieves a journal entry", func(t *testing.T) {
		entry := sampleJournalEntry()
		repo := &mockJournalRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.JournalEntry, error) {
				return entry, nil
			},
		}

		uc := usecase.NewGetJournalEntry(repo)

		resp, err := uc.Execute(context.Background(), entry.ID())

		require.NoError(t, err)
		assert.Equal(t, entry.ID(), resp.ID)
		assert.Equal(t, entry.TenantID(), resp.TenantID)
		assert.Equal(t, "POSTED", resp.Status)
		assert.Equal(t, "Test entry", resp.Description)
		assert.Equal(t, "REF-001", resp.Reference)
	})

	t.Run("fails when entry not found", func(t *testing.T) {
		repo := &mockJournalRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.JournalEntry, error) {
				return model.JournalEntry{}, fmt.Errorf("entry not found")
			},
		}

		uc := usecase.NewGetJournalEntry(repo)

		_, err := uc.Execute(context.Background(), uuid.New())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find entry")
	})
}
