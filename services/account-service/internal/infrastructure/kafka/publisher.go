package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/services/account-service/internal/domain/event"
)

// Publisher implements port.EventPublisher using Kafka.
type Publisher struct {
	producer *pkgkafka.Producer
	logger   *slog.Logger
}

// NewPublisher creates a new Kafka-based event publisher.
func NewPublisher(producer *pkgkafka.Producer, logger *slog.Logger) *Publisher {
	return &Publisher{
		producer: producer,
		logger:   logger,
	}
}

// Publish sends domain events to the specified Kafka topic.
func (p *Publisher) Publish(ctx context.Context, topic string, events ...event.DomainEvent) error {
	messages := make([]pkgkafka.Message, 0, len(events))
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.EventType(), err)
		}

		key := evt.AggregateID()

		p.logger.DebugContext(ctx, "publishing event",
			"topic", topic,
			"event_type", evt.EventType(),
			"aggregate_id", key,
			"payload_size", len(payload),
		)

		messages = append(messages, pkgkafka.Message{
			Key:   []byte(key),
			Value: payload,
			Headers: map[string]string{
				"event_type":     evt.EventType(),
				"aggregate_type": evt.AggregateType(),
				"event_id":       evt.EventID(),
			},
		})
	}

	if len(messages) == 0 {
		return nil
	}

	if err := p.producer.Publish(ctx, topic, messages...); err != nil {
		return fmt.Errorf("failed to publish events to topic %s: %w", topic, err)
	}
	return nil
}

// Close shuts down the Kafka publisher.
func (p *Publisher) Close() error {
	return p.producer.Close()
}
