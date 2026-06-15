package collector

import (
	"context"
	"time"
)

// Registry manages all collectors and executes collection cycles.
type Registry struct {
	systemCollectors []SystemCollector
	gpuCollectors    []GPUCollector
	lastSystem       *SystemSnapshot
	lastGPUDevices   []GPUDeviceInfo
	lastGPUMetrics   []GPUMetricInfo
	lastGPUResources []GPUResource
	lastDiagnostics  []CollectorDiagnosis
}

// NewRegistry creates a new collector registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterSystem registers a system collector.
func (r *Registry) RegisterSystem(c SystemCollector) {
	r.systemCollectors = append(r.systemCollectors, c)
}

// RegisterGPU registers a GPU collector.
func (r *Registry) RegisterGPU(c GPUCollector) {
	r.gpuCollectors = append(r.gpuCollectors, c)
}

// Collect runs all collectors and returns a resource report.
// On collector failure, retains the previous successful state.
func (r *Registry) Collect(ctx context.Context, agentID string) *ResourceReport {
	collectedAt := time.Now()
	report := &ResourceReport{
		AgentID:     agentID,
		CollectedAt: collectedAt,
	}

	// Collect system.
	var sysSnapshot *SystemSnapshot
	var sysDiags []CollectorDiagnosis
	for _, c := range r.systemCollectors {
		snapshot, diag := c.Collect(ctx)
		sysDiags = append(sysDiags, *diag)
		if snapshot != nil {
			sysSnapshot = snapshot
		}
	}
	if sysSnapshot != nil {
		r.lastSystem = sysSnapshot
		report.System = sysSnapshot
	} else {
		// Retain previous successful state.
		report.System = r.lastSystem
	}

	// Collect GPU devices and metrics.
	var gpuDevices []GPUDeviceInfo
	var gpuMetrics []GPUMetricInfo
	for _, c := range r.gpuCollectors {
		devices, diag := c.Discover(ctx)
		sysDiags = append(sysDiags, *diag)
		if diag.Available {
			if devices != nil {
				gpuDevices = append(gpuDevices, devices...)
			}
			metrics, mDiag := c.Metrics(ctx)
			sysDiags = append(sysDiags, *mDiag)
			if metrics != nil {
				gpuMetrics = append(gpuMetrics, metrics...)
			}
		}
	}
	if len(gpuDevices) > 0 {
		r.lastGPUDevices = gpuDevices
		report.GPUDevices = gpuDevices
	} else {
		report.GPUDevices = r.lastGPUDevices
	}
	if len(gpuMetrics) > 0 {
		r.lastGPUMetrics = gpuMetrics
		report.GPUMetrics = gpuMetrics
	} else {
		report.GPUMetrics = r.lastGPUMetrics
	}

	// Normalize into unified GPUResource slice.
	report.GPUResources = NormalizeGPUs(report.GPUDevices, report.GPUMetrics)
	if len(report.GPUResources) > 0 {
		r.lastGPUResources = report.GPUResources
	} else if len(r.lastGPUResources) > 0 {
		report.GPUResources = r.lastGPUResources
	}

	// Copy device names into metrics by UUID (backward compat for Prometheus labels).
	r.PairMetricsWithDevices()

	report.Diagnostics = sysDiags
	r.lastDiagnostics = sysDiags

	return report
}

// PairMetricsWithDevices copies device names into metrics by UUID matching.
func (r *Registry) PairMetricsWithDevices() {
	if len(r.lastGPUDevices) == 0 || len(r.lastGPUMetrics) == 0 {
		return
	}
	nameMap := make(map[string]string)
	for _, d := range r.lastGPUDevices {
		nameMap[d.UUID] = d.Name
	}
	for i := range r.lastGPUMetrics {
		if name, ok := nameMap[r.lastGPUMetrics[i].UUID]; ok {
			r.lastGPUMetrics[i].Name = name
		}
	}
}

// GPUCount returns the total number of GPU devices from the last collection.
func (r *Registry) GPUCount() int {
	return len(r.lastGPUDevices)
}

// LastDiagnostics returns the last diagnostics.
func (r *Registry) LastDiagnostics() []CollectorDiagnosis {
	return r.lastDiagnostics
}
