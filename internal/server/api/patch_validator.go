package api

import (
	"fmt"
	"os"
	"strings"

	"lightai-go/internal/common/log"
)

// Ensure os is used (for os.Getenv).

// PATCH mode configuration.
// LIGHTAI_STRICT_PATCH=true  → strict: unknown/immutable fields return 400. (default)
// LIGHTAI_STRICT_PATCH=false → lax:    unknown/immutable fields log WARN and are dropped.
// Recommended: keep strict in dev/CI; only disable in production if backward compat required.
//
// Default: strict. This is the safe default — production deployments that need
// backward-compatible lax mode must explicitly opt in via env var.

// IsStrictPatch returns true if strict PATCH mode is active (default: true).
func IsStrictPatch() bool {
	v := os.Getenv("LIGHTAI_STRICT_PATCH")
	if v == "false" || v == "0" {
		return false
	}
	return true // default strict
}

// PatchFieldPolicy classifies a field relative to a PATCH handler.
type PatchFieldPolicy struct {
	Allowed   []string // fields that can be updated via PATCH
	Immutable []string // fields that exist in the resource but cannot be changed after creation
}

// PatchResult holds the outcome of PATCH field validation.
type PatchResult struct {
	CleanPayload    map[string]interface{}
	UnknownFields   []string
	ImmutableFields []string
}

// ValidatePatchFields checks the request payload against the policy.
// In strict mode unknown/immutable fields cause an error.
// In lax mode they are dropped with a WARN log.
func ValidatePatchFields(req map[string]interface{}, policy PatchFieldPolicy, handlerName, resourceID string) (PatchResult, error) {
	allowed := make(map[string]bool, len(policy.Allowed))
	for _, k := range policy.Allowed {
		allowed[k] = true
	}
	immutable := make(map[string]bool, len(policy.Immutable))
	for _, k := range policy.Immutable {
		immutable[k] = true
	}

	strict := IsStrictPatch()
	result := PatchResult{CleanPayload: make(map[string]interface{})}
	var unknown, immut []string

	for k, v := range req {
		if allowed[k] {
			result.CleanPayload[k] = v
		} else if immutable[k] {
			immut = append(immut, k)
		} else {
			unknown = append(unknown, k)
		}
	}

	result.UnknownFields = unknown
	result.ImmutableFields = immut

	// Log and possibly error.
	if len(unknown) > 0 {
		msg := fmt.Sprintf("PATCH %s/%s: unknown fields %v", handlerName, resourceID, unknown)
		if strict {
			log.Warn(msg + " (rejected in strict mode)")
			return result, fmt.Errorf("unsupported fields: %s", strings.Join(unknown, ", "))
		}
		log.Warn(msg + " (dropped in lax mode)")
	}

	if len(immut) > 0 {
		msg := fmt.Sprintf("PATCH %s/%s: immutable fields %v", handlerName, resourceID, immut)
		if strict {
			log.Warn(msg + " (rejected in strict mode)")
			return result, fmt.Errorf("immutable fields cannot be patched: %s", strings.Join(immut, ", "))
		}
		log.Warn(msg + " (dropped in lax mode)")
	}

	return result, nil
}
