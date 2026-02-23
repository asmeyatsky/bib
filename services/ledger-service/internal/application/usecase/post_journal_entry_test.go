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

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/service"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// --- Mock implementations ---

// mockJournalRepository implements port.JournalRepository for testing.
type mockJournalRepository struct {
	savedEntries []model.JournalEntry
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.JournalEntry, error)
	saveFunc     func(ctx context.Context, entry model.JournalEntry) error
}

func (m *mockJournalRepository) Save(ctx context.Context, entry model.JournalEntry) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, entry)
	}
	m.savedEntries = append(m.savedEntries, entry)
	return nil
}

func (m *mockJournalRepository) FindByID(ctx context.Context, id uuid.UUID) (model.JournalEntry, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.JournalEntry{}, fmt.Errorf("entry not found: %s", id)
}

func (m *mockJournalRepository) ListByAccount(_ context.Context, _ uuid.UUID, _ valueobject.AccountCode, _, _ time.Time, _, _ int) ([]model.JournalEntry, int, error) {
	return nil, 0, nil
}

func (m *mockJournalRepository) ListByTenant(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int) ([]model.JournalEntry, int, error) {
	return nil, 0, nil
}

// mockBalanceRepository implements port.BalanceRepository for testing.
type mockBalanceRepository struct {
	updates        []balanceUpdate
	updateFunc     func(ctx context.Context, account valueobject.AccountCode, currency string, delta decimal.Decimal) error
	getBalanceFunc func(ctx context.Context, account valueobject.AccountCode, currency string, asOf time.Time) (decimal.Decimal, error)
}

type balanceUpdate struct {
	Account  valueobject.AccountCode
	Currency string
	Delta    decimal.Decimal
}

func (m *mockBalanceRepository) UpdateBalance(ctx context.Context, account valueobject.AccountCode, currency string, delta decimal.Decimal) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, account, currency, delta)
	}
	m.updates = append(m.updates, balanceUpdate{Account: account, Currency: currency, Delta: delta})
	return nil
}

func (m *mockBalanceRepository) GetBalance(ctx context.Context, account valueobject.AccountCode, currency string, asOf time.Time) (decimal.Decimal, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, account, currency, asOf)
	}
	return decimal.Zero, nil
}

// mockEventPublisher implements port.EventPublisher for testing.
type mockEventPublisher struct {
	publishedEvents []events.DomainEvent
	publishFunc     func(ctx context.Context, topic string, events ...events.DomainEvent) error
}

func (m *mockEventPublisher) Publish(ctx context.Context, topic string, evts ...events.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

// --- Tests ---

func validPostRequest() dto.PostJournalEntryRequest {
	return dto.PostJournalEntryRequest{
		TenantID:      uuid.New(),
		EffectiveDate: time.Now().UTC(),
		Postings: []dto.PostingPairDTO{
			{
				DebitAccount:  "1000",
				CreditAccount: "2000",
				Amount:        decimal.NewFromInt(500),
				Currency:      "USD",
				Description:   "Test posting",
			},
		},
		Description: "Test journal entry",
		Reference:   "REF-001",
	}
}

func TestPostJournalEntry_Success(t *testing.T) {
	journalRepo := &mockJournalRepository{}
	balanceRepo := &mockBalanceRepository{}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := validPostRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, req.TenantID, resp.TenantID)
	assert.Equal(t, "POSTED", resp.Status)
	assert.Equal(t, req.Description, resp.Description)
	assert.Equal(t, req.Reference, resp.Reference)
	assert.Equal(t, 2, resp.Version) // version incremented on post
	assert.Len(t, resp.Postings, 1)
	assert.Equal(t, "1000", resp.Postings[0].DebitAccount)
	assert.Equal(t, "2000", resp.Postings[0].CreditAccount)
	assert.True(t, decimal.NewFromInt(500).Equal(resp.Postings[0].Amount))

	// Verify journal was saved
	require.Len(t, journalRepo.savedEntries, 1)

	// Verify balance updates (one debit, one credit)
	require.Len(t, balanceRepo.updates, 2)
	assert.Equal(t, "1000", balanceRepo.updates[0].Account.Code())
	assert.True(t, decimal.NewFromInt(500).Equal(balanceRepo.updates[0].Delta))
	assert.Equal(t, "2000", balanceRepo.updates[1].Account.Code())
	assert.True(t, decimal.NewFromInt(-500).Equal(balanceRepo.updates[1].Delta))

	// Verify events were published
	assert.NotEmpty(t, publisher.publishedEvents)
}

