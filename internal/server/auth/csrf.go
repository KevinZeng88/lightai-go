package auth

import (
	"crypto/subtle"
	"net/http"
	"net/url"
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

// ValidateOrigin validates the Origin/Referer header against the request Host.
func ValidateOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Fall back to Referer for same-origin requests.
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return false
	}

	host := r.Host
	if host == "" {
		return false
	}

	// Parse origin URL and compare host components.
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	originHost := parsed.Host

	// Allow if origin host matches request host exactly.
	return originHost == host
}
