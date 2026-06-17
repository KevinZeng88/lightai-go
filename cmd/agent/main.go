// LightAI Go Agent - Execution plane running on GPU servers.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"lightai-go/internal/agent/collector"
	"lightai-go/internal/agent/metrics"
	"lightai-go/internal/agent/register"
	agentruntime "lightai-go/internal/agent/runtime"
	"lightai-go/internal/agent/state"
	"lightai-go/internal/common/config"
	"lightai-go/internal/common/log"
	"lightai-go/internal/common/types"
	"lightai-go/internal/common/version"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var configPath = flag.String("config", "", "path to config file (YAML)")
var showVersion = flag.Bool("version", false, "show version and exit")

func main() {
	flag.Parse()

	// --- Version mode (exits early) ---
	if *showVersion {
		fmt.Println(version.String())
		return
	}

	cfg, err := config.LoadAgentConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Determine hostname early (needed for identity and logging).
	hostname, _ := os.Hostname()

	// P1-010: logging.level is primary; fall back to top-level log_level.
	logLevel := cfg.Logging.Level
	if logLevel == "" {
		logLevel = cfg.LogLevel
	}

	log.Init(log.Config{
		Level:         logLevel,
		Format:        cfg.Logging.Format,
		Dir:           cfg.Logging.Dir,
		File:          cfg.Logging.File,
		Stdout:        cfg.Logging.Stdout,
		FileEnabled:   cfg.Logging.FileEnabled,
		Append:        cfg.Logging.Append,
		MaxSizeMB:     cfg.Logging.MaxSizeMB,
		MaxFiles:      cfg.Logging.MaxFiles,
		RetentionDays: cfg.Logging.RetentionDays,
	})

	agentID := cfg.AgentID
	if agentID == "" {
		agentID = uuid.NewString()
	}

	// Load or generate persistent node identity.
	identityDir := cfg.IdentityDir
	if identityDir == "" {
		identityDir = "runtime"
	}
	st, err := state.Load(identityDir, agentID, hostname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load agent identity: %v\n", err)
		os.Exit(1)
	}

	// Honour explicit node_id from config (validate against identity file).
	if cfg.NodeID != "" {
		if st.CachedNodeID() != cfg.NodeID {
			fmt.Fprintf(os.Stderr, "ERROR: config node_id=%s conflicts with identity file node_id=%s\n  Remove %s or set node_id in config to match.\n", cfg.NodeID, st.CachedNodeID(), st.Path())
			os.Exit(1)
		}
	}

	advertisedAddr := hostname
	if cfg.AdvertisedAddr != "" {
		advertisedAddr = cfg.AdvertisedAddr
	}

	// Compute primary IP: prefer config, then auto-detect first non-loopback IPv4.
	primaryIP := cfg.PrimaryIP
	if primaryIP == "" {
		primaryIP = detectPrimaryIP()
	}

	// Gather OS / arch / kernel info.
	osName := runtime.GOOS
	archName := runtime.GOARCH
	kernelVer := detectKernelVersion()

	// REVIEW-001: Refuse startup with default/empty agent token.
	if cfg.AgentToken == "" || cfg.AgentToken == "lightai-agent-token-change-me" || cfg.AgentToken == "dev-agent-token" {
		fmt.Fprintf(os.Stderr, "ERROR: Default or empty agent token detected.\n")
		fmt.Fprintf(os.Stderr, "Set LIGHTAI_AGENT_TOKEN to a secure random value.\n")
		fmt.Fprintf(os.Stderr, "Example: export LIGHTAI_AGENT_TOKEN=$(openssl rand -hex 32)\n")
		os.Exit(1)
	}
	// REVIEW-020: Warn about config fields that are documented but not yet implemented.
	if cfg.Collectors.ReportInterval != 0 && cfg.Collectors.ReportInterval != 5*time.Second {
		log.Warn("config: collectors.report_interval is set but not yet implemented — using default 5s",
			"configured", cfg.Collectors.ReportInterval.String())
	}
	if cfg.Metrics.AdvertiseAddr != "" {
		log.Warn("config: metrics.advertise_addr is set but not yet implemented — agent uses auto-detected address",
			"configured", cfg.Metrics.AdvertiseAddr)
	}

	log.Info("agent starting",
		"version", version.String(),
		"agent_id", agentID,
		"node_id", st.CachedNodeID(),
		"server_url", cfg.ServerURL,
		"hostname", hostname,
		"advertise_address", advertisedAddr,
		"metrics_listen", fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port),
		"identity_file", st.Path(),
		"gpu_profile", cfg.GPU.Profile,
		"log_level", cfg.LogLevel,
	)

	// Setup Prometheus metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	// Setup metrics snapshot (read by /metrics, never triggers collection).
	snap := metrics.NewSnapshot("", agentID, hostname)
	metrics.Register(reg, snap)

	// Setup collectors based on profile.
	registry := collector.NewRegistry()

	if cfg.Collectors.System.Enabled {
		registry.RegisterSystem(collector.NewSystemCollector())
	}

	profile := cfg.GPU.Profile
	if profile == "" {
		profile = "production"
	}

	collectorMode := cfg.GPU.CollectorMode
	if collectorMode == "" {
		collectorMode = "auto"
	}

	timeout := cfg.Collectors.GPUExternal.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	switch collectorMode {
	case "disabled":
		log.Info("gpu collector_mode=disabled, skipping all GPU collectors")

	case "explicit":
		// Explicit mode: only collectors explicitly enabled in config.
		for _, def := range cfg.Collectors.GPUExternal.Collectors {
			if !def.Enabled {
				continue
			}
			extCfg := collector.ExternalCommandConfig{
				Name:        def.Name,
				Vendor:      def.Vendor,
				Enabled:     def.Enabled,
				DiscoverCmd: def.DiscoverCmd,
				MetricsCmd:  def.MetricsCmd,
				Timeout:     timeout,
			}
			registry.RegisterGPU(collector.NewExternalCommandCollector(extCfg))
			log.Info("explicit gpu collector enabled",
				"collector", def.Name,
				"vendor", def.Vendor,
			)
		}

	default: // "auto"
		// Auto mode: probe each vendor, enable those with GPUs.
		probes := cfg.Collectors.GPUExternal.AutoDetect.Probes
		if len(probes) == 0 {
			// Use built-in defaults.
			for _, p := range collector.DefaultProbes() {
				probes = append(probes, config.ExternalCollectorDef{
					Name:        p.Name,
					Vendor:      p.Vendor,
					DiscoverCmd: p.DiscoverCmd,
					MetricsCmd:  p.MetricsCmd,
				})
			}
		}

		enabledVendors := make([]string, 0)
		ctx := context.Background()
		for _, def := range probes {
			probeDef := collector.ProbeDef{
				Name:        def.Name,
				Vendor:      def.Vendor,
				DiscoverCmd: def.DiscoverCmd,
				MetricsCmd:  def.MetricsCmd,
				Timeout:     timeout,
			}
			result := collector.Probe(ctx, probeDef)
			if result.Available {
				extCfg := collector.ExternalCommandConfig{
					Name:        def.Name,
					Vendor:      def.Vendor,
					Enabled:     true,
					DiscoverCmd: def.DiscoverCmd,
					MetricsCmd:  def.MetricsCmd,
					Timeout:     timeout,
				}
				registry.RegisterGPU(collector.NewExternalCommandCollector(extCfg))
				enabledVendors = append(enabledVendors, def.Vendor)
			}
		}
		log.Info("auto-detect GPU collectors complete",
			"mode", "auto",
			"enabled_vendors", enabledVendors,
		)
	}

	// Mock GPU for development/test (applies to all modes).
	if profile == "development" || profile == "test" {
		if cfg.Collectors.MockGPU.Enabled {
			registry.RegisterGPU(collector.NewMockGPUCollector())
			log.Info("mock gpu collector enabled", "profile", profile)
		}
	}

	log.Info("collectors configured",
		"system_enabled", cfg.Collectors.System.Enabled,
		"external_gpu_enabled", cfg.Collectors.GPUExternal.Enabled,
		"mock_enabled", cfg.Collectors.MockGPU.Enabled,
		"gpu_profile", profile,
		"heartbeat_interval_s", cfg.Heartbeat.Interval.Seconds(),
		"collect_interval_s", cfg.Collectors.System.Interval.Seconds(),
		"report_interval_s", cfg.Collectors.ReportInterval.Seconds(),
		"request_timeout_s", cfg.RequestTimeout.Seconds(),
	)

	// P1-003: Only start metrics HTTP server if enabled.
	var healthSrv *http.Server
	if cfg.Metrics.Enabled {
		healthMux := http.NewServeMux()
		healthMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
		})
		healthMux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		healthMux.HandleFunc("GET /docker-images", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			out, err := execCmd("docker", "images", "--format", "{{.Repository}}:{{.Tag}}\t{{.Size}}")
			if err != nil {
				json.NewEncoder(w).Encode([]map[string]interface{}{})
				return
			}
			lines := strings.Split(strings.TrimSpace(out), "\n")
			var images []map[string]interface{}
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "\t", 2)
				if strings.Contains(parts[0], "<none>") {
					continue
				}
				size := ""
				if len(parts) > 1 {
					size = parts[1]
				}
				images = append(images, map[string]interface{}{
					"image": parts[0],
					"size":  size,
				})
			}
			if images == nil {
				images = []map[string]interface{}{}
			}
			json.NewEncoder(w).Encode(images)
		})

		metricsAddr := fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port)
		healthSrv = &http.Server{
			Addr:         metricsAddr,
			Handler:      healthMux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Info("metrics server listening", "addr", metricsAddr)
			if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("metrics server failed", "error", err)
			}
		}()
	} else {
		log.Info("metrics server disabled (metrics.enabled=false)")
	}

	log.Info("agent started", "agent_id", agentID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runAgentLoop(ctx, cfg, agentID, hostname, primaryIP, advertisedAddr, osName, archName, kernelVer, registry, st, snap)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("agent shutting down", "signal", sig.String())

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if healthSrv != nil {
		if err := healthSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("metrics server forced to shutdown", "error", err)
		}
	}

	log.Info("agent stopped")
}

