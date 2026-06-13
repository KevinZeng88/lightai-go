// Package metrics provides LightAI custom Prometheus metrics for Agent.
// /metrics reads from the latest snapshot only — never triggers real collection.
package metrics

import (
	"sync"
	"time"

	"lightai-go/internal/agent/collector"

	"github.com/prometheus/client_golang/prometheus"
)

// Snapshot holds the latest collected data for /metrics scraping.
type Snapshot struct {
	mu sync.RWMutex

	GPUMetrics  []collector.GPUMetricInfo
	GPUDevices  []collector.GPUDeviceInfo
	System      *collector.SystemSnapshot
	Diagnostics []collector.CollectorDiagnosis

	CollectErrors int64
	ReportSuccess int64
	ReportErrors  int64
	LastSuccessAt time.Time
	NodeOnline    int // 1 = online

	// Agent context for metric labels.
	NodeID   string
	AgentID  string
	Hostname string
}

// NewSnapshot creates a new metrics snapshot.
func NewSnapshot(nodeID, agentID, hostname string) *Snapshot {
	return &Snapshot{NodeID: nodeID, AgentID: agentID, Hostname: hostname}
}

func (s *Snapshot) SetGPUMetrics(m []collector.GPUMetricInfo) {
	s.mu.Lock()
	s.GPUMetrics = m
	s.LastSuccessAt = time.Now()
	s.mu.Unlock()
}

func (s *Snapshot) SetGPUDevices(d []collector.GPUDeviceInfo) {
	s.mu.Lock()
	s.GPUDevices = d
	s.mu.Unlock()
}

func (s *Snapshot) SetSystem(sys *collector.SystemSnapshot) {
	s.mu.Lock()
	s.System = sys
	s.mu.Unlock()
}

func (s *Snapshot) IncCollectErrors() {
	s.mu.Lock()
	s.CollectErrors++
	s.mu.Unlock()
}

func (s *Snapshot) IncReportSuccess() {
	s.mu.Lock()
	s.ReportSuccess++
	s.mu.Unlock()
}

func (s *Snapshot) IncReportErrors() {
	s.mu.Lock()
	s.ReportErrors++
	s.mu.Unlock()
}

func (s *Snapshot) SetOnline(v bool) {
	s.mu.Lock()
	if v {
		s.NodeOnline = 1
	} else {
		s.NodeOnline = 0
	}
	s.mu.Unlock()
}

// SetNodeID updates the node_id in the snapshot (called after registration).
func (s *Snapshot) SetNodeID(nodeID string) {
	s.mu.Lock()
	s.NodeID = nodeID
	s.mu.Unlock()
}

// Register registers all LightAI Agent custom metrics on the given registry.
func Register(reg *prometheus.Registry, snap *Snapshot) {
	reg.MustRegister(newGPUCollector(snap))
	reg.MustRegister(newAgentCollector(snap))
}

// --- GPU metrics collector ---

type gpuCollector struct {
	snap *Snapshot

	memTotalDesc *prometheus.Desc
	memUsedDesc  *prometheus.Desc
	memFreeDesc  *prometheus.Desc
	gpuUtilDesc  *prometheus.Desc
	memUtilDesc  *prometheus.Desc
	tempDesc     *prometheus.Desc
	powerDesc    *prometheus.Desc
	healthDesc   *prometheus.Desc
	statusDesc   *prometheus.Desc
}

