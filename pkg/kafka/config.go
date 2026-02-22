package kafka

// Config holds Kafka connection parameters.
type Config struct {
	Brokers       []string
	ConsumerGroup string

	// TLS enables TLS for Kafka connections.
	TLS bool

	// SASL configuration for authentication.
	SASLEnabled   bool
	SASLMechanism string // "PLAIN" or "SCRAM-SHA-256" or "SCRAM-SHA-512"
	SASLUsername  string
	SASLPassword  string
}