func runAgentLoop(ctx context.Context, cfg *config.AgentConfig, agentID, hostname, primaryIP, advertisedAddr, osName, archName, kernelVer string, registry *collector.Registry, st *state.State, snap *metrics.Snapshot) {
	timeout := cfg.RequestTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	// Register with server.
	regCfg := register.Config{
		ServerURL:      cfg.ServerURL,
		AgentToken:     cfg.AgentToken,
		AgentID:        agentID,
		Hostname:       hostname,
		PrimaryIP:      primaryIP,
		AdvertisedAddr: advertisedAddr,
		OS:             osName,
		Arch:           archName,
		Kernel:         kernelVer,
		MetricsEnabled: cfg.Metrics.Enabled,
		MetricsScheme:  cfg.Metrics.Scheme,
		MetricsPort:    cfg.Metrics.Port,
		MetricsPath:    cfg.Metrics.Path,
		Version:        version.String(),
		RequestTimeout: timeout,
	}

	// Initial registration with backoff.
	nodeID := st.CachedNodeID()
	regBackoff := 1 * time.Second
	maxRegBackoff := 60 * time.Second
	regFailures := 0
	for {
		var regErr error
		nodeID, regErr = register.Do(client, regCfg, st)
		if regErr == nil {
			log.Info("agent registered with server",
				"node_id", nodeID,
				"agent_id", agentID,
			)
			snap.SetNodeID(nodeID)
			break
		}
		regFailures++
		if regFailures == 1 {
			log.Error("initial registration failed", "error", regErr)
		} else if regFailures%10 == 0 {
			// Every ~10 failures, emit a warning to keep visibility.
			log.Warn("registration still failing",
				"failures", regFailures,
				"next_retry_s", regBackoff.Seconds(),
				"error", regErr,
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(regBackoff):
		}
		regBackoff *= 2
		if regBackoff > maxRegBackoff {
			regBackoff = maxRegBackoff
		}
	}

	// Heartbeat ticker.
	hbInterval := cfg.Heartbeat.Interval
	if hbInterval == 0 {
		hbInterval = 2 * time.Second
	}
	heartbeatTicker := time.NewTicker(hbInterval)
	defer heartbeatTicker.Stop()

	// Collection/report ticker.
	collectInterval := cfg.Collectors.System.Interval
	if collectInterval == 0 {
		collectInterval = 5 * time.Second
	}
	collectTicker := time.NewTicker(collectInterval)
	defer collectTicker.Stop()

	// Periodic summary trackers for high-frequency operations.
	hbSummary := log.NewPeriodicSummary("heartbeat", log.SummaryConfig.HeartbeatInterval)
	taskPollSummary := log.NewPeriodicSummary("task_poll", log.SummaryConfig.TaskPollInterval)
	gpuMetricsSummary := log.NewPeriodicSummary("gpu_metrics", log.SummaryConfig.MetricsInterval)
	hbFirstSuccess := false // track first heartbeat success after startup/failure

	// Collect immediately after registration.
	collectAndReport(ctx, client, cfg, agentID, registry, snap, gpuMetricsSummary)

	// Task processing: fire-and-forget goroutines per task with concurrency
	// limit so that slow tasks (Docker pull, stop) never block the heartbeat.
	maxTasks := cfg.Task.MaxConcurrentTasks
	if maxTasks <= 0 {
		maxTasks = 3
	}
	taskSem := make(chan struct{}, maxTasks)
	var taskWg sync.WaitGroup

	consecutiveFailures := 0
	lastHBFailLog := time.Time{} // rate-limit heartbeat failure logs

	// REVIEW-005: Periodic managed container reconciliation (every 60s).
	reconcileTicker := time.NewTicker(60 * time.Second)
	defer reconcileTicker.Stop()

	// Initial reconciliation at startup.
	go reconcileManagedContainers(ctx)

	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: wait for running tasks to finish (with timeout).
			log.Info("shutting down, waiting for running tasks to complete")
			taskDone := make(chan struct{})
			go func() {
				taskWg.Wait()
				close(taskDone)
			}()
			select {
			case <-taskDone:
				log.Info("all running tasks completed")
			case <-time.After(30 * time.Second):
				log.Warn("timed out waiting for running tasks to complete during shutdown")
			}
			return

		case <-heartbeatTicker.C:
			hbStart := time.Now()
			hbResp, err := register.SendHeartbeat(client, cfg.ServerURL, cfg.AgentToken, agentID, nodeID)
			hbLatency := time.Since(hbStart).Milliseconds()
			if err != nil {
				consecutiveFailures++
				hbSummary.RecordFailure()
				// Rate-limit: log at most once per 30 seconds for repeated heartbeat failures.
				if consecutiveFailures == 1 || time.Since(lastHBFailLog) > 30*time.Second {
					log.Warn("heartbeat failed",
						"agent_id", agentID,
						"node_id", nodeID,
						"error", err,
						"consecutive_failure_count", consecutiveFailures,
					)
					lastHBFailLog = time.Now()
				}
				if hbResp != nil && hbResp.NeedRegister {
					log.Info("server requested re-registration",
						"agent_id", agentID,
					)
					nodeID, regErr := register.Do(client, regCfg, st)
					if regErr != nil {
						log.Error("re-registration failed", "error", regErr)
					} else {
						log.Info("re-registration successful",
							"node_id", nodeID,
						)
						snap.SetNodeID(nodeID)
						consecutiveFailures = 0
					}
				}
			} else {
				if consecutiveFailures > 0 {
					// Heartbeat recovered from failure — log at INFO.
					log.Info("heartbeat recovered",
						"agent_id", agentID,
						"node_id", nodeID,
						"after_failures", consecutiveFailures,
					)
				}
				consecutiveFailures = 0
				hbSummary.RecordSuccess(hbLatency)

				// State change detection: log first success or state changes.
				if !hbFirstSuccess {
					hbFirstSuccess = true
					log.Info("heartbeat first success", "agent_id", agentID, "node_id", nodeID, "latency_ms", hbLatency)
				}

				// Periodic heartbeat summary.
				if ok, name, data := hbSummary.ShouldSummarize(); ok {
					log.Info("heartbeat.summary",
						"operation", name, "stage", "summary",
						"node_id", nodeID,
						"success_count", data["success_count"],
						"failure_count", data["failure_count"],
						"last_latency_ms", data["last_value"],
						"max_latency_ms", data["max_value"],
						"avg_latency_ms", data["avg_value"],
					)
				}

				// Fire-and-forget: spawn one goroutine per task so that slow
				// tasks never block the heartbeat loop.
				taskCount := 0
				if hbResp != nil {
					taskCount = len(hbResp.Tasks)
					for _, task := range hbResp.Tasks {
						task := task // capture loop variable
						// Acquire semaphore (respect shutdown).
						select {
						case taskSem <- struct{}{}:
						case <-ctx.Done():
							return
						}
						taskWg.Add(1)
						go func() {
							defer taskWg.Done()
							defer func() { <-taskSem }()
							processTask(ctx, client, cfg, agentID, task)
						}()
					}
				}

				// Task poll summary (combined with heartbeat response).
				if taskCount == 0 {
					taskPollSummary.RecordSuccess(0) // no_task
				} else {
					taskPollSummary.RecordSuccess(int64(taskCount)) // claimed
				}
				if ok, name, data := taskPollSummary.ShouldSummarize(); ok {
					log.Info("task_poll.summary",
						"operation", name, "stage", "summary",
						"node_id", nodeID,
						"no_task_count", data["success_count"],
						"claimed_count", data["last_value"],
						"failure_count", data["failure_count"],
						"total_count", data["total_count"],
					)
				}
			}

		case <-reconcileTicker.C:
			go reconcileManagedContainers(ctx)

		case <-collectTicker.C:
			collectAndReport(ctx, client, cfg, agentID, registry, snap, gpuMetricsSummary)
			// P1-001: Data staleness — mark offline if no success for too long.
			if time.Since(snap.LastSuccessTime()) > 3*collectInterval {
				snap.SetOnline(false)
			}
		}
	}

}

