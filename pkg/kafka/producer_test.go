package kafka

import (
	"testing"
)

func TestNewProducer(t *testing.T) {
	cfg := Config{
		Brokers:       []string{"localhost:9092", "localhost:9093"},
		ConsumerGroup: "test-group",
		TLS:           false,
	}

	p := NewProducer(cfg)
	if p == nil {
		t.Fatal("expected non-nil producer")
	}
	if len(p.brokers) != 2 {
		t.Fatalf("expected 2 brokers, got %d", len(p.brokers))
	}
	if p.brokers[0] != "localhost:9092" {
		t.Errorf("expected broker localhost:9092, got %s", p.brokers[0])
	}
	if p.brokers[1] != "localhost:9093" {
		t.Errorf("expected broker localhost:9093, got %s", p.brokers[1])
	}
	if p.writers == nil {
		t.Fatal("expected writers map to be initialized")
	}
	if len(p.writers) != 0 {
		t.Errorf("expected empty writers map, got %d entries", len(p.writers))
	}
}

func TestNewProducerSingleBroker(t *testing.T) {
	cfg := Config{
		Brokers: []string{"kafka:9092"},
	}

	p := NewProducer(cfg)
	if p == nil {
		t.Fatal("expected non-nil producer")
	}
	if len(p.brokers) != 1 {
		t.Fatalf("expected 1 broker, got %d", len(p.brokers))
	}
}

func TestMessageConstruction(t *testing.T) {
	msg := Message{
		Key:   []byte("order-123"),
		Value: []byte(`{"amount":100}`),
		Headers: map[string]string{
			"content-type":   "application/json",
			"correlation-id": "abc-def-ghi",
		},
	}

	if string(msg.Key) != "order-123" {
		t.Errorf("expected key order-123, got %s", string(msg.Key))
	}
	if string(msg.Value) != `{"amount":100}` {
		t.Errorf("unexpected value: %s", string(msg.Value))
	}
	if len(msg.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(msg.Headers))
	}
	if msg.Headers["content-type"] != "application/json" {
		t.Errorf("unexpected content-type header: %s", msg.Headers["content-type"])
	}
	if msg.Headers["correlation-id"] != "abc-def-ghi" {
		t.Errorf("unexpected correlation-id header: %s", msg.Headers["correlation-id"])
	}
}

func TestMessageNilHeaders(t *testing.T) {
	msg := Message{}

	if msg.Headers != nil {
		t.Error("expected nil headers when not set")
	}
}

func TestGetOrCreateWriter(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	p := NewProducer(cfg)

	w1 := p.getOrCreateWriter("topic-a")
	if w1 == nil {
		t.Fatal("expected non-nil writer")
	}

	// Same topic should return the same writer instance.
	w2 := p.getOrCreateWriter("topic-a")
	if w1 != w2 {
		t.Error("expected same writer instance for same topic")
	}

	// Different topic should return a different writer.
	w3 := p.getOrCreateWriter("topic-b")
	if w3 == nil {
		t.Fatal("expected non-nil writer for topic-b")
	}
	if w1 == w3 {
		t.Error("expected different writer instance for different topic")
	}

	if len(p.writers) != 2 {
		t.Errorf("expected 2 writers, got %d", len(p.writers))
	}
}

func TestProducerClose(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	p := NewProducer(cfg)

	// Create a few writers.
	_ = p.getOrCreateWriter("topic-a")
	_ = p.getOrCreateWriter("topic-b")

	if len(p.writers) != 2 {
		t.Fatalf("expected 2 writers before close, got %d", len(p.writers))
	}

	err := p.Close()
	if err != nil {
		t.Fatalf("unexpected error on close: %v", err)
	}

	if len(p.writers) != 0 {
		t.Errorf("expected 0 writers after close, got %d", len(p.writers))
	}
}
