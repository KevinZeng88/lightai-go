// Package metrics provides LightAI Server custom Prometheus metrics.
package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ServerMetrics holds all LightAI Server Prometheus metrics.
type ServerMetrics struct {
	Info prometheus.Gauge

	// Gauges refreshed from DB on each scrape.
	NodesTotal    prometheus.GaugeFunc
	NodesOnline   prometheus.GaugeFunc
	GPUsTotal     prometheus.GaugeFunc
	GPUsAvailable prometheus.GaugeFunc
	GPUsHealthy   prometheus.GaugeFunc

	APIRequests        *prometheus.CounterVec
	APIRequestDuration *prometheus.HistogramVec
	AgentHeartbeats    prometheus.Counter
	AgentReports       prometheus.Counter
	AuthLoginTotal     prometheus.Counter
	AuthLoginFailed    prometheus.Counter
}

// New creates and registers all Server metrics.
// db may be nil if gauges should return 0 (e.g., in tests).
func New(reg *prometheus.Registry, db *sql.DB) *ServerMetrics {
	f := promauto.With(reg)

	m := &ServerMetrics{
		Info: f.NewGauge(prometheus.GaugeOpts{
			Name:        "lightai_server_info",
			Help:        "LightAI Server information (always 1).",
			ConstLabels: prometheus.Labels{"version": "0.1.0"},
		}),
		APIRequests: f.NewCounterVec(prometheus.CounterOpts{
			Name: "lightai_server_api_requests_total",
			Help: "Total API requests.",
		}, []string{"endpoint", "method", "code"}),
		APIRequestDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lightai_server_api_request_duration_seconds",
			Help:    "API request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"endpoint", "method"}),
		AgentHeartbeats: f.NewCounter(prometheus.CounterOpts{
			Name: "lightai_server_agent_heartbeats_total",
			Help: "Total agent heartbeat requests.",
		}),
		AgentReports: f.NewCounter(prometheus.CounterOpts{
			Name: "lightai_server_agent_reports_total",
			Help: "Total agent resource report requests.",
		}),
		AuthLoginTotal: f.NewCounter(prometheus.CounterOpts{
			Name: "lightai_server_auth_login_total",
			Help: "Total successful login attempts.",
		}),
		AuthLoginFailed: f.NewCounter(prometheus.CounterOpts{
			Name: "lightai_server_auth_login_failed_total",
			Help: "Total failed login attempts.",
		}),
	}

	// Register GaugeFunc metrics that read from DB on each scrape.
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "lightai_server_nodes_total",
			Help: "Total number of nodes.",
		},
		func() float64 {
			if db == nil {
				return 0
			}
			var count int
			db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&count)
			return float64(count)
		},
	))
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "lightai_server_nodes_online",
			Help: "Number of online nodes.",
		},
		func() float64 {
			if db == nil {
				return 0
			}
			var count int
			db.QueryRow("SELECT COUNT(*) FROM nodes WHERE status = 'online'").Scan(&count)
			return float64(count)
		},
	))
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "lightai_server_gpus_total",
			Help: "Total number of GPUs.",
		},
		func() float64 {
			if db == nil {
				return 0
			}
			var count int
			db.QueryRow("SELECT COUNT(*) FROM gpu_devices").Scan(&count)
			return float64(count)
		},
	))
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "lightai_server_gpus_available",
			Help: "Number of available GPUs.",
		},
		func() float64 {
			if db == nil {
				return 0
			}
			var count int
			db.QueryRow("SELECT COUNT(*) FROM gpu_devices WHERE status = 'available'").Scan(&count)
			return float64(count)
		},
	))
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "lightai_server_gpus_healthy",
			Help: "Number of healthy GPUs.",
		},
		func() float64 {
			if db == nil {
				return 0
			}
			var count int
			db.QueryRow("SELECT COUNT(*) FROM gpu_devices WHERE health = 'healthy'").Scan(&count)
			return float64(count)
		},
	))

	// Legacy gauges for code compatibility.
	m.NodesTotal = nil // Using GaugeFunc above
	m.NodesOnline = nil
	m.GPUsTotal = nil
	m.GPUsAvailable = nil
	m.GPUsHealthy = nil

	m.Info.Set(1)
	return m
}
