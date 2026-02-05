package client

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains Prometheus metrics for RPC monitoring.
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
	errorsTotal     *prometheus.CounterVec
}

// NewMetrics creates and registers Prometheus metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotron_rpc_requests_total",
				Help: "Total number of RPC requests",
			},
			[]string{"method", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gotron_rpc_duration_seconds",
				Help:    "RPC request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gotron_rpc_in_flight",
				Help: "Number of RPC requests currently in progress",
			},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotron_rpc_errors_total",
				Help: "Total number of RPC errors by type",
			},
			[]string{"method", "error_type"},
		),
	}

	reg.MustRegister(m.requestsTotal, m.requestDuration, m.requestsInFlight, m.errorsTotal)
	return m
}

// IncInFlight increments the in-flight requests counter.
func (m *Metrics) IncInFlight() {
	m.requestsInFlight.Inc()
}

// DecInFlight decrements the in-flight requests counter.
func (m *Metrics) DecInFlight() {
	m.requestsInFlight.Dec()
}

// RecordRequest records metrics for an RPC request.
func (m *Metrics) RecordRequest(method string, duration float64, err error) {
	status := "success"
	if err != nil {
		status = "error"
		m.recordError(method, err)
	}
	m.requestsTotal.WithLabelValues(method, status).Inc()
	m.requestDuration.WithLabelValues(method).Observe(duration)
}

// recordError categorizes and records error metrics.
func (m *Metrics) recordError(method string, err error) {
	errorType := classifyError(err)
	m.errorsTotal.WithLabelValues(method, errorType).Inc()
}

// classifyError determines the error type for metrics.
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()

	switch {
	case contains(errStr, "context deadline exceeded", "timeout"):
		return "timeout"
	case contains(errStr, "connection refused", "connection reset", "no route to host", "network is unreachable"):
		return "connection"
	case contains(errStr, "context canceled"):
		return "canceled"
	case contains(errStr, "unavailable", "service unavailable"):
		return "unavailable"
	default:
		return "other"
	}
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
