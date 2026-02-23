package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/services/card-service/internal/domain/event"
)

// EventPublisher implements the EventPublisher port using Kafka.
type EventPublisher struct {
	producer *pkgkafka.Producer
	logger   *slog.Logger
	topic    string
}

// NewEventPublisher creates a new EventPublisher.
func NewEventPublisher(producer *pkgkafka.Producer, topic string, logger *slog.Logger) *EventPublisher {
	return &EventPublisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

// Publish sends domain events to Kafka.
func (p *EventPublisher) Publish(ctx context.Context, events []event.DomainEvent) error {
	messages := make([]pkgkafka.Message, 0, len(events))
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.EventType(), err)
		}

		p.logger.DebugContext(ctx, "publishing event to Kafka",
			slog.String("topic", p.topic),
			slog.String("event_type", evt.EventType()),
			slog.Int("payload_size", len(payload)),
		)

		messages = append(messages, pkgkafka.Message{
			Value: payload,
			Headers: map[string]string{
				"event_type": evt.EventType(),
			},
		})
	}

	if len(messages) == 0 {
		return nil
	}

	if err := p.producer.Publish(ctx, p.topic, messages...); err != nil {
		return fmt.Errorf("failed to publish events to topic %s: %w", p.topic, err)
	}

	return nil
}
