package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/service"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockPaymentOrderRepository struct {
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.PaymentOrder, error)
	saveFunc     func(ctx context.Context, order model.PaymentOrder) error
	savedOrders  []model.PaymentOrder
}

func (m *mockPaymentOrderRepository) Save(ctx context.Context, order model.PaymentOrder) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, order)
	}
	m.savedOrders = append(m.savedOrders, order)
	return nil
}

func (m *mockPaymentOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (model.PaymentOrder, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.PaymentOrder{}, fmt.Errorf("payment order not found: %s", id)
}

func (m *mockPaymentOrderRepository) ListByAccount(_ context.Context, _ uuid.UUID, _, _ int) ([]model.PaymentOrder, int, error) {
	return nil, 0, nil
}

func (m *mockPaymentOrderRepository) ListByTenant(_ context.Context, _ uuid.UUID, _, _ int) ([]model.PaymentOrder, int, error) {
	return nil, 0, nil
}

type mockEventPublisher struct {
	publishFunc     func(ctx context.Context, topic string, events ...events.DomainEvent) error
	publishedEvents []events.DomainEvent
}

func (m *mockEventPublisher) Publish(ctx context.Context, topic string, evts ...events.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

type mockFraudClient struct {
	assessFunc func(ctx context.Context, tenantID, accountID uuid.UUID, amount decimal.Decimal, currency string) (bool, error)
}

func (m *mockFraudClient) AssessTransaction(ctx context.Context, tenantID, accountID uuid.UUID, amount decimal.Decimal, currency string) (bool, error) {
	if m.assessFunc != nil {
		return m.assessFunc(ctx, tenantID, accountID, amount, currency)
	}
	return true, nil
}

// --- Tests ---

func validInitiateRequest() dto.InitiatePaymentRequest {
	return dto.InitiatePaymentRequest{
		TenantID:              uuid.New(),
		SourceAccountID:       uuid.New(),
		DestinationAccountID:  uuid.Nil,
		Amount:                decimal.NewFromInt(1000),
		Currency:              "USD",
		RoutingNumber:         "021000021",
		ExternalAccountNumber: "123456789",
		DestinationCountry:    "US",
		Reference:             "PAY-001",
		Description:           "ACH payment",
	}
}

func TestInitiatePayment_Success(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()

	uc := usecase.NewInitiatePayment(repo, publisher, engine, nil)

	req := validInitiateRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "INITIATED", resp.Status)
	assert.Equal(t, "ACH", resp.Rail) // USD domestic -> ACH
	assert.False(t, resp.CreatedAt.IsZero())

	// Verify order was saved.
	require.Len(t, repo.savedOrders, 1)
	saved := repo.savedOrders[0]
	assert.Equal(t, req.TenantID, saved.TenantID())
	assert.Equal(t, req.SourceAccountID, saved.SourceAccountID())
	assert.True(t, req.Amount.Equal(saved.Amount()))
	assert.Equal(t, "USD", saved.Currency())
	assert.Equal(t, valueobject.RailACH, saved.Rail())

	// Verify events were published.
	assert.NotEmpty(t, publisher.publishedEvents)
	assert.Equal(t, "payment.order.initiated", publisher.publishedEvents[0].EventType())
}

func TestInitiatePayment_InternalTransfer(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()

	uc := usecase.NewInitiatePayment(repo, publisher, engine, nil)

	req := dto.InitiatePaymentRequest{
		TenantID:             uuid.New(),
		SourceAccountID:      uuid.New(),
		DestinationAccountID: uuid.New(), // internal destination
		Amount:               decimal.NewFromInt(500),
		Currency:             "USD",
		Reference:            "INT-001",
		Description:          "Internal transfer",
	}

	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "INTERNAL", resp.Rail)
	assert.Equal(t, "INITIATED", resp.Status)
}

func TestInitiatePayment_EURRoutesSEPA(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()

	uc := usecase.NewInitiatePayment(repo, publisher, engine, nil)

	req := validInitiateRequest()
	req.Currency = "EUR"
	req.DestinationCountry = "DE"

	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "SEPA", resp.Rail)
}

func TestInitiatePayment_InvalidRoutingInfo(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()

	uc := usecase.NewInitiatePayment(repo, publisher, engine, nil)

	req := validInitiateRequest()
	req.RoutingNumber = "INVALID" // not 9 digits

	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid routing info")
	assert.Empty(t, repo.savedOrders)
}

func TestInitiatePayment_FraudRejected(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()
	fraudClient := &mockFraudClient{
		assessFunc: func(_ context.Context, _, _ uuid.UUID, _ decimal.Decimal, _ string) (bool, error) {
			return false, nil
		},
	}

	uc := usecase.NewInitiatePayment(repo, publisher, engine, fraudClient)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "payment rejected by fraud assessment")
	assert.Empty(t, repo.savedOrders)
	assert.Empty(t, publisher.publishedEvents)
}

func TestInitiatePayment_FraudServiceError(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()
	fraudClient := &mockFraudClient{
		assessFunc: func(_ context.Context, _, _ uuid.UUID, _ decimal.Decimal, _ string) (bool, error) {
			return false, fmt.Errorf("fraud service unavailable")
		},
	}

	uc := usecase.NewInitiatePayment(repo, publisher, engine, fraudClient)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fraud assessment failed")
	assert.Contains(t, err.Error(), "fraud service unavailable")
	assert.Empty(t, repo.savedOrders)
}

func TestInitiatePayment_FraudApproved(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()
	fraudClient := &mockFraudClient{
		assessFunc: func(_ context.Context, _, _ uuid.UUID, _ decimal.Decimal, _ string) (bool, error) {
			return true, nil
		},
	}

	uc := usecase.NewInitiatePayment(repo, publisher, engine, fraudClient)

	req := validInitiateRequest()
	resp, err := uc.Execute(context.Background(), req)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "INITIATED", resp.Status)
	require.Len(t, repo.savedOrders, 1)
}

func TestInitiatePayment_RepoSaveError(t *testing.T) {
	repo := &mockPaymentOrderRepository{
		saveFunc: func(_ context.Context, _ model.PaymentOrder) error {
			return fmt.Errorf("database connection lost")
		},
	}
	publisher := &mockEventPublisher{}
	engine := service.NewRoutingEngine()

	uc := usecase.NewInitiatePayment(repo, publisher, engine, nil)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save payment order")
	assert.Contains(t, err.Error(), "database connection lost")
	assert.Empty(t, publisher.publishedEvents)
}

func TestInitiatePayment_PublishError(t *testing.T) {
	repo := &mockPaymentOrderRepository{}
	publisher := &mockEventPublisher{
		publishFunc: func(_ context.Context, _ string, _ ...events.DomainEvent) error {
			return fmt.Errorf("broker unreachable")
		},
	}
	engine := service.NewRoutingEngine()

	uc := usecase.NewInitiatePayment(repo, publisher, engine, nil)

	req := validInitiateRequest()
	_, err := uc.Execute(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish events")
	assert.Contains(t, err.Error(), "broker unreachable")

	// Order was saved even though publish failed.
	require.Len(t, repo.savedOrders, 1)
}
