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

func main() {
	flag.Parse()

	cfg, err := config.LoadAgentConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Init file logging if configured.
	initLogging(cfg)

	log.Init(cfg.LogLevel)

	agentID := cfg.AgentID
	if agentID == "" {
		agentID = uuid.NewString()
	}

	// Load persistent state (node_id cache).
	st, err := state.Load(cfg.DataDir, agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load agent state: %v\n", err)
		os.Exit(1)
	}

	hostname, _ := os.Hostname()
	advertisedAddr := hostname
	if cfg.AdvertisedAddr != "" {
		advertisedAddr = cfg.AdvertisedAddr
	}

	log.Info("agent starting",
		"version", version.String(),
		"agent_id", agentID,
		"server_url", cfg.ServerURL,
		"hostname", hostname,
		"advertise_address", advertisedAddr,
		"metrics_listen", fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port),
		"data_dir", cfg.DataDir,
		"cached_node_id", st.CachedNodeID(),
		"gpu_profile", cfg.GPU.Profile,
		"log_level", cfg.LogLevel,
	)

	// Setup Prometheus metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

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

	go runAgentLoop(ctx, cfg, agentID, hostname, advertisedAddr, registry, st)

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

func runAgentLoop(ctx context.Context, cfg *config.AgentConfig, agentID, hostname, advertisedAddr string, registry *collector.Registry, st *state.State) {
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

	nodeID, err := register.Do(client, regCfg, st)
	if err != nil {
		log.Error("initial registration failed", "error", err)
		// Continue with heartbeat loop - will retry on next heartbeat failure.
	} else {
		log.Info("agent registered with server",
			"node_id", nodeID,
			"agent_id", agentID,
		)
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
	collectAndReport(ctx, client, cfg, agentID, registry)

	consecutiveFailures := 0
	lastSuccessAt := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			hbResp, err := register.SendHeartbeat(client, cfg.ServerURL, cfg.AgentToken, agentID)
			if err != nil {
				consecutiveFailures++
				log.Warn("heartbeat failed",
					"agent_id", agentID,
					"error", err,
					"consecutive_failure_count", consecutiveFailures,
				)
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
						consecutiveFailures = 0
					}
				}
			} else {
				consecutiveFailures = 0
			}
		case <-collectTicker.C:
			collectAndReport(ctx, client, cfg, agentID, registry)
			if consecutiveFailures == 0 {
				lastSuccessAt = time.Now()
			}
		}
	}

	_ = lastSuccessAt
}

func collectAndReport(ctx context.Context, client *http.Client, cfg *config.AgentConfig, agentID string, registry *collector.Registry) {
	log.Debug("resource collect start", "agent_id", agentID)

	start := time.Now()
	report := registry.Collect(ctx, agentID)
	if report == nil {
		log.Warn("resource collection returned nil report")
		return
	}
	collectDuration := time.Since(start)

	// Marshal and send to server.
	bodyBytes, err := json.Marshal(report)
	if err != nil {
		log.Error("failed to marshal resource report", "error", err)
		return
	}

	req, err := http.NewRequest("POST", cfg.ServerURL+"/api/agent/resources/report", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Error("failed to create resource report request", "error", err)
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
		return
	}
	defer resp.Body.Close()

	_ = resp // consume

	gpuCount := registry.GPUCount()
	log.Info("resource report success",
		"agent_id", agentID,
		"gpu_count", gpuCount,
		"collect_ms", collectDuration.Milliseconds(),
		"report_latency_ms", reportLatency.Milliseconds(),
		"payload_bytes", len(bodyBytes),
	)
}

func initLogging(cfg *config.AgentConfig) {
	// File logging setup is handled in config/log package.
	// This is a placeholder for future file log initialization.
}