func newGPUCollector(snap *Snapshot) *gpuCollector {
	return &gpuCollector{
		snap: snap,
		memTotalDesc: prometheus.NewDesc(
			"lightai_gpu_memory_total_bytes",
			"Total GPU memory in bytes.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		memUsedDesc: prometheus.NewDesc(
			"lightai_gpu_memory_used_bytes",
			"Used GPU memory in bytes.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		memFreeDesc: prometheus.NewDesc(
			"lightai_gpu_memory_free_bytes",
			"Free GPU memory in bytes.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		gpuUtilDesc: prometheus.NewDesc(
			"lightai_gpu_utilization_percent",
			"GPU utilization percent (0-100).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		memUtilDesc: prometheus.NewDesc(
			"lightai_gpu_memory_utilization_percent",
			"GPU memory utilization percent (0-100).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		tempDesc: prometheus.NewDesc(
			"lightai_gpu_temperature_celsius",
			"GPU temperature in Celsius.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		powerDesc: prometheus.NewDesc(
			"lightai_gpu_power_draw_watts",
			"GPU power draw in Watts.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		healthDesc: prometheus.NewDesc(
			"lightai_gpu_health_status",
			"GPU health status (1=healthy, 0=not healthy).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		statusDesc: prometheus.NewDesc(
			"lightai_gpu_available_status",
			"GPU available status (1=available, 0=not available).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
	}
}

func (c *gpuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memTotalDesc
	ch <- c.memUsedDesc
	ch <- c.memFreeDesc
	ch <- c.gpuUtilDesc
	ch <- c.memUtilDesc
	ch <- c.tempDesc
	ch <- c.powerDesc
	ch <- c.healthDesc
	ch <- c.statusDesc
}

func (c *gpuCollector) Collect(ch chan<- prometheus.Metric) {
	c.snap.mu.RLock()
	defer c.snap.mu.RUnlock()

	for _, m := range c.snap.GPUMetrics {
		lbls := []string{m.Vendor, m.UUID, itoa(m.Index), m.Name, c.snap.NodeID, c.snap.AgentID, c.snap.Hostname}

		ch <- prometheus.MustNewConstMetric(c.memTotalDesc, prometheus.GaugeValue, float64(m.MemoryTotalBytes), lbls...)
		ch <- prometheus.MustNewConstMetric(c.memUsedDesc, prometheus.GaugeValue, float64(m.MemoryUsedBytes), lbls...)
		ch <- prometheus.MustNewConstMetric(c.memFreeDesc, prometheus.GaugeValue, float64(m.MemoryFreeBytes), lbls...)

		if m.GPUUtilization != nil {
			ch <- prometheus.MustNewConstMetric(c.gpuUtilDesc, prometheus.GaugeValue, *m.GPUUtilization, lbls...)
		}
		if m.MemoryUtilization != nil {
			ch <- prometheus.MustNewConstMetric(c.memUtilDesc, prometheus.GaugeValue, *m.MemoryUtilization, lbls...)
		}
		if m.Temperature != nil {
			ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, *m.Temperature, lbls...)
		}
		if m.PowerDraw != nil {
			ch <- prometheus.MustNewConstMetric(c.powerDesc, prometheus.GaugeValue, *m.PowerDraw, lbls...)
		}

		healthVal := 0.0
		if m.Health == "healthy" {
			healthVal = 1.0
		}
		ch <- prometheus.MustNewConstMetric(c.healthDesc, prometheus.GaugeValue, healthVal, lbls...)
	}
}

// --- Agent collector metrics ---

type agentCollector struct {
	snap *Snapshot

	lastSuccessDesc   *prometheus.Desc
	errorsDesc        *prometheus.Desc
	reportSuccessDesc *prometheus.Desc
	reportErrorsDesc  *prometheus.Desc
	nodeOnlineDesc    *prometheus.Desc
}

func newAgentCollector(snap *Snapshot) *agentCollector {
	return &agentCollector{
		snap: snap,
		lastSuccessDesc: prometheus.NewDesc(
			"lightai_agent_collector_last_success_timestamp_seconds",
			"Unix timestamp of last successful collector run.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		errorsDesc: prometheus.NewDesc(
			"lightai_agent_collector_errors_total",
			"Total number of collector errors.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		reportSuccessDesc: prometheus.NewDesc(
			"lightai_agent_report_success_total",
			"Total number of successful resource reports.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		reportErrorsDesc: prometheus.NewDesc(
			"lightai_agent_report_errors_total",
			"Total number of resource report errors.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		nodeOnlineDesc: prometheus.NewDesc(
			"lightai_node_online",
			"Node online status (1=online, 0=offline).",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
	}
}

func (c *agentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.lastSuccessDesc
	ch <- c.errorsDesc
	ch <- c.reportSuccessDesc
	ch <- c.reportErrorsDesc
	ch <- c.nodeOnlineDesc
}

func (c *agentCollector) Collect(ch chan<- prometheus.Metric) {
	c.snap.mu.RLock()
	defer c.snap.mu.RUnlock()

	lbls := []string{c.snap.NodeID, c.snap.AgentID, c.snap.Hostname}
	ts := float64(c.snap.LastSuccessAt.Unix())
	if ts > 0 {
		ch <- prometheus.MustNewConstMetric(c.lastSuccessDesc, prometheus.GaugeValue, ts, lbls...)
	}
	ch <- prometheus.MustNewConstMetric(c.errorsDesc, prometheus.CounterValue, float64(c.snap.CollectErrors), lbls...)
	ch <- prometheus.MustNewConstMetric(c.reportSuccessDesc, prometheus.CounterValue, float64(c.snap.ReportSuccess), lbls...)
	ch <- prometheus.MustNewConstMetric(c.reportErrorsDesc, prometheus.CounterValue, float64(c.snap.ReportErrors), lbls...)
	ch <- prometheus.MustNewConstMetric(c.nodeOnlineDesc, prometheus.GaugeValue, float64(c.snap.NodeOnline), lbls...)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	n := i
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
