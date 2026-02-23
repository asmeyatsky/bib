package kafka

// Config holds Kafka connection parameters.
type Config struct {
	ConsumerGroup string

	// SASL configuration for authentication.
	SASLMechanism string // "PLAIN" or "SCRAM-SHA-256" or "SCRAM-SHA-512"
	SASLUsername  string
	SASLPassword  string

	Brokers []string

	// TLS enables TLS for Kafka connections.
	TLS         bool
	SASLEnabled bool
}
