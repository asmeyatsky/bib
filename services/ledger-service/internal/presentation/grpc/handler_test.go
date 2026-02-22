package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/model"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/service"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockJournalRepo struct {
	saveErr      error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.JournalEntry, error)
}

func (m *mockJournalRepo) Save(_ context.Context, _ model.JournalEntry) error {
	return m.saveErr
}

func (m *mockJournalRepo) FindByID(ctx context.Context, id uuid.UUID) (model.JournalEntry, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.JournalEntry{}, fmt.Errorf("not found")
}

func (m *mockJournalRepo) ListByAccount(_ context.Context, _ uuid.UUID, _ valueobject.AccountCode, _, _ time.Time, _, _ int) ([]model.JournalEntry, int, error) {
	return nil, 0, nil
}

func (m *mockJournalRepo) ListByTenant(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int) ([]model.JournalEntry, int, error) {
	return nil, 0, nil
}

type mockBalanceRepo struct {
	balance    decimal.Decimal
	balanceErr error
	updateErr  error
}

func (m *mockBalanceRepo) GetBalance(_ context.Context, _ valueobject.AccountCode, _ string, _ time.Time) (decimal.Decimal, error) {
	if m.balanceErr != nil {
		return decimal.Zero, m.balanceErr
	}
	return m.balance, nil
}

func (m *mockBalanceRepo) UpdateBalance(_ context.Context, _ valueobject.AccountCode, _ string, _ decimal.Decimal) error {
	return m.updateErr
}

type mockFiscalPeriodRepo struct{}

func (m *mockFiscalPeriodRepo) GetPeriodStatus(_ context.Context, _ uuid.UUID, _ valueobject.FiscalPeriod) (valueobject.PeriodStatus, error) {
	return valueobject.PeriodStatusOpen, nil
}

func (m *mockFiscalPeriodRepo) ClosePeriod(_ context.Context, _ uuid.UUID, _ valueobject.FiscalPeriod) error {
	return nil
}

type mockEventPublisher struct {
	publishErr error
}

func (m *mockEventPublisher) Publish(_ context.Context, _ string, _ ...events.DomainEvent) error {
	return m.publishErr
}

// --- Helpers ---

func contextWithClaims() context.Context {
	claims := &auth.Claims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Roles:    []string{auth.RoleAdmin},
	}
	return auth.ContextWithClaims(context.Background(), claims)
}

func buildTestHandler() *LedgerHandler {
	journalRepo := &mockJournalRepo{}
	balanceRepo := &mockBalanceRepo{balance: decimal.NewFromInt(1000)}
	periodRepo := &mockFiscalPeriodRepo{}
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()

	return NewLedgerHandler(
		usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator),
		usecase.NewGetJournalEntry(journalRepo),
		usecase.NewGetBalance(balanceRepo),
		usecase.NewListJournalEntries(journalRepo),
		usecase.NewBackvalueEntry(journalRepo),
		usecase.NewPeriodClose(periodRepo, publisher),
	)
}

func buildHandlerWithRepos(journalRepo port.JournalRepository, balanceRepo port.BalanceRepository) *LedgerHandler {
	publisher := &mockEventPublisher{}
	validator := service.NewPostingValidator()
	periodRepo := &mockFiscalPeriodRepo{}

	return NewLedgerHandler(
		usecase.NewPostJournalEntry(journalRepo, balanceRepo, publisher, validator),
		usecase.NewGetJournalEntry(journalRepo),
		usecase.NewGetBalance(balanceRepo),
		usecase.NewListJournalEntries(journalRepo),
		usecase.NewBackvalueEntry(journalRepo),
		usecase.NewPeriodClose(periodRepo, publisher),
	)
}

// --- Tests ---

