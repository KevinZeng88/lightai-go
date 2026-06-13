// LightAI Go Agent - Execution plane running on GPU servers.
package main

import (
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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configPath = flag.String("config", "", "path to config file (YAML)")
)

func main() {
	flag.Parse()

	// Load config.
	cfg, err := config.LoadAgentConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Init logger.
	log.Init(cfg.LogLevel)

	log.Info("starting lightai agent",
		"version", version.String(),
		"agent_id", cfg.AgentID,
		"server_url", cfg.ServerURL,
		"log_level", cfg.LogLevel,
	)

	// Setup metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	// Setup HTTP mux for health and metrics.
	healthMux := http.NewServeMux()

	// Register healthz.
	healthMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})

	// Register metrics.
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

	log.Info("agent started")

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("agent shutting down", "signal", sig.String())

	// Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := healthSrv.Shutdown(ctx); err != nil {
		log.Error("agent health server forced to shutdown", "error", err)
	}

	log.Info("agent stopped")
}
