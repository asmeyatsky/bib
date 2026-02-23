package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
)

// ProcessPayment handles the processing of payment orders.
// It is typically triggered by a Kafka consumer after a PaymentInitiated event.
type ProcessPayment struct {
	paymentRepo port.PaymentOrderRepository
	railAdapter port.RailAdapter
	publisher   port.EventPublisher
}

func NewProcessPayment(
	paymentRepo port.PaymentOrderRepository,
	railAdapter port.RailAdapter,
	publisher port.EventPublisher,
) *ProcessPayment {
	return &ProcessPayment{
		paymentRepo: paymentRepo,
		railAdapter: railAdapter,
		publisher:   publisher,
	}
}

func (uc *ProcessPayment) Execute(ctx context.Context, paymentID uuid.UUID) error {
	// Fetch the order.
	order, err := uc.paymentRepo.FindByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to find payment order %s: %w", paymentID, err)
	}

	now := time.Now().UTC()

	// Transition to PROCESSING.
	processing, err := order.MarkProcessing(now)
	if err != nil {
		return fmt.Errorf("failed to mark processing: %w", err)
	}

	// Persist the PROCESSING state.
	if saveErr := uc.paymentRepo.Save(ctx, processing); saveErr != nil {
		return fmt.Errorf("failed to save processing state: %w", saveErr)
	}

	// Submit to the rail adapter.
	submitErr := uc.railAdapter.Submit(ctx, processing)

	now = time.Now().UTC()
	if submitErr != nil {
		// Rail submission failed; mark the order as FAILED.
		failed, failErr := processing.Fail(submitErr.Error(), now)
		if failErr != nil {
			return fmt.Errorf("failed to mark failure after submit error: %w (submit error: %v)", failErr, submitErr)
		}

		if saveErr := uc.paymentRepo.Save(ctx, failed); saveErr != nil {
			return fmt.Errorf("failed to save failed state: %w", saveErr)
		}

		if events := failed.DomainEvents(); len(events) > 0 {
			if pubErr := uc.publisher.Publish(ctx, TopicPaymentOrders, events...); pubErr != nil {
				return fmt.Errorf("failed to publish failure events: %w", pubErr)
			}
		}

		return nil
	}

	// Rail submission succeeded; mark the order as SETTLED.
	settled, err := processing.Settle(now)
	if err != nil {
		return fmt.Errorf("failed to mark settled: %w", err)
	}

	if saveErr := uc.paymentRepo.Save(ctx, settled); saveErr != nil {
		return fmt.Errorf("failed to save settled state: %w", saveErr)
	}

	if events := settled.DomainEvents(); len(events) > 0 {
		if pubErr := uc.publisher.Publish(ctx, TopicPaymentOrders, events...); pubErr != nil {
			return fmt.Errorf("failed to publish settlement events: %w", pubErr)
		}
	}

	return nil
}
