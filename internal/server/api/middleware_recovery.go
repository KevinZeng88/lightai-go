// Package api — panic recovery middleware for HTTP handlers.
package api

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"lightai-go/internal/common/log"
)

// RecoveryMiddleware wraps an http.Handler with panic recovery.
// If a handler panics, it logs the error with stack trace and returns HTTP 500.
// This prevents the server process from crashing due to unhandled panics.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Extract request_id from context if available.
				ctx := r.Context()
				reqID := log.RequestIDFromContext(ctx)

				// Get stack trace.
				stack := string(debug.Stack())

				// Log the panic with full context.
				log.ErrorContext(ctx, "handler.panic.recovered",
					"request_id", reqID,
					"method", r.Method,
					"path", r.URL.Path,
					"panic", fmt.Sprintf("%v", rec),
					"stack", stack,
				)

				// Return 500 to client.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
