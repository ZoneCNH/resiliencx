package templatex

const (
	MetricClientCreatedTotal           = "client_created_total"
	MetricClientClosedTotal            = "client_closed_total"
	MetricClientErrorsTotal            = "client_errors_total"
	MetricClientHealthStatus           = "client_health_status"
	MetricClientHealthLatencyMS        = "client_health_latency_ms"
	MetricClientRequestsTotal          = "client_requests_total"
	MetricClientRequestDurationSeconds = "client_request_duration_seconds"
	MetricClientRetriesTotal           = "client_retries_total"
	MetricClientInflight               = "client_inflight"
)

// Metrics defines the interface for collecting client observability data.
// Implementations can export counters, histograms, and gauges to any metrics backend
// (e.g., Prometheus, StatsD, OpenTelemetry). Use [NoopMetrics] if metrics collection is not needed.
type Metrics interface {
	IncCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

// NoopMetrics is a no-op implementation of [Metrics] that silently discards all recorded data.
// It is the default metrics client used when no custom [Metrics] implementation is provided via [WithMetrics].
type NoopMetrics struct{}

// IncCounter discards the counter increment. Implements [Metrics].
func (NoopMetrics) IncCounter(name string, labels map[string]string) {}

// ObserveHistogram discards the histogram observation. Implements [Metrics].
func (NoopMetrics) ObserveHistogram(name string, value float64, labels map[string]string) {}

// SetGauge discards the gauge update. Implements [Metrics].
func (NoopMetrics) SetGauge(name string, value float64, labels map[string]string) {}