func TestHandlePostJournalEntry(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandlePostJournalEntry(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no postings returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandlePostJournalEntry(contextWithClaims(), &PostJournalEntryRequest{
			TenantID:      uuid.New().String(),
			EffectiveDate: "2024-01-15",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "at least one posting is required")
	})

	t.Run("invalid effective_date returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandlePostJournalEntry(contextWithClaims(), &PostJournalEntryRequest{
			TenantID:      uuid.New().String(),
			EffectiveDate: "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid effective_date")
	})

	t.Run("invalid posting amount returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandlePostJournalEntry(contextWithClaims(), &PostJournalEntryRequest{
			TenantID:      uuid.New().String(),
			EffectiveDate: "2024-01-15",
			Postings: []*PostingPairMsg{
				{
					DebitAccount:  "1000",
					CreditAccount: "2000",
					Amount:        "not-a-number",
					Currency:      "USD",
				},
			},
			Description: "test",
			Reference:   "REF-001",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("happy path returns journal entry", func(t *testing.T) {
		h := buildTestHandler()
		resp, err := h.HandlePostJournalEntry(contextWithClaims(), &PostJournalEntryRequest{
			TenantID:      uuid.New().String(),
			EffectiveDate: "2024-01-15",
			Postings: []*PostingPairMsg{
				{
					DebitAccount:  "1000",
					CreditAccount: "2000",
					Amount:        "100.00",
					Currency:      "USD",
					Description:   "test posting",
				},
			},
			Description: "Test entry",
			Reference:   "REF-001",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Entry)
		assert.NotEmpty(t, resp.Entry.ID)
		assert.Equal(t, "POSTED", resp.Entry.Status)
		assert.Equal(t, "Test entry", resp.Entry.Description)
		assert.Equal(t, "REF-001", resp.Entry.Reference)
	})

	t.Run("use case failure returns Internal", func(t *testing.T) {
		journalRepo := &mockJournalRepo{saveErr: fmt.Errorf("db error")}
		balanceRepo := &mockBalanceRepo{balance: decimal.NewFromInt(1000)}
		h := buildHandlerWithRepos(journalRepo, balanceRepo)

		_, err := h.HandlePostJournalEntry(contextWithClaims(), &PostJournalEntryRequest{
			TenantID:      uuid.New().String(),
			EffectiveDate: "2024-01-15",
			Postings: []*PostingPairMsg{
				{
					DebitAccount:  "1000",
					CreditAccount: "2000",
					Amount:        "100.00",
					Currency:      "USD",
				},
			},
			Description: "Test entry",
			Reference:   "REF-001",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestHandleGetBalance(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleGetBalance(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid as_of date returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleGetBalance(contextWithClaims(), &GetBalanceRequest{
			AccountCode: "1000",
			Currency:    "USD",
			AsOf:        "not-a-date",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid as_of date")
	})

	t.Run("happy path returns balance", func(t *testing.T) {
		balanceRepo := &mockBalanceRepo{balance: decimal.NewFromInt(5000)}
		h := buildHandlerWithRepos(&mockJournalRepo{}, balanceRepo)

		resp, err := h.HandleGetBalance(contextWithClaims(), &GetBalanceRequest{
			AccountCode: "1000",
			Currency:    "USD",
		})
		require.NoError(t, err)
		assert.Equal(t, "1000", resp.AccountCode)
		assert.Equal(t, "5000", resp.Amount)
		assert.Equal(t, "USD", resp.Currency)
	})

	t.Run("happy path with as_of date", func(t *testing.T) {
		balanceRepo := &mockBalanceRepo{balance: decimal.NewFromInt(3000)}
		h := buildHandlerWithRepos(&mockJournalRepo{}, balanceRepo)

		resp, err := h.HandleGetBalance(contextWithClaims(), &GetBalanceRequest{
			AccountCode: "2000",
			Currency:    "EUR",
			AsOf:        "2024-06-30",
		})
		require.NoError(t, err)
		assert.Equal(t, "2000", resp.AccountCode)
		assert.Equal(t, "3000", resp.Amount)
		assert.Equal(t, "EUR", resp.Currency)
		assert.Equal(t, "2024-06-30", resp.AsOf)
	})

	t.Run("use case failure returns Internal", func(t *testing.T) {
		balanceRepo := &mockBalanceRepo{balanceErr: fmt.Errorf("db error")}
		h := buildHandlerWithRepos(&mockJournalRepo{}, balanceRepo)

		// GetBalance swallows the error and returns zero when no row found,
		// but we pass an invalid account code to trigger error path.
		_, err := h.HandleGetBalance(contextWithClaims(), &GetBalanceRequest{
			AccountCode: "invalid-code",
			Currency:    "USD",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestToJournalEntryMsg(t *testing.T) {
	now := time.Now().UTC()
	entryID := uuid.New()
	tenantID := uuid.New()

	msg := toJournalEntryMsg(dto.JournalEntryResponse{
		ID:            entryID,
		TenantID:      tenantID,
		EffectiveDate: now,
		Postings: []dto.PostingPairDTO{
			{
				DebitAccount:  "1000",
				CreditAccount: "2000",
				Amount:        decimal.NewFromInt(100),
				Currency:      "USD",
				Description:   "test",
			},
		},
		Status:      "POSTED",
		Description: "Test entry",
		Reference:   "REF-001",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	assert.Equal(t, entryID.String(), msg.ID)
	assert.Equal(t, tenantID.String(), msg.TenantID)
	assert.Equal(t, "POSTED", msg.Status)
	assert.Equal(t, "Test entry", msg.Description)
	assert.Equal(t, "REF-001", msg.Reference)
	assert.Equal(t, int32(1), msg.Version)
	require.Len(t, msg.Postings, 1)
	assert.Equal(t, "1000", msg.Postings[0].DebitAccount)
	assert.Equal(t, "2000", msg.Postings[0].CreditAccount)
	assert.Equal(t, "100", msg.Postings[0].Amount)
	assert.Equal(t, "USD", msg.Postings[0].Currency)
}

// requireGRPCCode asserts that an error is a gRPC status error with the given code.
func requireGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %T: %v", err, err)
	assert.Equal(t, code, st.Code(), "expected gRPC code %s, got %s: %s", code, st.Code(), st.Message())
}
