package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/pkg/events"
	pkgkafka "github.com/bibbank/bib/pkg/kafka"
)

// Publisher implements port.EventPublisher using Kafka.
type Publisher struct {
	producer *pkgkafka.Producer
	logger   *slog.Logger
	topic    string
}

// NewPublisher creates a new Kafka event publisher.
func NewPublisher(producer *pkgkafka.Producer, topic string, logger *slog.Logger) *Publisher {
	return &Publisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

// Publish sends domain events to Kafka.
func (p *Publisher) Publish(ctx context.Context, domainEvents ...events.DomainEvent) error {
	messages := make([]pkgkafka.Message, 0, len(domainEvents))
	for _, evt := range domainEvents {
		eventType := evt.EventType()

		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", eventType, err)
		}

		p.logger.DebugContext(ctx, "publishing event",
			slog.String("event_type", eventType),
			slog.String("topic", p.topic),
			slog.Int("payload_size", len(payload)),
		)

		messages = append(messages, pkgkafka.Message{
			Value: payload,
			Headers: map[string]string{
				"event_type": eventType,
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
