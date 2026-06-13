// Package metrics provides LightAI Server custom Prometheus metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ServerMetrics holds all LightAI Server Prometheus metrics.
type ServerMetrics struct {
	Info               prometheus.Gauge
	AgentsTotal        prometheus.Gauge
	NodesTotal         prometheus.Gauge
	NodesOnline        prometheus.Gauge
	GPUsTotal          prometheus.Gauge
	GPUsAvailable      prometheus.Gauge
	GPUsHealthy        prometheus.Gauge
	APIRequests        *prometheus.CounterVec
	APIRequestDuration *prometheus.HistogramVec
	AgentHeartbeats    prometheus.Counter
	AgentReports       prometheus.Counter
	AuthLoginTotal     prometheus.Counter
	AuthLoginFailed    prometheus.Counter
}

// New creates and registers all Server metrics.
func New(reg *prometheus.Registry) *ServerMetrics {
	f := promauto.With(reg)

	m := &ServerMetrics{
		Info: f.NewGauge(prometheus.GaugeOpts{
			Name:        "lightai_server_info",
			Help:        "LightAI Server information (always 1).",
			ConstLabels: prometheus.Labels{"version": "0.1.0"},
		}),
		AgentsTotal: f.NewGauge(prometheus.GaugeOpts{
			Name: "lightai_server_agents_total",
			Help: "Total number of registered agents.",
		}),
		NodesTotal: f.NewGauge(prometheus.GaugeOpts{
			Name: "lightai_server_nodes_total",
			Help: "Total number of nodes.",
		}),
		NodesOnline: f.NewGauge(prometheus.GaugeOpts{
			Name: "lightai_server_nodes_online",
			Help: "Number of online nodes.",
		}),
		GPUsTotal: f.NewGauge(prometheus.GaugeOpts{
			Name: "lightai_server_gpus_total",
			Help: "Total number of GPUs.",
		}),
		GPUsAvailable: f.NewGauge(prometheus.GaugeOpts{
			Name: "lightai_server_gpus_available",
			Help: "Number of available GPUs.",
		}),
		GPUsHealthy: f.NewGauge(prometheus.GaugeOpts{
			Name: "lightai_server_gpus_healthy",
			Help: "Number of healthy GPUs.",
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

	m.Info.Set(1)
	return m
}
