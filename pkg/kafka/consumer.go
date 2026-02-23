package kafka

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Handler processes a consumed Kafka message.
type Handler func(ctx context.Context, msg Message) error

// Consumer wraps kafka-go reader for consuming messages.
type Consumer struct {
	reader  *kafkago.Reader
	handler Handler
	logger  *slog.Logger
}

// NewConsumer creates a new Consumer for the given topic with the provided handler.
func NewConsumer(cfg Config, topic string, handler Handler, logger *slog.Logger) *Consumer {
	readerCfg := kafkago.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    topic,
		GroupID:  cfg.ConsumerGroup,
		MinBytes: 1,
		MaxBytes: 10 * 1024 * 1024, // 10 MB
	}

	// Configure dialer for TLS and SASL authentication.
	if cfg.TLS || cfg.SASLEnabled {
		dialer := &kafkago.Dialer{}

		if cfg.TLS {
			dialer.TLS = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		if cfg.SASLEnabled {
			dialer.SASLMechanism = resolveConsumerSASL(cfg)
		}

		readerCfg.Dialer = dialer
	}

	r := kafkago.NewReader(readerCfg)

	return &Consumer{
		reader:  r,
		handler: handler,
		logger:  logger,
	}
}

// resolveConsumerSASL returns the appropriate SASL mechanism for the consumer.
func resolveConsumerSASL(cfg Config) sasl.Mechanism {
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

// Start begins consuming messages. Blocks until the context is canceled.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("consumer starting", "topic", c.reader.Config().Topic, "group", c.reader.Config().GroupID)

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.logger.Info("consumer stopping due to context cancellation")
				return nil
			}
			return fmt.Errorf("fetching message: %w", err)
		}

		msg := Message{
			Key:     m.Key,
			Value:   m.Value,
			Headers: make(map[string]string, len(m.Headers)),
		}
		for _, h := range m.Headers {
			msg.Headers[h.Key] = string(h.Value)
		}

		if err := c.handler(ctx, msg); err != nil {
			c.logger.Error("handler error",
				"topic", m.Topic,
				"partition", m.Partition,
				"offset", m.Offset,
				"error", err,
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			c.logger.Error("commit error",
				"topic", m.Topic,
				"partition", m.Partition,
				"offset", m.Offset,
				"error", err,
			)
		}
	}
}

// Close closes the reader.
func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("closing kafka reader: %w", err)
	}
	return nil
}
