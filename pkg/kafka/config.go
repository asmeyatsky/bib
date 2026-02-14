package kafka

// Config holds Kafka connection parameters.
type Config struct {
	Brokers       []string
	ConsumerGroup string
	TLS           bool
}
