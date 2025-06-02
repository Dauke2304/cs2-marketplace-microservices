package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP/gRPC Request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_service_requests_total",
			Help: "Total number of requests processed by inventory service",
		},
		[]string{"method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "inventory_service_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Business logic metrics
	ItemsInInventory = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "inventory_service_items_total",
			Help: "Total number of items in inventory",
		},
	)

	DatabaseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "inventory_service_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	// Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_service_errors_total",
			Help: "Total number of errors in inventory service",
		},
		[]string{"type", "method"},
	)

	// NATS messaging metrics
	MessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_service_messages_published_total",
			Help: "Total number of messages published to NATS",
		},
		[]string{"subject"},
	)

	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_service_messages_received_total",
			Help: "Total number of messages received from NATS",
		},
		[]string{"subject"},
	)

	// Service health
	ServiceUp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "inventory_service_up",
			Help: "Whether the inventory service is up (1) or down (0)",
		},
	)
)

func init() {
	// Set service as up when metrics package is initialized
	ServiceUp.Set(1)
}
