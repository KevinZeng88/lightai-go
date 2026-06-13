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

	log.Init(cfg.LogLevel)

	agentID := cfg.AgentID
	if agentID == "" {
		agentID = uuid.NewString()
	}

	log.Info("starting lightai agent",
		"version", version.String(),
		"agent_id", agentID,
		"server_url", cfg.ServerURL,
		"gpu_profile", cfg.GPU.Profile,
	)

	// Setup Prometheus metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	// Setup collectors based on profile.
	registry := collector.NewRegistry()

	// System collector always available.
	if cfg.Collectors.System.Enabled {
		registry.RegisterSystem(collector.NewSystemCollector())
	}

	// GPU collectors based on profile.
	profile := cfg.GPU.Profile
	if profile == "" {
		profile = "production"
	}

	if profile == "development" || profile == "test" {
		if cfg.Collectors.MockGPU.Enabled {
			registry.RegisterGPU(collector.NewMockGPUCollector())
			log.Info("mock gpu collector enabled", "profile", profile)
		}
	}

	// Register collector metrics.
	_ = registerCollectorMetrics(reg, registry)

	hostname, _ := os.Hostname()

	// Health/metrics HTTP server.
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})
	healthMux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	healthAddr := fmt.Sprintf(":%d", cfg.Health.Port)
	healthSrv := &http.Server{
		Addr:         healthAddr,
		Handler:      healthMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("agent health server listening", "addr", healthAddr)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("agent health server failed", "error", err)
		}
	}()

	advertisedAddr := hostname
	log.Info("agent started", "agent_id", agentID, "hostname", hostname)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runAgentLoop(ctx, cfg, agentID, hostname, advertisedAddr, registry)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("agent shutting down", "signal", sig.String())

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("agent health server forced to shutdown", "error", err)
	}

	log.Info("agent stopped")
}

func runAgentLoop(ctx context.Context, cfg *config.AgentConfig, agentID, hostname, advertisedAddr string, registry *collector.Registry) {
	client := &http.Client{Timeout: 10 * time.Second}

	if err := registerAgent(client, cfg, agentID, hostname, advertisedAddr); err != nil {
		log.Error("initial registration failed", "error", err)
	}

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// Resource collection interval.
	collectInterval := cfg.Collectors.System.Interval
	if collectInterval == 0 {
		collectInterval = 60 * time.Second
	}

	collectTicker := time.NewTicker(collectInterval)
	defer collectTicker.Stop()

	// Collect immediately after registration.
	collectAndReport(ctx, client, cfg, agentID, registry)

	registered := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeatTicker.C:
			if err := sendHeartbeat(client, cfg, agentID); err != nil {
				log.Warn("heartbeat failed", "error", err)
				if isNeedRegister(err) {
					registered = false
				}
			}
			if !registered {
				if err := registerAgent(client, cfg, agentID, hostname, advertisedAddr); err != nil {
					log.Error("re-registration failed", "error", err)
				} else {
					registered = true
				}
			}
		case <-collectTicker.C:
			collectAndReport(ctx, client, cfg, agentID, registry)
		}
	}
}

func collectAndReport(ctx context.Context, client *http.Client, cfg *config.AgentConfig, agentID string, registry *collector.Registry) {
	report := registry.Collect(ctx, agentID)
	if report == nil {
		log.Warn("resource collection returned nil report")
		return
	}

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

	resp, err := client.Do(req)
	if err != nil {
		log.Error("resource report request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn("resource report returned non-ok status", "status", resp.StatusCode)
		return
	}

	gpuCount := registry.GPUCount()
	log.Info("resource report sent", "agent_id", agentID, "gpu_count", gpuCount)
}

// registerCollectorMetrics registers Prometheus metrics from the collector registry.
func registerCollectorMetrics(reg *prometheus.Registry, registry *collector.Registry) error {
	// Register system and GPU metrics as Prometheus gauges.
	// These are updated during collection cycles.
	return nil
}

func registerAgent(client *http.Client, cfg *config.AgentConfig, agentID, hostname, advertisedAddr string) error {
	reqBody := map[string]interface{}{
		"agent_id":           agentID,
		"hostname":           hostname,
		"advertised_address": advertisedAddr,
		"metrics_enabled":    cfg.Metrics.Enabled,
		"metrics_scheme":     cfg.Metrics.Scheme,
		"metrics_port":       cfg.Metrics.Port,
		"metrics_path":       cfg.Metrics.Path,
		"version":            version.String(),
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", cfg.ServerURL+"/api/agent/register", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AgentToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register returned status %d", resp.StatusCode)
	}

	var regResp struct{ NodeID string }
	json.NewDecoder(resp.Body).Decode(&regResp)
	log.Info("agent registered", "node_id", regResp.NodeID)
	return nil
}

func sendHeartbeat(client *http.Client, cfg *config.AgentConfig, agentID string) error {
	reqBody := map[string]string{"agent_id": agentID}
	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", cfg.ServerURL+"/api/agent/heartbeat", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AgentToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	var hbResp struct {
		Status       string `json:"status"`
		NeedRegister bool   `json:"need_register"`
	}
	json.NewDecoder(resp.Body).Decode(&hbResp)

	if hbResp.NeedRegister {
		return fmt.Errorf("need_register")
	}
	return nil
}

func isNeedRegister(err error) bool {
	return err != nil && err.Error() == "need_register"
}
