// Package log — unified redaction helpers for LightAI Go.
package log

import (
	"strings"
)

// sensitiveKeyFragments are substrings that, when found case-insensitively in a key,
// indicate the value should be redacted from logs.
var sensitiveKeyFragments = []string{
	"KEY", "TOKEN", "PASSWORD", "PASSWD", "PWD",
	"SECRET", "AUTH", "CREDENTIAL", "ACCESS",
	"API_KEY", "APIKEY", "ACCESS_KEY", "SECRET_KEY",
	"AUTHORIZATION", "BEARER",
	"HF_TOKEN", "DASHSCOPE_API_KEY", "OPENAI_API_KEY",
	"AK", "SK", "PRIVATE",
	"COOKIE", "SESSION", "CSRF",
}

const redactedValue = "<redacted>"

// IsSensitiveKey returns true if the key (case-insensitive) contains a known
// sensitive fragment.
func IsSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, sk := range sensitiveKeyFragments {
		if strings.Contains(upper, sk) {
			return true
		}
	}
	return false
}

// RedactValue returns "<redacted>" if the key is sensitive, otherwise the
// original value.
func RedactValue(key, value string) string {
	if IsSensitiveKey(key) {
		return redactedValue
	}
	return value
}

// RedactMap returns a copy of the map with sensitive values replaced by
// "<redacted>". The original map is not modified.
func RedactMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if IsSensitiveKey(k) {
			out[k] = redactedValue
		} else {
			out[k] = v
		}
	}
	return out
}

// RedactEnvForLog returns a log-safe representation of environment variables.
// Format: "KEY1=<redacted> KEY2=visible_value". Sensitive values are replaced.
func RedactEnvForLog(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	var b strings.Builder
	first := true
	for k, v := range env {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		b.WriteString(k)
		b.WriteByte('=')
		if IsSensitiveKey(k) {
			b.WriteString(redactedValue)
		} else {
			b.WriteString(v)
		}
	}
	return b.String()
}

// RedactEnvKeys returns a space-separated list of env keys (no values) for
// safe logging of what environment variables are set.
func RedactEnvKeys(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	return strings.Join(keys, " ")
}

// SanitizeHeaderValue redacts sensitive header values like Authorization.
// Returns the original value for non-sensitive headers, "<redacted>" for
// sensitive ones.
func SanitizeHeaderValue(headerName, value string) string {
	if IsSensitiveKey(headerName) {
		return redactedValue
	}
	return value
}
