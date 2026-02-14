package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/event"
)

// KafkaPublisher implements port.EventPublisher using Kafka.
type KafkaPublisher struct {
	brokers []string
	topic   string
	logger  *slog.Logger
}

// NewKafkaPublisher creates a new Kafka event publisher.
func NewKafkaPublisher(brokers []string, topic string, logger *slog.Logger) *KafkaPublisher {
	return &KafkaPublisher{
		brokers: brokers,
		topic:   topic,
		logger:  logger,
	}
}

// Publish sends domain events to Kafka.
func (p *KafkaPublisher) Publish(ctx context.Context, events ...interface{}) error {
	for _, evt := range events {
		eventType := "unknown"
		switch e := evt.(type) {
		case event.AssessmentCompleted:
			eventType = e.EventType()
		case event.HighRiskDetected:
			eventType = e.EventType()
		}

		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", eventType, err)
		}

		p.logger.Info("publishing event",
			slog.String("event_type", eventType),
			slog.String("topic", p.topic),
			slog.Int("payload_size", len(payload)),
		)

		// TODO: Integrate with actual Kafka producer from pkg/kafka.
		// For now, log the event for development.
		p.logger.Debug("event payload",
			slog.String("event_type", eventType),
			slog.String("payload", string(payload)),
		)
	}

	return nil
}
