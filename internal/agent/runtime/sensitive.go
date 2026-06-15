package runtime

import (
	"fmt"
	"strings"
)

// sensitiveKeyFragments is the list of substrings that, when found in an env
// key (case-insensitive), mark that key as containing sensitive data.
//
// This mirrors the server-side sensitiveKeys() in
// internal/server/api/model_handlers.go.
var sensitiveKeyFragments = []string{
	"KEY",
	"TOKEN",
	"PASSWORD",
	"SECRET",
	"AUTH",
	"CREDENTIAL",
	"ACCESS",
	"PRIVATE",
}

// isSensitive reports whether envKey contains a known sensitive fragment.
func isSensitive(envKey string) bool {
	upper := strings.ToUpper(envKey)
	for _, frag := range sensitiveKeyFragments {
		if strings.Contains(upper, frag) {
			return true
		}
	}
	return false
}

// redactEnv returns a copy of env with sensitive values replaced by
// "<redacted>". Non-sensitive keys are left unchanged.
func redactEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return env
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		if isSensitive(k) {
			out[k] = "<redacted>"
		} else {
			out[k] = v
		}
	}
	return out
}

// redactEnvForLog returns a log-safe string representation of env.
// Sensitive values are replaced with "<redacted>"; non-sensitive values
// are shown in "KEY=VALUE" format.
func redactEnvForLog(env map[string]string) string {
	if len(env) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(env))
	for k, v := range env {
		if isSensitive(k) {
			parts = append(parts, fmt.Sprintf("%s=<redacted>", k))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
