package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/services/account-service/internal/domain/event"
)

// Publisher implements port.EventPublisher using Kafka.
type Publisher struct {
	brokers []string
	logger  *slog.Logger
}

// NewPublisher creates a new Kafka-based event publisher.
func NewPublisher(brokers []string, logger *slog.Logger) *Publisher {
	return &Publisher{
		brokers: brokers,
		logger:  logger,
	}
}

// Publish sends domain events to the specified Kafka topic.
func (p *Publisher) Publish(ctx context.Context, topic string, events ...event.DomainEvent) error {
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.EventType(), err)
		}

		key := evt.AggregateID().String()

		p.logger.Info("publishing event",
			"topic", topic,
			"event_type", evt.EventType(),
			"aggregate_id", key,
			"payload_size", len(payload),
		)

		// In a real implementation, this would use a Kafka producer client.
		// For now, we log the event. The outbox pattern in the repository
		// ensures events are not lost even if Kafka is temporarily unavailable.
		if err := p.publishMessage(ctx, topic, key, payload); err != nil {
			return fmt.Errorf("failed to publish event %s to topic %s: %w", evt.EventType(), topic, err)
		}
	}
	return nil
}

// publishMessage sends a single message to a Kafka topic.
// This is a placeholder that should be replaced with actual Kafka producer logic.
func (p *Publisher) publishMessage(ctx context.Context, topic, key string, payload []byte) error {
	// TODO: Integrate with actual Kafka producer (e.g., confluent-kafka-go or segmentio/kafka-go).
	// The outbox relay process would typically handle this, reading from the outbox table
	// and publishing to Kafka, then marking events as published.
	p.logger.Debug("kafka message published",
		"topic", topic,
		"key", key,
		"payload_size", len(payload),
	)
	return nil
}

// Close shuts down the Kafka publisher.
func (p *Publisher) Close() error {
	p.logger.Info("closing kafka publisher")
	return nil
}
