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

// RequestLoggingMiddleware wraps an http.Handler with structured request logging.
// It generates a request_id (or reads X-Request-ID header), injects it into the
// request context, and logs request start/completion with duration.
//
// High-frequency endpoints (heartbeat, resource report) use DEBUG for success.
// Slow requests (> SummaryConfig.SlowAPIThresholdMs) get a WARN.
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

		if wr.statusCode >= 500 {
			log.ErrorContext(ctx, "api.request.completed",
				"method", r.Method,
				"path", path,
				"status", wr.statusCode,
				"duration_ms", durationMs,
			)
		} else if wr.statusCode >= 400 {
			log.WarnContext(ctx, "api.request.completed",
				"method", r.Method,
				"path", path,
				"status", wr.statusCode,
				"duration_ms", durationMs,
			)
		} else if hf {
			// High-frequency success: DEBUG only.
			log.DebugContext(ctx, "api.request.completed",
				"method", r.Method,
				"path", path,
				"status", wr.statusCode,
				"duration_ms", durationMs,
			)
		} else {
			log.InfoContext(ctx, "api.request.completed",
				"method", r.Method,
				"path", path,
				"status", wr.statusCode,
				"duration_ms", durationMs,
			)
		}

		// Slow request warning.
		if durationMs > log.SummaryConfig.SlowAPIThresholdMs {
			log.SlowOperation(ctx, "api.request", "http_handler", durationMs, log.SummaryConfig.SlowAPIThresholdMs,
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
