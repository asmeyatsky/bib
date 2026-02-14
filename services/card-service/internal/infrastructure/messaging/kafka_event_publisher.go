package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/services/card-service/internal/domain/event"
)

// KafkaEventPublisher implements the EventPublisher port using Kafka.
type KafkaEventPublisher struct {
	brokerAddr string
	topic      string
	logger     *slog.Logger
}

// NewKafkaEventPublisher creates a new KafkaEventPublisher.
func NewKafkaEventPublisher(brokerAddr, topic string, logger *slog.Logger) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		brokerAddr: brokerAddr,
		topic:      topic,
		logger:     logger,
	}
}

// Publish sends domain events to Kafka.
func (p *KafkaEventPublisher) Publish(ctx context.Context, events []event.DomainEvent) error {
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.EventType(), err)
		}

		p.logger.Info("publishing event to Kafka",
			slog.String("topic", p.topic),
			slog.String("event_type", evt.EventType()),
			slog.String("payload_size", fmt.Sprintf("%d bytes", len(payload))),
		)

		// TODO: integrate with actual Kafka producer from pkg/kafka.
		// For now, events are persisted via the outbox pattern in the repository.
		// The outbox relay will pick up events and publish them to Kafka.
		_ = payload
	}

	return nil
}
