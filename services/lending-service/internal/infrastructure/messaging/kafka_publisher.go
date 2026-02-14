package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/services/lending-service/internal/domain/event"
)

// KafkaEventPublisher implements port.EventPublisher by writing events to Kafka.
type KafkaEventPublisher struct {
	broker string
	topic  string
	logger *slog.Logger
}

// NewKafkaEventPublisher creates a publisher targeting the given broker and topic.
func NewKafkaEventPublisher(broker, topic string, logger *slog.Logger) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		broker: broker,
		topic:  topic,
		logger: logger,
	}
}

// Publish serialises and sends domain events. In a production implementation
// this would use a proper Kafka producer; here we provide a stub that logs
// the events for development/testing purposes.
func (p *KafkaEventPublisher) Publish(ctx context.Context, events ...event.DomainEvent) error {
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", evt.EventType(), err)
		}

		p.logger.Info("publishing domain event",
			"event_type", evt.EventType(),
			"aggregate_id", evt.AggregateID(),
			"tenant_id", evt.TenantID(),
			"topic", p.topic,
			"payload_size", len(payload),
		)

		// TODO: replace with actual Kafka producer call
		// producer.Produce(&kafka.Message{
		//     TopicPartition: kafka.TopicPartition{Topic: &p.topic},
		//     Key:            []byte(evt.AggregateID()),
		//     Value:          payload,
		// }, nil)
	}
	return nil
}
