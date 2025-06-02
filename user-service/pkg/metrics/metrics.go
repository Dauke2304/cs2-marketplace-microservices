package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// gRPC Request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_service_requests_total",
			Help: "Total number of requests processed by user service",
		},
		[]string{"method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "user_service_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// User management metrics
	UsersRegistered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_service_users_registered_total",
			Help: "Total number of users registered",
		},
	)

	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "user_service_active_sessions",
			Help: "Number of active user sessions",
		},
	)

	LoginAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_service_login_attempts_total",
			Help: "Total number of login attempts",
		},
		[]string{"status"}, // success, failed
	)

	// Email metrics
	EmailsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_service_emails_sent_total",
			Help: "Total number of emails sent",
		},
		[]string{"type"}, // reset_password, verification, etc.
	)

	// Database metrics
	DatabaseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "user_service_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	// Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_service_errors_total",
			Help: "Total number of errors in user service",
		},
		[]string{"type", "method"},
	)

	// NATS messaging metrics
	MessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_service_messages_published_total",
			Help: "Total number of messages published to NATS",
		},
		[]string{"subject"},
	)

	// Service health
	ServiceUp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "user_service_up",
			Help: "Whether the user service is up (1) or down (0)",
		},
	)
)

func init() {
	ServiceUp.Set(1)
}
