package auth

import (
	"crypto/subtle"
	"net/http"
)

// CSRFHeader is the header name for CSRF tokens.
const CSRFHeader = "X-CSRF-Token"

// ValidateCSRF validates a CSRF token against the session's CSRF secret.
func ValidateCSRF(r *http.Request, csrfSecretHash string) bool {
	token := r.Header.Get(CSRFHeader)
	if token == "" {
		return false
	}
	tokenHash := hashString(token)
	return subtle.ConstantTimeCompare([]byte(tokenHash), []byte(csrfSecretHash)) == 1
}

// ValidateOrigin validates the Origin/Referer header.
func ValidateOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Fall back to Referer for same-origin requests.
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return false
	}

	// For development, allow localhost origins.
	// In production, this should match the actual server origin.
	host := r.Host
	if host == "" {
		return false
	}

	// Simple origin check: the origin should contain the host.
	// This is a basic check - production should be more strict.
	return containsOrigin(origin, host)
}

func containsOrigin(origin, host string) bool {
	// Check if origin contains the host as a host component.
	// e.g., "http://127.0.0.1:18080" contains "127.0.0.1:18080"
	return len(origin) >= len(host) &&
		origin[len(origin)-len(host):] == host
}
