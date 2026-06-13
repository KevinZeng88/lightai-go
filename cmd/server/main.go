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
	"lightai-go/internal/server/api"
	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	"lightai-go/internal/server/rbac"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
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

	// Open database.
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal("failed to open database", "error", err)
	}
	defer database.Close()

	// Run migrations.
	if err := database.Migrate(); err != nil {
		log.Fatal("failed to run migrations", "error", err)
	}

	// Initialize bootstrap admin.
	bootstrapCfg := auth.BootstrapConfig{
		Username:            "admin",
		Password:            "",
		PasswordEnv:         "LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD",
		ForceChangePassword: true,
	}
	if err := auth.InitBootstrap(database, bootstrapCfg); err != nil {
		log.Fatal("failed to initialize bootstrap", "error", err)
	}

	// Setup session store.
	sessionCfg := auth.DefaultSessionConfig()
	sessionStore := auth.NewSessionStore(database, sessionCfg)

	// Setup rate limiter (1 login per second, burst 5).
	rateLimiter := auth.NewLoginRateLimiter(rate.Limit(1), 5)

	// Setup handlers.
	authHandler := &auth.AuthHandler{
		DB:           database,
		SessionStore: sessionStore,
		SessionCfg:   sessionCfg,
		RateLimiter:  rateLimiter,
		BootstrapCfg: bootstrapCfg,
	}
	rbacHandler := rbac.NewHandler(database)

	// Setup metrics.
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	// Create metrics targets store (populated in Phase 1).
	targetsStore := &metricsTargetsStore{
		targets: []types.MetricTarget{},
	}

	// Setup HTTP mux.
	mux := http.NewServeMux()

	// Public endpoints.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})

	// Metrics (no auth).
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Metrics targets (Prometheus HTTP SD).
	mux.HandleFunc("GET /metrics/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		targets := targetsStore.List()
		if targets == nil {
			targets = []types.MetricTarget{}
		}
		json.NewEncoder(w).Encode(targets)
	})

	// Setup API routes.
	api.SetupRoutes(mux, api.RouterConfig{
		DB:           database,
		AgentToken:   cfg.AgentToken,
		SessionStore: sessionStore,
		SessionCfg:   sessionCfg,
		AuthHandler:  authHandler,
		RBACHandler:  rbacHandler,
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
type metricsTargetsStore struct {
	targets []types.MetricTarget
}

func (s *metricsTargetsStore) List() []types.MetricTarget {
	return s.targets
}
