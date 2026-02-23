package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	ServiceName string
	Port        int
}

// InitMetrics initializes the Prometheus metrics exporter.
// Returns the MeterProvider and an HTTP handler for /metrics endpoint.
func InitMetrics(_ MetricsConfig) (*sdkmetric.MeterProvider, http.Handler, error) {
	exporter, err := promexporter.New()
	if err != nil {
		return nil, nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	handler := promhttp.Handler()

	return provider, handler, nil
}
