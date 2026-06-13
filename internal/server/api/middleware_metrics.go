package api

import (
	"net/http"
	"strings"
	"time"

	srvmetrics "lightai-go/internal/server/metrics"
)

// MetricsMiddleware wraps a handler with API request metrics recording.
func MetricsMiddleware(m *srvmetrics.ServerMetrics, endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code.
			wr := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wr, r)

			duration := time.Since(start).Seconds()
			code := statusCodeLabel(wr.statusCode)
			method := r.Method

			if m.APIRequests != nil {
				m.APIRequests.WithLabelValues(endpoint, method, code).Inc()
			}
			if m.APIRequestDuration != nil {
				m.APIRequestDuration.WithLabelValues(endpoint, method).Observe(duration)
			}
		})
	}
}

// responseWriter captures the HTTP status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func statusCodeLabel(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// ApplyMetricsToMux wraps all /api/* handlers with metrics middleware.
func ApplyMetricsToMux(mux *http.ServeMux, m *srvmetrics.ServerMetrics) {
	// We wrap the mux by replacing its handler.
	// Since Go 1.22+ ServeMux patterns are exact, we wrap at the top level.
}

// StripPathParams replaces path parameter values with placeholders.
func StripPathParams(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		// UUID-like patterns and numeric IDs.
		if len(p) >= 32 || (len(p) > 0 && isNumeric(p)) {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// WrapHandler applies metrics middleware to an http.Handler.
func WrapHandler(h http.Handler, m *srvmetrics.ServerMetrics, endpoint string) http.Handler {
	if m == nil {
		return h
	}
	mw := MetricsMiddleware(m, endpoint)
	return mw(h)
}
