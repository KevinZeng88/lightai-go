package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRecoveryMiddleware_PanicReturns500 verifies that a panicking handler
// is caught by RecoveryMiddleware and returns HTTP 500.
func TestRecoveryMiddleware_PanicReturns500(t *testing.T) {
	// Handler that panics.
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic: something went wrong")
	})

	middleware := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	// Should not panic — middleware should catch it.
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic handler: got status %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if !strings.Contains(body, "internal server error") {
		t.Errorf("panic handler: response body %q should contain 'internal server error'", body)
	}
}

// TestRecoveryMiddleware_NormalRequestPasses verifies that normal requests
// pass through RecoveryMiddleware without modification.
func TestRecoveryMiddleware_NormalRequestPasses(t *testing.T) {
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	middleware := RecoveryMiddleware(normalHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("normal handler: got status %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != `{"status":"ok"}` {
		t.Errorf("normal handler: got body %q, want %q", body, `{"status":"ok"}`)
	}
}

// TestRecoveryMiddleware_PanicWithNil verifies that a nil panic
// is handled gracefully.
func TestRecoveryMiddleware_PanicWithNil(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	})

	middleware := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// panic(nil) is still a panic — should return 500.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("nil panic: got status %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestRecoveryMiddleware_PanicWithStruct verifies that panicking with a struct
// is handled gracefully.
func TestRecoveryMiddleware_PanicWithStruct(t *testing.T) {
	type customError struct {
		Code    int
		Message string
	}

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(customError{Code: 42, Message: "custom error"})
	})

	middleware := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("struct panic: got status %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestRecoveryMiddleware_ChainedWithLogging verifies that RecoveryMiddleware
// works correctly when chained with RequestLoggingMiddleware.
func TestRecoveryMiddleware_ChainedWithLogging(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("chained panic")
	})

	// Recovery should be outermost (catches panics from logging middleware too).
	handler := RecoveryMiddleware(RequestLoggingMiddleware(panicHandler))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("chained panic: got status %d, want %d", w.Code, http.StatusInternalServerError)
	}

	// Verify X-Request-ID is still set (logging middleware ran before panic).
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header not set in chained middleware")
	}
}