func collectAndReport(ctx context.Context, client *http.Client, cfg *config.AgentConfig, agentID string, registry *collector.Registry, snap *metrics.Snapshot, gpuMetricsSummary *log.PeriodicSummary) {
	log.Debug("resource collect start", "agent_id", agentID)

	start := time.Now()
	report := registry.Collect(ctx, agentID)
	if report == nil {
		log.Warn("resource collection returned nil report")
		// P0-008: Record collection failure.
		snap.IncCollectErrors()
		return
	}
	collectDuration := time.Since(start)

	// Marshal and send to server.
	bodyBytes, err := json.Marshal(report)
	if err != nil {
		log.Error("failed to marshal resource report", "error", err)
		snap.IncCollectErrors()
		return
	}

	req, err := http.NewRequest("POST", cfg.ServerURL+"/api/v1/agent/resources/report", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Error("failed to create resource report request", "error", err)
		snap.IncCollectErrors()
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AgentToken)

	reportStart := time.Now()
	resp, err := client.Do(req)
	reportLatency := time.Since(reportStart)
	if err != nil {
		log.Error("resource report failed",
			"agent_id", agentID,
			"error", err,
			"collect_ms", collectDuration.Milliseconds(),
		)
		// P0-008: Record report failure.
		snap.IncReportErrors()
		return
	}
	defer resp.Body.Close()

	// P0-008: Check HTTP status code.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error("resource report rejected by server",
			"agent_id", agentID,
			"http_status", resp.StatusCode,
			"collect_ms", collectDuration.Milliseconds(),
		)
		snap.IncReportErrors()
		return
	}

	// P0-008: Record success.
	snap.IncReportSuccess()

	// Update metrics snapshot from latest collection.
	updateSnapshot(snap, registry, agentID, report)

	// Record metrics summary.
	gpuMetricsSummary.RecordSuccess(collectDuration.Milliseconds())
	if ok, name, data := gpuMetricsSummary.ShouldSummarize(); ok {
		gpuCount := registry.GPUCount()
		log.Info("gpu_metrics.summary",
			"operation", name, "stage", "summary",
			"node_id", snap.NodeID,
			"gpu_count", gpuCount,
			"success_count", data["success_count"],
			"failure_count", data["failure_count"],
			"last_collect_ms", data["last_value"],
			"max_collect_ms", data["max_value"],
			"avg_collect_ms", data["avg_value"],
		)
	}

	gpuCount := registry.GPUCount()
	log.Debug("resource report success",
		"agent_id", agentID,
		"gpu_count", gpuCount,
		"collect_ms", collectDuration.Milliseconds(),
		"report_latency_ms", reportLatency.Milliseconds(),
		"payload_bytes", len(bodyBytes),
	)
}

