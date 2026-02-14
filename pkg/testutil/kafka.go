package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

// KafkaContainer wraps a testcontainers Kafka instance.
type KafkaContainer struct {
	Container *kafka.KafkaContainer
	Brokers   []string
}

// NewKafkaContainer starts a Kafka container for testing.
// The caller should defer container.Cleanup(t).
func NewKafkaContainer(ctx context.Context, t *testing.T) *KafkaContainer {
	t.Helper()

	kafkaContainer, err := kafka.Run(ctx,
		"confluentinc/confluent-local:7.6.1",
		kafka.WithClusterID("test-cluster"),
	)
	if err != nil {
		t.Fatalf("failed to start kafka container: %v", err)
	}

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		t.Fatalf("failed to get kafka brokers: %v", err)
	}

	return &KafkaContainer{
		Container: kafkaContainer,
		Brokers:   brokers,
	}
}

// Cleanup terminates the container.
func (kc *KafkaContainer) Cleanup(t *testing.T) {
	t.Helper()

	if kc.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := kc.Container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate kafka container: %v", err)
		}
	}
}
