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
	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/service"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// --- Mock implementations ---

type mockPaymentRepo struct {
	saveErr      error
	findByIDFunc func(ctx context.Context, id uuid.UUID) (model.PaymentOrder, error)
	listFunc     func(ctx context.Context, id uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error)
}

func (m *mockPaymentRepo) Save(_ context.Context, _ model.PaymentOrder) error {
	return m.saveErr
}

func (m *mockPaymentRepo) FindByID(ctx context.Context, id uuid.UUID) (model.PaymentOrder, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return model.PaymentOrder{}, fmt.Errorf("not found")
}

func (m *mockPaymentRepo) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, accountID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockPaymentRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
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

func buildTestHandler() *PaymentHandler {
	repo := &mockPaymentRepo{}
	publisher := &mockEventPublisher{}
	routingEngine := service.NewRoutingEngine()

	return NewPaymentHandler(
		usecase.NewInitiatePayment(repo, publisher, routingEngine, nil),
		usecase.NewGetPayment(repo),
		usecase.NewListPayments(repo),
	)
}

func buildHandlerWithRepo(repo port.PaymentOrderRepository) *PaymentHandler {
	publisher := &mockEventPublisher{}
	routingEngine := service.NewRoutingEngine()

	return NewPaymentHandler(
		usecase.NewInitiatePayment(repo, publisher, routingEngine, nil),
		usecase.NewGetPayment(repo),
		usecase.NewListPayments(repo),
	)
}

func makeTestPaymentOrder() model.PaymentOrder {
	routingInfo, _ := valueobject.NewRoutingInfo("021000021", "123456789")
	rail, _ := valueobject.NewPaymentRail("ACH")
	st, _ := valueobject.NewPaymentStatus("INITIATED")

	return model.Reconstruct(
		uuid.New(), uuid.New(), uuid.New(), uuid.Nil,
		decimal.NewFromInt(100), "USD", rail, st, routingInfo,
		"REF-001", "Test payment", "",
		time.Now().UTC(), nil, 1, time.Now().UTC(), time.Now().UTC(),
	)
}

// --- Tests ---