// updateSnapshot copies latest collector results into the metrics snapshot.
func updateSnapshot(snap *metrics.Snapshot, registry *collector.Registry, agentID string, report *collector.ResourceReport) {
	if report == nil {
		return
	}
	if report.GPUResources != nil {
		snap.SetGPUResources(report.GPUResources)
	}
	if report.System != nil {
		snap.SetSystem(report.System)
		if len(report.GPUResources) == 0 {
			snap.SetGPUResources(report.GPUResources)
		}
	}
	snap.SetOnline(true)
}

// detectPrimaryIP finds the first non-loopback IPv4 address.
// Returns empty string if none is found.
func detectPrimaryIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP
		if ip.IsLoopback() || ip.To4() == nil {
			continue
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}
		return ip.String()
	}
	return ""
}

// detectKernelVersion returns the kernel version using syscall.Uname.
func detectKernelVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return ""
	}
	// Convert [65]int8 to string.
	b := make([]byte, 0, len(uname.Release))
	for _, c := range uname.Release {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}

// processTask handles a single agent task received via heartbeat.
// It is called from a fire-and-forget goroutine with concurrency limited by
// taskSem so that slow tasks never block the heartbeat loop.
func processTask(ctx context.Context, client *http.Client, cfg *config.AgentConfig, agentID string, task register.AgentTask) {
	startTime := time.Now()
	// Parse AgentRunSpec to extract operation_id
	var spec agentruntime.AgentRunSpec
	opID := ""
	if err := json.Unmarshal(task.AgentRunSpec, &spec); err == nil {
		opID = spec.OperationID
	}
	if opID == "" {
		opID = task.ID
	} // fallback
	log.Info("task execution: begin",
		"task_id", task.ID,
		"task_type", task.TaskType,
		"instance_id", task.InstanceID,
		"operation_id", opID,
	)

	result := register.TaskResult{
		TaskID:      task.ID,
		OperationID: opID,
		InstanceID:  task.InstanceID,
	}

	startTime = time.Now()
	result.StartedAt = startTime.Format(time.RFC3339)

	switch task.TaskType {
	case "model_instance_start":
		processStartTask(ctx, task, &result)
	case "model_instance_stop":
		processStopTask(ctx, task, &result)
	case "model_instance_logs":
		processLogsTask(ctx, task, &result)
	default:
		result.Success = false
		result.ErrorMessage = "unknown task type: " + task.TaskType
	}

	result.FinishedAt = time.Now().Format(time.RFC3339)

	if result.Success {
		log.Info("task execution: completed",
			"task_id", task.ID,
			"operation_id", opID,
			"instance_id", task.InstanceID,
			"container_id", result.ContainerID,
			"runtime_state", result.RuntimeState,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
	} else {
		log.Error("task execution: failed",
			"task_id", task.ID,
			"operation_id", opID,
			"instance_id", task.InstanceID,
			"error", result.ErrorMessage,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
	}

	// Report result back to server.
	reportStart := time.Now()
	log.Debug("task_result.report_started", "task_id", task.ID, "operation_id", opID, "instance_id", task.InstanceID)
	if err := register.ReportTaskResult(client, cfg.ServerURL, cfg.AgentToken, task.ID, result); err != nil {
		log.Error("task_result.report_failed",
			"task_id", task.ID,
			"operation_id", opID,
			"instance_id", task.InstanceID,
			"error", err,
			"duration_ms", time.Since(reportStart).Milliseconds(),
		)
	} else {
		log.Info("task_result.report_completed",
			"task_id", task.ID,
			"operation_id", opID,
			"instance_id", task.InstanceID,
			"status", result.RuntimeState,
			"success", result.Success,
			"duration_ms", time.Since(reportStart).Milliseconds(),
		)
	}
}

func processStartTask(ctx context.Context, task register.AgentTask, result *register.TaskResult) {
	startTime := time.Now()
	log.Info("start task: begin", "task_id", task.ID, "instance_id", task.InstanceID)
	// Parse the AgentRunSpec from task payload.
	var spec agentruntime.AgentRunSpec
	if err := json.Unmarshal(task.AgentRunSpec, &spec); err != nil {
		log.Error("task payload: parse failed", "task_id", task.ID, "error", err)
		result.Success = false
		result.ErrorMessage = "invalid agent run spec: " + err.Error()
		log.Error("start task: invalid payload", "task_id", task.ID, "error", err)
		return
	}

	// Create real Docker client and driver.
	log.Info("start task: creating docker client", "task_id", task.ID)
	realCli, err := agentruntime.NewRealDockerClient()
	// Docker client created successfully
	if err != nil {
		result.Success = false
		result.ErrorMessage = "cannot create docker client: " + err.Error()
		log.Error("start task: docker client failed", "task_id", task.ID, "error", err)
		return
	}
	defer realCli.Close()

	// Check Docker daemon availability.
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if _, err := realCli.Ping(pingCtx); err != nil {
		result.Success = false
		result.ErrorMessage = "docker daemon unavailable: " + err.Error()
		return
	}

	driver := agentruntime.NewDockerRuntimeDriver(realCli)

	// Apply per-task timeout from the task's timeout_seconds.
	taskCtx, taskCancel := context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
	defer taskCancel()
	inst, err := driver.Start(taskCtx, spec)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		result.ExitCode = -1
		return
	}

	log.Info("start task: completed", "task_id", task.ID, "instance_id", task.InstanceID, "duration_ms", time.Since(startTime).Milliseconds())
	result.Success = true
	result.ContainerID = inst.ContainerID
	result.RuntimeState = "running"
	result.InstanceID = inst.InstanceID
	result.DeploymentID = task.DeploymentID
	result.NodeID = task.NodeID
}

func processStopTask(ctx context.Context, task register.AgentTask, result *register.TaskResult) {
	// Parse payload to get instance ID and container ID.
	var payload struct {
		InstanceID    string `json:"instance_id"`
		ContainerID   string `json:"container_id"`
		ContainerName string `json:"container_name,omitempty"`
	}
	if err := json.Unmarshal(task.AgentRunSpec, &payload); err != nil {
		result.Success = false
		result.ErrorMessage = "invalid stop payload: " + err.Error()
		return
	}

	realCli, err := agentruntime.NewRealDockerClient()
	// Docker client created successfully
	if err != nil {
		result.Success = false
		result.ErrorMessage = "cannot create docker client: " + err.Error()
		log.Error("start task: docker client failed", "task_id", task.ID, "error", err)
		return
	}
	defer realCli.Close()

	driver := agentruntime.NewDockerRuntimeDriver(realCli)

	taskCtx, taskCancel := context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
	defer taskCancel()
	if err := driver.Stop(taskCtx, payload.InstanceID); err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		result.ExitCode = -1
		return
	}

	result.Success = true
	result.RuntimeState = "stopped"
	result.InstanceID = payload.InstanceID
	result.ContainerID = payload.ContainerID
	result.DeploymentID = task.DeploymentID
	result.NodeID = task.NodeID
}

