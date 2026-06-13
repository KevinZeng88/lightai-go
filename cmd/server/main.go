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

var configPath = flag.String("config", "", "path to config file (YAML)")

func main() {
	flag.Parse()

	cfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Init(cfg.LogLevel)

	log.Info("server starting",
		"version", version.String(),
		"listen", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		"log_level", cfg.LogLevel,
		"db_path", cfg.DBPath,
	)

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal("failed to open database", "error", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatal("failed to run migrations", "error", err)
	}

	bootstrapCfg := auth.BootstrapConfig{
		Username:            "admin",
		Password:            "",
		PasswordEnv:         "LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD",
		ForceChangePassword: true,
	}
	if err := auth.InitBootstrap(database, bootstrapCfg); err != nil {
		log.Fatal("failed to initialize bootstrap", "error", err)
	}

	sessionCfg := auth.DefaultSessionConfig()
	sessionStore := auth.NewSessionStore(database, sessionCfg)
	rateLimiter := auth.NewLoginRateLimiter(rate.Limit(1), 5)

	authHandler := &auth.AuthHandler{
		DB:           database,
		SessionStore: sessionStore,
		SessionCfg:   sessionCfg,
		RateLimiter:  rateLimiter,
		BootstrapCfg: bootstrapCfg,
	}
	rbacHandler := rbac.NewHandler(database)
	agentHandler := api.NewAgentHandler(database)
	resourceHandler := api.NewResourceHandler(database)

	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.HealthResponse{Status: "ok"})
	})

	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	mux.HandleFunc("GET /metrics/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		targets := agentHandler.GetMetricsTargets()
		if targets == nil {
			targets = []map[string]interface{}{}
		}
		json.NewEncoder(w).Encode(targets)
	})

	api.SetupRoutes(mux, api.RouterConfig{
		DB:              database,
		AgentToken:      cfg.AgentToken,
		SessionStore:    sessionStore,
		SessionCfg:      sessionCfg,
		AuthHandler:     authHandler,
		RBACHandler:     rbacHandler,
		AgentHandler:    agentHandler,
		ResourceHandler: resourceHandler,
	})

	// Serve embedded web assets or fallback.
	serveEmbeddedWeb(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("server shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server stopped")
}

// serveFallbackPage returns a helpful HTML page when web assets are not built.
func serveFallbackPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html><head><title>LightAI Go</title></head>
<body style="font-family: sans-serif; padding: 2rem;">
<h1>LightAI Go Server</h1>
<p>API server is running.</p>
<p>Web assets not built. Run:</p>
<pre>cd web && npm run build</pre>
<p>Then rebuild with <code>-tags web</code>.</p>
<hr>
<p><small>API: <a href="/api/auth/me">/api/auth/me</a> | <a href="/healthz">/healthz</a> | <a href="/metrics">/metrics</a> | <a href="/metrics/targets">/metrics/targets</a></small></p>
</body></html>`))
}