func TestHandleInitiatePayment(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleInitiatePayment(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("missing currency returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:        uuid.New().String(),
			SourceAccountID: uuid.New().String(),
			Amount:          "100.00",
			Currency:        "",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "currency is required")
	})

	t.Run("invalid source_account_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:        uuid.New().String(),
			SourceAccountID: "bad-uuid",
			Amount:          "100.00",
			Currency:        "USD",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid source_account_id")
	})

	t.Run("invalid destination_account_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:             uuid.New().String(),
			SourceAccountID:      uuid.New().String(),
			DestinationAccountID: "bad-uuid",
			Amount:               "100.00",
			Currency:             "USD",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid destination_account_id")
	})

	t.Run("invalid amount returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:        uuid.New().String(),
			SourceAccountID: uuid.New().String(),
			Amount:          "not-a-number",
			Currency:        "USD",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
		assert.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("happy path for internal payment", func(t *testing.T) {
		h := buildTestHandler()
		resp, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:             uuid.New().String(),
			SourceAccountID:      uuid.New().String(),
			DestinationAccountID: uuid.New().String(),
			Amount:               "500.00",
			Currency:             "USD",
			Reference:            "TRANSFER-001",
			Description:          "Internal transfer",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.NotEmpty(t, resp.Status)
		assert.NotEmpty(t, resp.Rail)
	})

	t.Run("happy path for external ACH payment", func(t *testing.T) {
		h := buildTestHandler()
		resp, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:              uuid.New().String(),
			SourceAccountID:       uuid.New().String(),
			Amount:                "250.00",
			Currency:              "USD",
			RoutingNumber:         "021000021",
			ExternalAccountNumber: "123456789",
			DestinationCountry:    "US",
			Reference:             "PAY-001",
			Description:           "External payment",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.NotEmpty(t, resp.Status)
	})

	t.Run("save failure returns Internal", func(t *testing.T) {
		repo := &mockPaymentRepo{saveErr: fmt.Errorf("db error")}
		h := buildHandlerWithRepo(repo)

		_, err := h.HandleInitiatePayment(contextWithClaims(), &InitiatePaymentRequest{
			TenantID:             uuid.New().String(),
			SourceAccountID:      uuid.New().String(),
			DestinationAccountID: uuid.New().String(),
			Amount:               "100.00",
			Currency:             "USD",
			Reference:            "REF-001",
			Description:          "test",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestHandleGetPayment(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleGetPayment(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid payment_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleGetPayment(contextWithClaims(), &GetPaymentRequestMsg{PaymentID: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns payment", func(t *testing.T) {
		order := makeTestPaymentOrder()
		repo := &mockPaymentRepo{
			findByIDFunc: func(_ context.Context, id uuid.UUID) (model.PaymentOrder, error) {
				if id == order.ID() {
					return order, nil
				}
				return model.PaymentOrder{}, fmt.Errorf("not found")
			},
		}
		h := buildHandlerWithRepo(repo)

		resp, err := h.HandleGetPayment(contextWithClaims(), &GetPaymentRequestMsg{
			PaymentID: order.ID().String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Payment)
		assert.Equal(t, order.ID().String(), resp.Payment.ID)
	})

	t.Run("not found returns Internal", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleGetPayment(contextWithClaims(), &GetPaymentRequestMsg{
			PaymentID: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestHandleListPayments(t *testing.T) {
	t.Run("nil request returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleListPayments(contextWithClaims(), nil)
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid page_size returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleListPayments(contextWithClaims(), &ListPaymentsRequestMsg{
			TenantID: uuid.New().String(),
			PageSize: -1,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid account_id returns InvalidArgument", func(t *testing.T) {
		h := buildTestHandler()
		_, err := h.HandleListPayments(contextWithClaims(), &ListPaymentsRequestMsg{
			TenantID:  uuid.New().String(),
			AccountID: "bad-uuid",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path returns payments list", func(t *testing.T) {
		h := buildTestHandler()
		resp, err := h.HandleListPayments(contextWithClaims(), &ListPaymentsRequestMsg{
			TenantID: uuid.New().String(),
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.TotalCount)
	})
}

func TestToPaymentOrderMsg(t *testing.T) {
	now := time.Now().UTC()
	orderID := uuid.New()
	tenantID := uuid.New()
	sourceID := uuid.New()

	msg := toPaymentOrderMsg(dto.PaymentOrderResponse{
		ID:                    orderID,
		TenantID:              tenantID,
		SourceAccountID:       sourceID,
		DestinationAccountID:  uuid.Nil,
		Amount:                decimal.NewFromInt(250),
		Currency:              "USD",
		Rail:                  "ACH",
		Status:                "INITIATED",
		RoutingNumber:         "021000021",
		ExternalAccountNumber: "123456789",
		Reference:             "REF-001",
		Description:           "Test",
		FailureReason:         "",
		InitiatedAt:           now,
		SettledAt:             nil,
		Version:               1,
		CreatedAt:             now,
		UpdatedAt:             now,
	})

	assert.Equal(t, orderID.String(), msg.ID)
	assert.Equal(t, tenantID.String(), msg.TenantID)
	assert.Equal(t, "250", msg.Amount)
	assert.Equal(t, "USD", msg.Currency)
	assert.Equal(t, "ACH", msg.Rail)
	assert.Equal(t, "INITIATED", msg.Status)
	assert.Nil(t, msg.SettledAt)
}

// requireGRPCCode asserts that an error is a gRPC status error with the given code.
func requireGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %T: %v", err, err)
	assert.Equal(t, code, st.Code(), "expected gRPC code %s, got %s: %s", code, st.Code(), st.Message())
}
