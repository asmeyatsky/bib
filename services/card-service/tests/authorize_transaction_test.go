package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/card-service/internal/application/dto"
	"github.com/bibbank/bib/services/card-service/internal/application/usecase"
	"github.com/bibbank/bib/services/card-service/internal/domain/event"
	"github.com/bibbank/bib/services/card-service/internal/domain/model"
	"github.com/bibbank/bib/services/card-service/internal/domain/service"
	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

// --- Mock implementations for ports ---

// mockCardRepository is an in-memory card repository for testing.
type mockCardRepository struct {
	cards        map[uuid.UUID]model.Card
	transactions []mockTransaction
}

type mockTransaction struct {
	Amount           decimal.Decimal
	Currency         string
	MerchantName     string
	MerchantCategory string
	AuthCode         string
	Status           string
	CardID           uuid.UUID
}

func newMockCardRepository() *mockCardRepository {
	return &mockCardRepository{
		cards: make(map[uuid.UUID]model.Card),
	}
}

func (r *mockCardRepository) Save(_ context.Context, card model.Card) error {
	r.cards[card.ID()] = card
	return nil
}

func (r *mockCardRepository) Update(_ context.Context, card model.Card) error {
	if _, exists := r.cards[card.ID()]; !exists {
		return fmt.Errorf("card not found: %s", card.ID())
	}
	r.cards[card.ID()] = card
	return nil
}

func (r *mockCardRepository) FindByID(_ context.Context, id uuid.UUID) (model.Card, error) {
	card, exists := r.cards[id]
	if !exists {
		return model.Card{}, fmt.Errorf("card not found: %s", id)
	}
	return card, nil
}

func (r *mockCardRepository) FindByAccountID(_ context.Context, accountID uuid.UUID) ([]model.Card, error) {
	var result []model.Card
	for _, card := range r.cards {
		if card.AccountID() == accountID {
			result = append(result, card)
		}
	}
	return result, nil
}

func (r *mockCardRepository) FindByTenantID(_ context.Context, tenantID uuid.UUID) ([]model.Card, error) {
	var result []model.Card
	for _, card := range r.cards {
		if card.TenantID() == tenantID {
			result = append(result, card)
		}
	}
	return result, nil
}

func (r *mockCardRepository) SaveTransaction(_ context.Context, cardID uuid.UUID, amount decimal.Decimal, currency, merchantName, merchantCategory, authCode, status string) error {
	r.transactions = append(r.transactions, mockTransaction{
		CardID:           cardID,
		Amount:           amount,
		Currency:         currency,
		MerchantName:     merchantName,
		MerchantCategory: merchantCategory,
		AuthCode:         authCode,
		Status:           status,
	})
	return nil
}

// mockEventPublisher captures published events for assertion.
type mockEventPublisher struct {
	publishedEvents []event.DomainEvent
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (p *mockEventPublisher) Publish(_ context.Context, events []event.DomainEvent) error {
	p.publishedEvents = append(p.publishedEvents, events...)
	return nil
}

// mockBalanceClient returns a configurable balance.
type mockBalanceClient struct {
	err     error
	balance decimal.Decimal
}

func newMockBalanceClient(balance decimal.Decimal) *mockBalanceClient {
	return &mockBalanceClient{balance: balance}
}

func (c *mockBalanceClient) GetAvailableBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return c.balance, c.err
}

// --- Tests ---

func TestAuthorizeTransactionUseCase_Success(t *testing.T) {
	ctx := context.Background()
	repo := newMockCardRepository()
	publisher := newMockEventPublisher()
	balanceClient := newMockBalanceClient(decimal.NewFromInt(10000))
	jitFunding := service.NewJITFundingService()

	uc := usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding)

	// Create and activate a card in the repo.
	card := createAndStoreActiveCard(t, repo)

	req := dto.AuthorizeTransactionRequest{
		CardID:           card.ID(),
		Amount:           decimal.NewFromInt(100),
		Currency:         "USD",
		MerchantName:     "Test Merchant",
		MerchantCategory: "5411",
	}

	resp, err := uc.Execute(ctx, req)
	require.NoError(t, err)

	assert.True(t, resp.Approved)
	assert.NotEmpty(t, resp.AuthCode)
	assert.Empty(t, resp.Reason)

	// Verify transaction was saved.
	assert.Len(t, repo.transactions, 1)
	assert.Equal(t, "AUTHORIZED", repo.transactions[0].Status)

	// Verify events were published.
	assert.NotEmpty(t, publisher.publishedEvents)
}

