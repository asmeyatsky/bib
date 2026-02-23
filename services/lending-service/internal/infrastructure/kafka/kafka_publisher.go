package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/services/lending-service/internal/domain/event"
)

// KafkaEventPublisher implements port.EventPublisher by writing events to Kafka.
type KafkaEventPublisher struct {
	producer *pkgkafka.Producer
	topic    string
	logger   *slog.Logger
}

// NewKafkaEventPublisher creates a publisher targeting the given Kafka producer and topic.
func NewKafkaEventPublisher(producer *pkgkafka.Producer, topic string, logger *slog.Logger) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

// Publish serialises and sends domain events to Kafka.
func (p *KafkaEventPublisher) Publish(ctx context.Context, events ...event.DomainEvent) error {
	messages := make([]pkgkafka.Message, 0, len(events))
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", evt.EventType(), err)
		}

		p.logger.DebugContext(ctx, "publishing domain event",
			"event_type", evt.EventType(),
			"aggregate_id", evt.AggregateID(),
			"tenant_id", evt.TenantID(),
			"topic", p.topic,
			"payload_size", len(payload),
		)

		messages = append(messages, pkgkafka.Message{
			Key:   []byte(evt.AggregateID()),
			Value: payload,
			Headers: map[string]string{
				"event_type": evt.EventType(),
				"event_id":   evt.EventID(),
				"tenant_id":  evt.TenantID(),
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
