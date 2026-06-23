// Package api — request logging middleware with request_id generation.
package api

import (
	"net/http"
	"strings"
	"time"

	"lightai-go/internal/common/log"
)

// highFrequencyPrefixes are API path prefixes that should not log INFO on success.
// These are typically polled by frontend or agent at regular intervals.
var highFrequencyPrefixes = []string{
	"/api/v1/agent/heartbeat",
	"/api/v1/agent/resources/report",
	"/api/v1/agent/tasks/", // task result endpoint
	"/metrics",
}

// staticAssetPrefixes are path prefixes for static web assets.
// Successful (2xx) requests to these paths are logged at DEBUG to reduce noise.
var staticAssetPrefixes = []string{
	"/assets/",
}

// staticAssetSuffixes are path suffixes for static web assets.
var staticAssetSuffixes = []string{
	"/favicon.ico",
	"/favicon.png",
	"/manifest.json",
	"/robots.txt",
}

// routeSpecificSlowThresholds maps route prefixes to their slow-operation thresholds in ms.
// Routes not listed here use the global SummaryConfig.SlowAPIThresholdMs (1000ms).
var routeSpecificSlowThresholds = map[string]int64{
	"/api/v1/node-run-plans/": 3000, // logs endpoint can legitimately take longer
}

// isHighFrequencyGET returns true for GET requests to frequently-polled resources.
func isHighFrequencyGET(method, path string) bool {
	if method != "GET" {
		return false
	}
	for _, prefix := range []string{
		"/api/v1/model-instances",
		"/api/v1/model-deployments",
		"/api/v1/nodes",
		"/api/v1/gpus",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// isHighFrequencyPath returns true for paths that should use DEBUG for success.
func isHighFrequencyPath(path string) bool {
	for _, prefix := range highFrequencyPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// isStaticAsset returns true for static web asset paths (CSS, JS, favicon, etc.).
func isStaticAsset(path string) bool {
	for _, prefix := range staticAssetPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, suffix := range staticAssetSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// getSlowThreshold returns the route-specific slow threshold for the given path,
// or the global default if no route-specific threshold is configured.
func getSlowThreshold(path string) int64 {
	for prefix, threshold := range routeSpecificSlowThresholds {
		if strings.HasPrefix(path, prefix) {
			return threshold
		}
	}
	return log.SummaryConfig.SlowAPIThresholdMs
}

// extractClientIP extracts the client IP from the request.
// It uses RemoteAddr (host:port format) and strips the port.
// Does NOT trust X-Forwarded-For by default — that requires explicit proxy configuration.
func extractClientIP(r *http.Request) string {
	ip := r.RemoteAddr
	// Strip port from host:port format.
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	// Strip IPv6 brackets if present.
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimSuffix(ip, "]")
	return ip
}

// truncateString truncates a string to maxLen characters, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// RequestLoggingMiddleware wraps an http.Handler with structured request logging.
// It generates a request_id (or reads X-Request-ID header), injects it into the
// request context, and logs request start/completion with duration.
//
// High-frequency endpoints (heartbeat, resource report) use DEBUG for success.
// Static assets (2xx) use DEBUG to reduce noise; errors are still logged.
// Slow requests get a WARN with route-specific thresholds.
func RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or read request_id.
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = log.NewRequestID()
		}
		ctx := log.WithRequestID(r.Context(), reqID)
		r = r.WithContext(ctx)

		// Set response header so clients can correlate.
		w.Header().Set("X-Request-ID", reqID)

		start := time.Now()
		wr := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request start at DEBUG level for all paths.
		log.DebugContext(ctx, "api.request.received",
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(wr, r)

		durationMs := time.Since(start).Milliseconds()
		path := StripPathParams(r.URL.Path)
		hf := isHighFrequencyPath(r.URL.Path) || isHighFrequencyGET(r.Method, path)
		static := isStaticAsset(r.URL.Path)
		clientIP := extractClientIP(r)
		userAgent := truncateString(r.UserAgent(), 200)

		// Common fields for all log entries.
		commonFields := []any{
			"method", r.Method,
			"path", path,
			"status", wr.statusCode,
			"duration_ms", durationMs,
			"client_ip", clientIP,
			"user_agent", userAgent,
		}

		if wr.statusCode >= 500 {
			log.ErrorContext(ctx, "api.request.completed", commonFields...)
		} else if wr.statusCode >= 400 {
			// Static asset 4xx (e.g., favicon 404) still logged as WARN.
			log.WarnContext(ctx, "api.request.completed", commonFields...)
		} else if hf || (static && wr.statusCode < 400) {
			// High-frequency success or static asset success: DEBUG only.
			log.DebugContext(ctx, "api.request.completed", commonFields...)
		} else {
			log.InfoContext(ctx, "api.request.completed", commonFields...)
		}

		// Slow request warning with route-specific threshold.
		threshold := getSlowThreshold(r.URL.Path)
		if durationMs > threshold {
			log.SlowOperation(ctx, "api.request", "http_handler", durationMs, threshold,
				"method", r.Method,
				"path", path,
			)
		}
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture the status code.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
