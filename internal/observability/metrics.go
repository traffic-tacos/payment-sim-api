package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestTotal     *prometheus.CounterVec
	WebhookDeliveryTotal *prometheus.CounterVec
	WebhookLatency       prometheus.Histogram
	IdempotencyHitsTotal prometheus.Counter
	ScenarioCounterTotal *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"route", "method", "status"},
		),
		HTTPRequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"route", "status"},
		),
		WebhookDeliveryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhook_delivery_total",
				Help: "Total number of webhook deliveries",
			},
			[]string{"result"}, // success, failure, timeout
		),
		WebhookLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "webhook_latency_seconds",
				Help:    "Latency of webhook deliveries in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		IdempotencyHitsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "idempotency_hits_total",
				Help: "Total number of idempotency hits",
			},
		),
		ScenarioCounterTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "scenario_counter_total",
				Help: "Total number of payment scenarios executed",
			},
			[]string{"scenario"}, // approve, fail, delay, random
		),
	}
}