func processLogsTask(ctx context.Context, task register.AgentTask, result *register.TaskResult) {
	var payload struct {
		InstanceID    string `json:"instance_id"`
		ContainerID   string `json:"container_id"`
		ContainerName string `json:"container_name,omitempty"`
	}
	if err := json.Unmarshal(task.AgentRunSpec, &payload); err != nil {
		result.Success = false
		result.ErrorMessage = "invalid logs payload: " + err.Error()
		return
	}

	log.Info("processing logs task",
		"task_id", task.ID,
		"instance_id", payload.InstanceID,
		"container_id", payload.ContainerID,
	)

	realCli, err := agentruntime.NewRealDockerClient()
	// Docker client created successfully
	if err != nil {
		result.Success = false
		result.ErrorMessage = "cannot create docker client: " + err.Error()
		log.Error("start task: docker client failed", "task_id", task.ID, "error", err)
		return
	}
	defer realCli.Close()

	driver := agentruntime.NewDockerRuntimeDriver(realCli)

	// Use container_id or container_name to locate; fall back to instance-derived name.
	targetID := payload.ContainerID
	if targetID == "" {
		targetID = payload.ContainerName
	}
	if targetID == "" {
		targetID = payload.InstanceID
	}

	logs, err := driver.Logs(ctx, targetID, agentruntime.LogOptions{Tail: 100})
	if err != nil {
		log.Error("logs task failed",
			"task_id", task.ID,
			"instance_id", payload.InstanceID,
			"target_id", targetID,
			"error", err,
		)
		result.Success = false
		result.ErrorMessage = err.Error()
		return
	}

	log.Info("logs task completed",
		"task_id", task.ID,
		"instance_id", payload.InstanceID,
		"bytes", len(logs.Stdout),
	)

	result.Success = true
	result.RuntimeState = "ok"
	result.LogsSummary = logs.Stdout
	result.InstanceID = payload.InstanceID
	result.ContainerID = payload.ContainerID
	result.DeploymentID = task.DeploymentID
	result.NodeID = task.NodeID
}

// reconcileManagedContainers lists Docker containers with the LightAI naming prefix
// and logs discrepancies against what the agent expects. REVIEW-005: Agent reconciliation.
func reconcileManagedContainers(ctx context.Context) {
	out, err := execCmd("docker", "ps", "-a", "--format", "{{.Names}}\t{{.Status}}", "--filter", "name=lightai-")
	if err != nil {
		log.Debug("reconcile: docker ps failed (may be normal if Docker not available)", "error", err)
		return
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	managed := 0
	exited := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		managed++
		if strings.Contains(line, "Exited") {
			exited++
		}
	}
	if managed > 0 {
		log.Info("reconcile: managed containers found",
			"total", managed, "exited", exited, "running", managed-exited)
	} else {
		log.Debug("reconcile: no managed containers found")
	}
}

// execCmd runs a command and returns its stdout as a string.
func execCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