func TestAuthorizeTransactionUseCase_InsufficientFunds(t *testing.T) {
	ctx := context.Background()
	repo := newMockCardRepository()
	publisher := newMockEventPublisher()
	balanceClient := newMockBalanceClient(decimal.NewFromInt(10)) // Only 10 available.
	jitFunding := service.NewJITFundingService()

	uc := usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding)

	card := createAndStoreActiveCard(t, repo)

	req := dto.AuthorizeTransactionRequest{
		CardID:           card.ID(),
		Amount:           decimal.NewFromInt(100), // Requesting 100, only 10 available.
		Currency:         "USD",
		MerchantName:     "Test Merchant",
		MerchantCategory: "5411",
	}

	resp, err := uc.Execute(ctx, req)
	require.NoError(t, err) // Use case returns decline as a response, not an error.

	assert.False(t, resp.Approved)
	assert.Equal(t, "insufficient funds", resp.Reason)
	assert.Empty(t, resp.AuthCode)

	// No transaction should be saved.
	assert.Empty(t, repo.transactions)
}

func TestAuthorizeTransactionUseCase_CardNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newMockCardRepository()
	publisher := newMockEventPublisher()
	balanceClient := newMockBalanceClient(decimal.NewFromInt(10000))
	jitFunding := service.NewJITFundingService()

	uc := usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding)

	req := dto.AuthorizeTransactionRequest{
		CardID:           uuid.New(), // Non-existent card.
		Amount:           decimal.NewFromInt(100),
		Currency:         "USD",
		MerchantName:     "Test Merchant",
		MerchantCategory: "5411",
	}

	resp, err := uc.Execute(ctx, req)
	require.Error(t, err) // Card not found is an error.
	assert.False(t, resp.Approved)
}

func TestAuthorizeTransactionUseCase_ExceedsDailyLimit(t *testing.T) {
	ctx := context.Background()
	repo := newMockCardRepository()
	publisher := newMockEventPublisher()
	balanceClient := newMockBalanceClient(decimal.NewFromInt(100000))
	jitFunding := service.NewJITFundingService()

	uc := usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding)

	card := createAndStoreActiveCard(t, repo)

	// First transaction: 900 (within 1000 daily limit).
	req1 := dto.AuthorizeTransactionRequest{
		CardID:           card.ID(),
		Amount:           decimal.NewFromInt(900),
		Currency:         "USD",
		MerchantName:     "Merchant A",
		MerchantCategory: "5411",
	}

	resp1, err := uc.Execute(ctx, req1)
	require.NoError(t, err)
	assert.True(t, resp1.Approved)

	// Second transaction: 200 (900 + 200 = 1100 > 1000 daily limit).
	req2 := dto.AuthorizeTransactionRequest{
		CardID:           card.ID(),
		Amount:           decimal.NewFromInt(200),
		Currency:         "USD",
		MerchantName:     "Merchant B",
		MerchantCategory: "5411",
	}

	resp2, err := uc.Execute(ctx, req2)
	require.NoError(t, err) // Decline is a response, not an error.
	assert.False(t, resp2.Approved)
	assert.Contains(t, resp2.Reason, "daily spending limit exceeded")
}

func TestAuthorizeTransactionUseCase_FrozenCard(t *testing.T) {
	ctx := context.Background()
	repo := newMockCardRepository()
	publisher := newMockEventPublisher()
	balanceClient := newMockBalanceClient(decimal.NewFromInt(10000))
	jitFunding := service.NewJITFundingService()

	uc := usecase.NewAuthorizeTransactionUseCase(repo, publisher, balanceClient, jitFunding)

	// Create, activate, then freeze.
	card := createAndStoreActiveCard(t, repo)
	frozenCard, err := card.Freeze(time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, frozenCard.ClearEvents()))

	req := dto.AuthorizeTransactionRequest{
		CardID:           card.ID(),
		Amount:           decimal.NewFromInt(50),
		Currency:         "USD",
		MerchantName:     "Test Merchant",
		MerchantCategory: "5411",
	}

	resp, err := uc.Execute(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Approved)
	assert.Contains(t, resp.Reason, "card is not usable")
}

// createAndStoreActiveCard creates an active card and stores it in the mock repo.
func createAndStoreActiveCard(t *testing.T, repo *mockCardRepository) model.Card {
	t.Helper()

	card, err := model.NewCard(
		uuid.New(),
		uuid.New(),
		valueobject.CardTypeVirtual,
		"USD",
		decimal.NewFromInt(1000),
		decimal.NewFromInt(5000),
	)
	require.NoError(t, err)

	card = card.ClearEvents()

	activatedCard, err := card.Activate(time.Now().UTC())
	require.NoError(t, err)

	activatedCard = activatedCard.ClearEvents()
	require.NoError(t, repo.Save(context.Background(), activatedCard))

	return activatedCard
}
