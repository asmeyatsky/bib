package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bibbank/bib/pkg/events"
	pkgkafka "github.com/bibbank/bib/pkg/kafka"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
)

var _ port.EventPublisher = (*Publisher)(nil)

// Publisher implements EventPublisher using Kafka.
type Publisher struct {
	producer *pkgkafka.Producer
}

func NewPublisher(producer *pkgkafka.Producer) *Publisher {
	return &Publisher{producer: producer}
}

func (p *Publisher) Publish(ctx context.Context, topic string, domainEvents ...events.DomainEvent) error {
	var messages []pkgkafka.Message
	for _, evt := range domainEvents {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", evt.EventType(), err)
		}
		messages = append(messages, pkgkafka.Message{
			Key:   []byte(evt.AggregateID()),
			Value: payload,
			Headers: map[string]string{
				"event_type":     evt.EventType(),
				"aggregate_type": evt.AggregateType(),
				"event_id":       evt.EventID(),
			},
		})
	}
	if err := p.producer.Publish(ctx, topic, messages...); err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}
