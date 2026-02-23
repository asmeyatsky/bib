package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Message represents a Kafka message.
type Message struct {
	Headers map[string]string
	Key     []byte
	Value   []byte
}

// Producer wraps kafka-go writer for publishing messages.
type Producer struct {
	writers   map[string]*kafkago.Writer
	transport *kafkago.Transport
	brokers   []string
	mu        sync.Mutex
}

// NewProducer creates a new Producer with the given configuration.
func NewProducer(cfg Config) *Producer {
	transport := &kafkago.Transport{}

	if cfg.TLS {
		transport.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	if cfg.SASLEnabled {
		mechanism := resolveSASLMechanism(cfg)
		if mechanism != nil {
			transport.SASL = mechanism
		}
	}

	return &Producer{
		writers:   make(map[string]*kafkago.Writer),
		brokers:   cfg.Brokers,
		transport: transport,
	}
}

// resolveSASLMechanism returns the appropriate SASL mechanism based on config.
func resolveSASLMechanism(cfg Config) sasl.Mechanism {
	switch cfg.SASLMechanism {
	case "SCRAM-SHA-256":
		m, err := scram.Mechanism(scram.SHA256, cfg.SASLUsername, cfg.SASLPassword)
		if err != nil {
			return nil
		}
		return m
	case "SCRAM-SHA-512":
		m, err := scram.Mechanism(scram.SHA512, cfg.SASLUsername, cfg.SASLPassword)
		if err != nil {
			return nil
		}
		return m
	case "PLAIN", "":
		return &plain.Mechanism{
			Username: cfg.SASLUsername,
			Password: cfg.SASLPassword,
		}
	default:
		return nil
	}
}

// Publish sends messages to the specified topic.
func (p *Producer) Publish(ctx context.Context, topic string, messages ...Message) error {
	w := p.getOrCreateWriter(topic)

	kafkaMessages := make([]kafkago.Message, 0, len(messages))
	for _, msg := range messages {
		km := kafkago.Message{
			Key:   msg.Key,
			Value: msg.Value,
		}
		for k, v := range msg.Headers {
			km.Headers = append(km.Headers, kafkago.Header{
				Key:   k,
				Value: []byte(v),
			})
		}
		kafkaMessages = append(kafkaMessages, km)
	}

	if err := w.WriteMessages(ctx, kafkaMessages...); err != nil {
		return fmt.Errorf("kafka publish to %s: %w", topic, err)
	}
	return nil
}

// Close closes all writers.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for topic, w := range p.writers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("closing writer for topic %s: %w", topic, err)
		}
	}
	p.writers = make(map[string]*kafkago.Writer)
	return firstErr
}

// getOrCreateWriter lazily creates a writer for a topic.
func (p *Producer) getOrCreateWriter(topic string) *kafkago.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()

	if w, ok := p.writers[topic]; ok {
		return w
	}

	w := &kafkago.Writer{
		Addr:         kafkago.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafkago.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireAll,
		Transport:    p.transport,
	}
	p.writers[topic] = w
	return w
}
