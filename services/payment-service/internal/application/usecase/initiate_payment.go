package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/service"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

const TopicPaymentOrders = "bib.payment.orders"

// InitiatePayment handles the creation of new payment orders.
type InitiatePayment struct {
	paymentRepo   port.PaymentOrderRepository
	publisher     port.EventPublisher
	routingEngine *service.RoutingEngine
	fraudClient   port.FraudClient // optional, may be nil
}

func NewInitiatePayment(
	paymentRepo port.PaymentOrderRepository,
	publisher port.EventPublisher,
	routingEngine *service.RoutingEngine,
	fraudClient port.FraudClient,
) *InitiatePayment {
	return &InitiatePayment{
		paymentRepo:   paymentRepo,
		publisher:     publisher,
		routingEngine: routingEngine,
		fraudClient:   fraudClient,
	}
}

func (uc *InitiatePayment) Execute(ctx context.Context, req dto.InitiatePaymentRequest) (dto.InitiatePaymentResponse, error) {
	// Validate routing info for external payments.
	routingInfo, err := valueobject.NewRoutingInfo(req.RoutingNumber, req.ExternalAccountNumber)
	if err != nil {
		return dto.InitiatePaymentResponse{}, fmt.Errorf("invalid routing info: %w", err)
	}

	// Determine if the payment is internal.
	isInternal := req.DestinationAccountID != uuid.Nil

	// Optionally assess fraud risk.
	if uc.fraudClient != nil {
		approved, assessErr := uc.fraudClient.AssessTransaction(ctx, req.TenantID, req.SourceAccountID, req.Amount, req.Currency)
		if assessErr != nil {
			return dto.InitiatePaymentResponse{}, fmt.Errorf("fraud assessment failed: %w", assessErr)
		}
		if !approved {
			return dto.InitiatePaymentResponse{}, fmt.Errorf("payment rejected by fraud assessment")
		}
	}

	// Select optimal payment rail via the routing engine.
	rail := uc.routingEngine.SelectRail(req.Amount, req.Currency, isInternal, req.DestinationCountry)

	// Create the payment order aggregate.
	order, err := model.NewPaymentOrder(
		req.TenantID,
		req.SourceAccountID,
		req.DestinationAccountID,
		req.Amount,
		req.Currency,
		rail,
		routingInfo,
		req.Reference,
		req.Description,
	)
	if err != nil {
		return dto.InitiatePaymentResponse{}, fmt.Errorf("failed to create payment order: %w", err)
	}

	// Persist the order.
	if err := uc.paymentRepo.Save(ctx, order); err != nil {
		return dto.InitiatePaymentResponse{}, fmt.Errorf("failed to save payment order: %w", err)
	}

	// Publish domain events.
	if events := order.DomainEvents(); len(events) > 0 {
		if err := uc.publisher.Publish(ctx, TopicPaymentOrders, events...); err != nil {
			return dto.InitiatePaymentResponse{}, fmt.Errorf("failed to publish events: %w", err)
		}
	}

	return dto.InitiatePaymentResponse{
		ID:        order.ID(),
		Status:    order.Status().String(),
		Rail:      order.Rail().String(),
		CreatedAt: order.CreatedAt(),
	}, nil
}