func TestPostJournalEntry_InvalidDebitAccount(t *testing.T) {
	journalRepo := &mockJournalRepository{}
	balanceRepo := &mockBalanceRepository{}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := validPostRequest()
	req.Postings[0].DebitAccount = "INVALID"

	resp, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid debit account")
	assert.Equal(t, uuid.Nil, resp.ID)
	assert.Empty(t, journalRepo.savedEntries)
}

func TestPostJournalEntry_InvalidCreditAccount(t *testing.T) {
	journalRepo := &mockJournalRepository{}
	balanceRepo := &mockBalanceRepository{}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := validPostRequest()
	req.Postings[0].CreditAccount = "BAD"

	resp, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credit account")
	assert.Equal(t, uuid.Nil, resp.ID)
}

func TestPostJournalEntry_RepoSaveError(t *testing.T) {
	journalRepo := &mockJournalRepository{
		saveFunc: func(_ context.Context, _ model.JournalEntry) error {
			return fmt.Errorf("database connection lost")
		},
	}
	balanceRepo := &mockBalanceRepository{}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := validPostRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save entry")
	assert.Contains(t, err.Error(), "database connection lost")
	assert.Equal(t, uuid.Nil, resp.ID)

	// Verify no balance updates occurred after save failure
	assert.Empty(t, balanceRepo.updates)
	assert.Empty(t, publisher.publishedEvents)
}

func TestPostJournalEntry_BalanceUpdateError(t *testing.T) {
	journalRepo := &mockJournalRepository{}
	balanceRepo := &mockBalanceRepository{
		updateFunc: func(_ context.Context, _ valueobject.AccountCode, _ string, _ decimal.Decimal) error {
			return fmt.Errorf("balance store unavailable")
		},
	}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := validPostRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update debit balance")
	assert.Equal(t, uuid.Nil, resp.ID)
}

func TestPostJournalEntry_PublishError(t *testing.T) {
	journalRepo := &mockJournalRepository{}
	balanceRepo := &mockBalanceRepository{}
	publisher := &mockEventPublisher{
		publishFunc: func(_ context.Context, _ string, _ ...events.DomainEvent) error {
			return fmt.Errorf("broker unreachable")
		},
	}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := validPostRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish events")
	assert.Equal(t, uuid.Nil, resp.ID)
}

func TestPostJournalEntry_MultiplePostings(t *testing.T) {
	journalRepo := &mockJournalRepository{}
	balanceRepo := &mockBalanceRepository{}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	uc := usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator)

	req := dto.PostJournalEntryRequest{
		TenantID:      uuid.New(),
		EffectiveDate: time.Now().UTC(),
		Postings: []dto.PostingPairDTO{
			{
				DebitAccount:  "1000",
				CreditAccount: "2000",
				Amount:        decimal.NewFromInt(100),
				Currency:      "USD",
				Description:   "First posting",
			},
			{
				DebitAccount:  "3000",
				CreditAccount: "4000",
				Amount:        decimal.NewFromInt(200),
				Currency:      "USD",
				Description:   "Second posting",
			},
		},
		Description: "Multi-posting entry",
		Reference:   "REF-002",
	}

	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Postings, 2)
	assert.Equal(t, "POSTED", resp.Status)

	// 2 postings x 2 balance updates each = 4 total
	require.Len(t, balanceRepo.updates, 4)
}
