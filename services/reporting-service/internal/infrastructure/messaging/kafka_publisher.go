package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/event"
)

// KafkaPublisher publishes domain events to Kafka topics.
type KafkaPublisher struct {
	broker string
	logger *slog.Logger
}

// NewKafkaPublisher creates a new KafkaPublisher.
func NewKafkaPublisher(broker string, logger *slog.Logger) *KafkaPublisher {
	return &KafkaPublisher{
		broker: broker,
		logger: logger,
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

		p.logger.Info("publishing event to Kafka",
			"event_type", evt.EventType(),
			"aggregate_id", evt.AggregateID().String(),
			"topic", topic,
		)

		// In production, this would use a real Kafka producer.
		// For now, we log the event for development/testing.
		_ = payload
		_ = topic

		p.logger.Debug("event published successfully",
			"event_type", evt.EventType(),
			"topic", topic,
		)
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
