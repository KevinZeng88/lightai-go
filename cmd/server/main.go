// LightAI Go Server - Control plane for GPU infrastructure management.
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
	cfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Init logger.
	log.Init(cfg.LogLevel)

	log.Info("starting lightai server",
		"version", version.String(),
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
		"db_path", cfg.DBPath,
	)

	// Setup metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	// Create metrics targets store (placeholder for Phase 0).
	targetsStore := &metricsTargetsStore{
		targets: []types.MetricTarget{},
	}

	// Setup HTTP mux.
	mux := http.NewServeMux()

	// Register healthz.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})

	// Register metrics.
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Register metrics targets (Prometheus HTTP SD format).
	mux.HandleFunc("GET /metrics/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		targets := targetsStore.List()
		if targets == nil {
			targets = []types.MetricTarget{}
		}
		json.NewEncoder(w).Encode(targets)
	})

	// Create server.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine.
	go func() {
		log.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "error", err)
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("server shutting down", "signal", sig.String())

	// Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server stopped")
}

// metricsTargetsStore manages Prometheus HTTP SD targets.
// In Phase 0 this is a placeholder. In Phase 1 it will be populated
// from registered agents.
type metricsTargetsStore struct {
	targets []types.MetricTarget
}

// List returns all current targets.
func (s *metricsTargetsStore) List() []types.MetricTarget {
	return s.targets
}
