package client

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector defines the interface for collecting RPC metrics.
// Implementations can use their own Prometheus metrics with custom labels.
type MetricsCollector interface {
	// RecordRequest records an RPC request with result.
	// blockchain - the blockchain identifier (e.g., "tron")
	// method - RPC method name
	// status - "success" or "error"
	// duration - request duration
	RecordRequest(blockchain, method, status string, duration time.Duration)

	// RecordRetry records a retry attempt.
	RecordRetry(blockchain, method string)

	// SetPoolHealth updates node pool health metrics.
	SetPoolHealth(blockchain string, total, healthy, disabled int)
}

// Metrics contains built-in Prometheus metrics for RPC monitoring.
// It implements MetricsCollector.
type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
	retriesTotal     *prometheus.CounterVec
	poolTotal        *prometheus.GaugeVec
	poolHealthy      *prometheus.GaugeVec
	poolDisabled     *prometheus.GaugeVec
}

var _ MetricsCollector = (*Metrics)(nil)

// NewMetrics creates and registers Prometheus metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotron_rpc_requests_total",
				Help: "Total number of RPC requests",
			},
			[]string{"blockchain", "method", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gotron_rpc_duration_seconds",
				Help:    "RPC request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"blockchain", "method"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gotron_rpc_in_flight",
				Help: "Number of RPC requests currently in progress",
			},
		),
		retriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotron_rpc_retries_total",
				Help: "Total number of RPC retries",
			},
			[]string{"blockchain", "method"},
		),
		poolTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gotron_rpc_pool_total",
				Help: "Total number of nodes in the pool",
			},
			[]string{"blockchain"},
		),
		poolHealthy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gotron_rpc_pool_healthy",
				Help: "Number of healthy nodes in the pool",
			},
			[]string{"blockchain"},
		),
		poolDisabled: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gotron_rpc_pool_disabled",
				Help: "Number of disabled nodes in the pool",
			},
			[]string{"blockchain"},
		),
	}

	reg.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.retriesTotal,
		m.poolTotal,
		m.poolHealthy,
		m.poolDisabled,
	)
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
func (m *Metrics) RecordRequest(blockchain, method, status string, duration time.Duration) {
	m.requestsTotal.WithLabelValues(blockchain, method, status).Inc()
	m.requestDuration.WithLabelValues(blockchain, method).Observe(duration.Seconds())
}

// RecordRetry records a retry attempt.
func (m *Metrics) RecordRetry(blockchain, method string) {
	m.retriesTotal.WithLabelValues(blockchain, method).Inc()
}

// SetPoolHealth updates node pool health metrics.
func (m *Metrics) SetPoolHealth(blockchain string, total, healthy, disabled int) {
	m.poolTotal.WithLabelValues(blockchain).Set(float64(total))
	m.poolHealthy.WithLabelValues(blockchain).Set(float64(healthy))
	m.poolDisabled.WithLabelValues(blockchain).Set(float64(disabled))
}
