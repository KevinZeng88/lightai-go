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

	"lightai-go/internal/common/config"
	"lightai-go/internal/common/log"
	"lightai-go/internal/common/types"
	"lightai-go/internal/common/version"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configPath = flag.String("config", "", "path to config file (YAML)")
)

func main() {
	flag.Parse()

	cfg, err := config.LoadAgentConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Init(cfg.LogLevel)

	// Determine agent ID.
	agentID := cfg.AgentID
	if agentID == "" {
		agentID = uuid.NewString()
		log.Info("generated new agent id", "agent_id", agentID)
	}

	log.Info("starting lightai agent",
		"version", version.String(),
		"agent_id", agentID,
		"server_url", cfg.ServerURL,
		"log_level", cfg.LogLevel,
	)

	// Setup metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	// Determine hostname.
	hostname, _ := os.Hostname()

	// Setup HTTP mux for health and metrics.
	healthMux := http.NewServeMux()

	healthMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})

	healthMux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Start health/metrics server.
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

	// Determine advertised address.
	advertisedAddr := hostname
	// Use localhost for development; in production this should be the actual IP.

	log.Info("agent started", "agent_id", agentID, "hostname", hostname)

	// Run registration and heartbeat loops.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runAgentLoop(ctx, cfg, agentID, hostname, advertisedAddr)

	// Wait for shutdown signal.
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

// runAgentLoop runs the agent's main loop: register, then heartbeat periodically.
func runAgentLoop(ctx context.Context, cfg *config.AgentConfig, agentID, hostname, advertisedAddr string) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Register immediately.
	if err := registerAgent(client, cfg, agentID, hostname, advertisedAddr); err != nil {
		log.Error("initial registration failed", "error", err)
	}

	// Heartbeat every 30 seconds.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	registered := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sendHeartbeat(client, cfg, agentID); err != nil {
				log.Warn("heartbeat failed", "error", err)
				// If we get need_register, re-register.
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
		}
	}
}

// registerAgent sends a registration request to the server.
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

	var regResp struct {
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		log.Warn("failed to decode register response", "error", err)
	}

	log.Info("agent registered", "node_id", regResp.NodeID, "agent_id", agentID)
	return nil
}

// sendHeartbeat sends a heartbeat to the server.
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
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return fmt.Errorf("decode heartbeat response: %w", err)
	}

	if hbResp.NeedRegister {
		return fmt.Errorf("need_register")
	}

	return nil
}

func isNeedRegister(err error) bool {
	return err != nil && err.Error() == "need_register"
}
