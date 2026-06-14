// LightAI Go Agent - Execution plane running on GPU servers.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lightai-go/internal/agent/collector"
	"lightai-go/internal/agent/metrics"
	"lightai-go/internal/agent/register"
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


		// P0-011: Warn about default agent token in production.
		if cfg.AgentToken == "" || cfg.AgentToken == "lightai-agent-token-change-me" || cfg.AgentToken == "dev-agent-token" {
			log.Warn("using default agent token -- NOT safe for production",
				"agent_token", cfg.AgentToken,
				"help", "Set LIGHTAI_AGENT_TOKEN env var to a secure random value.",
			)
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

	// GPU collectors: prefer external command collectors (product path).
	if cfg.Collectors.GPUExternal.Enabled {
		timeout := cfg.Collectors.GPUExternal.Timeout
		if timeout == 0 {
			timeout = 5 * time.Second
		}
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
			log.Info("external gpu collector enabled",
				"collector", def.Name,
				"vendor", def.Vendor,
				"discover_cmd", def.DiscoverCmd,
				"metrics_cmd", def.MetricsCmd,
			)
		}
	}

	// Mock GPU for development/test.
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

	// Health/metrics HTTP server.
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})
	healthMux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	metricsAddr := fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port)
	healthSrv := &http.Server{
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

	log.Info("agent started", "agent_id", agentID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runAgentLoop(ctx, cfg, agentID, hostname, advertisedAddr, registry, st, snap)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("agent shutting down", "signal", sig.String())

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server forced to shutdown", "error", err)
	}

	log.Info("agent stopped")
}

func runAgentLoop(ctx context.Context, cfg *config.AgentConfig, agentID, hostname, advertisedAddr string, registry *collector.Registry, st *state.State, snap *metrics.Snapshot) {
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
		AdvertisedAddr: advertisedAddr,
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

	// Collect immediately after registration.
	collectAndReport(ctx, client, cfg, agentID, registry, snap)

	consecutiveFailures := 0
	lastHBFailLog := time.Time{} // rate-limit heartbeat failure logs

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			hbResp, err := register.SendHeartbeat(client, cfg.ServerURL, cfg.AgentToken, agentID, nodeID)
			if err != nil {
				consecutiveFailures++
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
				consecutiveFailures = 0
			}
		case <-collectTicker.C:
			collectAndReport(ctx, client, cfg, agentID, registry, snap)
			// P1-001: Data staleness — mark offline if no success for too long.
			if time.Since(snap.LastSuccessTime()) > 3*collectInterval {
				snap.SetOnline(false)
			}
		}
	}

}

func collectAndReport(ctx context.Context, client *http.Client, cfg *config.AgentConfig, agentID string, registry *collector.Registry, snap *metrics.Snapshot) {
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

	req, err := http.NewRequest("POST", cfg.ServerURL+"/api/agent/resources/report", bytes.NewReader(bodyBytes))
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
	if len(report.GPUResources) > 0 {
		snap.SetGPUResources(report.GPUResources)
	}
	if report.System != nil {
		snap.SetSystem(report.System)
	}
	snap.SetOnline(true)
}
