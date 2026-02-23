package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/event"
)

// KafkaPublisher publishes domain events to Kafka topics.
type KafkaPublisher struct {
	producer *pkgkafka.Producer
	logger   *slog.Logger
}

// NewKafkaPublisher creates a new KafkaPublisher.
func NewKafkaPublisher(producer *pkgkafka.Producer, logger *slog.Logger) *KafkaPublisher {
	return &KafkaPublisher{
		producer: producer,
		logger:   logger,
	}
}

// Publish publishes one or more domain events to Kafka.
func (p *KafkaPublisher) Publish(ctx context.Context, events ...event.DomainEvent) error {
	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.EventType(), err)
		}

		topic := topicForEvent(evt)

		p.logger.DebugContext(ctx, "publishing event to Kafka",
			"event_type", evt.EventType(),
			"aggregate_id", evt.AggregateID(),
			"topic", topic,
		)

		msg := pkgkafka.Message{
			Key:   []byte(evt.AggregateID()),
			Value: payload,
			Headers: map[string]string{
				"event_type": evt.EventType(),
			},
		}

		if err := p.producer.Publish(ctx, topic, msg); err != nil {
			return fmt.Errorf("failed to publish event %s to topic %s: %w", evt.EventType(), topic, err)
		}
	}

	return nil
}

// topicForEvent returns the Kafka topic for a given domain event.
func topicForEvent(evt event.DomainEvent) string {
	switch evt.(type) {
	case event.ReportGenerated:
		return "reporting.report.generated"
	case event.ReportSubmitted:
		return "reporting.report.submitted"
	case event.ReportAccepted:
		return "reporting.report.accepted"
	case event.ReportRejected:
		return "reporting.report.rejected"
	default:
		return "reporting.unknown"
	}
}
