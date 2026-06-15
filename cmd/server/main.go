// LightAI Go Server - Control plane for GPU infrastructure management.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"lightai-go/internal/common/config"
	"lightai-go/internal/common/log"
	"lightai-go/internal/common/types"
	"lightai-go/internal/common/version"
	"lightai-go/internal/server/api"
	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	srvmetrics "lightai-go/internal/server/metrics"
	"lightai-go/internal/server/rbac"
	"lightai-go/web"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

var configPath = flag.String("config", "", "path to config file (YAML)")
var resetAdminPassword = flag.String("reset-admin-password", "", "reset admin password to the given value and exit")
var resetAdminPasswordInteractive = flag.Bool("reset-admin-password-interactive", false, "reset admin password via interactive prompt")
var showVersion = flag.Bool("version", false, "show version and exit")

func main() {
	flag.Parse()

	// --- Version mode (exits early) ---
	if *showVersion {
		fmt.Println(version.String())
		return
	}

	// --- Password reset mode (exits early) ---
	if *resetAdminPassword != "" || *resetAdminPasswordInteractive {
		cfg, err := config.LoadServerConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}
		runResetAdminPassword(cfg, *resetAdminPassword, *resetAdminPasswordInteractive)
		return
	}

	cfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

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

	// P0-011: Check for default agent token in production.
	if cfg.AgentToken == "" || cfg.AgentToken == "lightai-agent-token-change-me" || cfg.AgentToken == "dev-agent-token" {
		if cfg.DevMode {
			log.Warn("using default agent token in dev mode — NOT safe for production",
				"agent_token", cfg.AgentToken,
			)
		} else {
			log.Error("DEFAULT AGENT TOKEN DETECTED",
				"agent_token", cfg.AgentToken,
				"help", "Set LIGHTAI_AGENT_TOKEN env var to a secure random value.",
			)
			fmt.Fprintf(os.Stderr, "\n=== SECURITY WARNING ===\n")
			fmt.Fprintf(os.Stderr, "Default agent token detected: %s\n", cfg.AgentToken)
			fmt.Fprintf(os.Stderr, "This is NOT safe for production.\n")
			fmt.Fprintf(os.Stderr, "Set LIGHTAI_AGENT_TOKEN env var to a secure random value.\n")
			fmt.Fprintf(os.Stderr, "Example: export LIGHTAI_AGENT_TOKEN=$(openssl rand -hex 32)\n")
			fmt.Fprintf(os.Stderr, "=========================\n\n")
		}
	}

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
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())
	serverMetrics := srvmetrics.New(reg, database.DB)

	agentHandler := api.NewAgentHandler(database, serverMetrics)
	resourceHandler := api.NewResourceHandler(database, serverMetrics)
	modelHandler := api.NewModelHandler(database)

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
		ServerMetrics:   serverMetrics,
		AgentToken:      cfg.AgentToken,
		SessionStore:    sessionStore,
		SessionCfg:      sessionCfg,
		AuthHandler:     authHandler,
		RBACHandler:     rbacHandler,
		AgentHandler:    agentHandler,
		ResourceHandler: resourceHandler,
		ModelHandler:    modelHandler,
	})

	// Return JSON 404 for unregistered /api/* paths — never fall back to SPA index.html.
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found","path":"` + r.URL.Path + `"}`))
	})

	// Serve embedded web assets or fallback.
	serveWeb(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      metricsWrapper(mux, serverMetrics),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// P0-009: Start node health checker (periodic offline detection).
	nodeHealthStop := make(chan struct{})
	go runNodeHealthChecker(agentHandler, resourceHandler, cfg, nodeHealthStop)

	// Start task/lease sweep loop for periodic timeout cleanup.
	sweepStop := make(chan struct{})
	go api.RunSweepLoop(database, 30*time.Second, sweepStop)

	go func() {
		log.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	close(nodeHealthStop)
	close(sweepStop)
	log.Info("server shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server stopped")
}

// serveWeb serves embedded web/dist or shows a fallback page.
func serveWeb(mux *http.ServeMux) {
	distFS, err := web.GetDist()
	if err != nil {
		log.Info("web assets not built; run 'cd web && npm run build'")
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
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
<p><small>API: <a href="/api/v1/auth/me">/api/auth/me</a> | <a href="/healthz">/healthz</a> | <a href="/metrics">/metrics</a> | <a href="/metrics/targets">/metrics/targets</a></small></p>
</body></html>`))
		})
		return
	}

	fileServer := http.FileServer(http.FS(distFS))
	mux.Handle("GET /assets/", fileServer)
	mux.Handle("GET /favicon.ico", fileServer)
	mux.Handle("GET /favicon.png", fileServer)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		data, err := fs.ReadFile(distFS, path)
		if err == nil {
			contentType := "application/octet-stream"
			switch {
			case strings.HasSuffix(path, ".html"):
				contentType = "text/html; charset=utf-8"
			case strings.HasSuffix(path, ".css"):
				contentType = "text/css"
			case strings.HasSuffix(path, ".js"):
				contentType = "application/javascript"
			case strings.HasSuffix(path, ".svg"):
				contentType = "image/svg+xml"
			case strings.HasSuffix(path, ".png"):
				contentType = "image/png"
			case strings.HasSuffix(path, ".ico"):
				contentType = "image/x-icon"
			}
			w.Header().Set("Content-Type", contentType)
			w.Write(data)
			return
		}
		// SPA fallback: serve index.html.
		data, err = fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
}

// metricsWrapper wraps the mux with API request metrics recording.
// Only /api/* paths are tracked; /metrics, /healthz, and static assets are excluded.
func metricsWrapper(mux *http.ServeMux, m *srvmetrics.ServerMetrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := &respWriter{ResponseWriter: w, statusCode: http.StatusOK}
		mux.ServeHTTP(wr, r)
		duration := time.Since(start).Seconds()

		// Only record API paths to avoid polluting counters with
		// Prometheus scrapes, health checks, and static asset requests.
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			return
		}

		if m != nil && m.APIRequests != nil {
			ep := api.StripPathParams(r.URL.Path)
			code := "2xx"
			if wr.statusCode >= 400 {
				code = "4xx"
			}
			if wr.statusCode >= 500 {
				code = "5xx"
			}
			m.APIRequests.WithLabelValues(ep, r.Method, code).Inc()
			if m.APIRequestDuration != nil {
				m.APIRequestDuration.WithLabelValues(ep, r.Method).Observe(duration)
			}
		}
	})
}

type respWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *respWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// runResetAdminPassword handles --reset-admin-password and --reset-admin-password-interactive.
func runResetAdminPassword(cfg *config.ServerConfig, newPassword string, interactive bool) {
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if interactive {
		fmt.Fprintf(os.Stderr, "Enter new admin password: ")
		var input string
		fmt.Scanln(&input)
		if input == "" {
			fmt.Fprintf(os.Stderr, "ERROR: empty password\n")
			os.Exit(1)
		}
		newPassword = input
	}

	if newPassword == "" {
		// Auto-generate a secure random password (16 bytes → 32 hex chars).
		pwBytes := make([]byte, 16)
		if _, err := rand.Read(pwBytes); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to generate password: %v\n", err)
			os.Exit(1)
		}
		newPassword = hex.EncodeToString(pwBytes)
	}

	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to hash password: %v\n", err)
		os.Exit(1)
	}

	result, err := database.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 1, updated_at = datetime('now') WHERE username = 'admin'`,
		passwordHash,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to update password: %v\n", err)
		os.Exit(1)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		fmt.Fprintf(os.Stderr, "WARNING: admin user not found (database may not be initialized yet)\n")
		os.Exit(1)
	}

	// Write reset credentials file.
	credDir := "runtime"
	os.MkdirAll(credDir, 0700)
	credPath := "runtime/reset-credentials.txt"
	timestamp := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(`============================================
LightAI Go - Password Reset
Reset time: %s
============================================

[Web/Admin]
Username: admin
Password: %s
Note: You must change this password after next login.
Service restart required: no
Next step: Login at the web UI with this new password.
`, timestamp, newPassword)

	if err := os.WriteFile(credPath, []byte(content), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: password updated but failed to write credentials file: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Admin password reset. Credentials written to %s\n", credPath)
	}
}

// runNodeHealthChecker periodically checks node heartbeats and marks
// nodes as offline if they exceed the configured threshold.
// P0-009: Node auto-offline implementation.
func runNodeHealthChecker(agentHandler *api.AgentHandler, resourceHandler *api.ResourceHandler, cfg *config.ServerConfig, stop <-chan struct{}) {
	threshold := cfg.NodeOfflineThreshold
	if threshold == 0 {
		threshold = 30 * time.Second
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Info("node health checker started",
		"offline_threshold", threshold.String(),
		"check_interval", "10s",
	)

	for {
		select {
		case <-stop:
			log.Info("node health checker stopped")
			return
		case <-ticker.C:
			n, err := agentHandler.MarkOfflineNodes(threshold)
			if err != nil {
				log.Error("node health check failed", "error", err)
			} else if n > 0 {
				log.Info("nodes marked offline by health checker", "count", n)

				// Also mark GPUs of offline nodes as stale.
				rows, err := agentHandler.DB.Query(
					`SELECT id FROM nodes WHERE status = 'offline'`,
				)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var nodeID string
						if err := rows.Scan(&nodeID); err != nil {
							continue
						}
						resourceHandler.MarkStaleGPUs(nodeID, 0)
					}
				}
			}
		}
	}
}
