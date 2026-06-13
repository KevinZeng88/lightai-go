// LightAI Go Server - Control plane for GPU infrastructure management.
package main

import (
	"context"
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
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())
	serverMetrics := srvmetrics.New(reg, database.DB)

	agentHandler := api.NewAgentHandler(database, serverMetrics)
	resourceHandler := api.NewResourceHandler(database, serverMetrics)

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
<p><small>API: <a href="/api/auth/me">/api/auth/me</a> | <a href="/healthz">/healthz</a> | <a href="/metrics">/metrics</a> | <a href="/metrics/targets">/metrics/targets</a></small></p>
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
		if !strings.HasPrefix(r.URL.Path, "/api/") {
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
