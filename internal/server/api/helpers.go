package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lightai-go/internal/server/auth"
)

// ==========================================================================
// HTTP helpers
// ==========================================================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ==========================================================================
// Request helpers
// ==========================================================================

func userID(r *http.Request) string {
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil {
		return info.UserID
	}
	return "system"
}

func tenantID(r *http.Request) string {
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil {
		return info.TenantID
	}
	return ""
}

func actorIDFromSession(r *http.Request) string {
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil && info.UserID != "" {
		return info.UserID
	}
	return "system"
}

func isPlatformAdmin(r *http.Request) bool {
	info := auth.SessionInfoFromContext(r.Context())
	return info != nil && info.IsPlatformAdmin
}

func tenantScopeCheck(r *http.Request, resourceTenantID string) bool {
	if isPlatformAdmin(r) {
		return true
	}
	return resourceTenantID == tenantID(r)
}

// ==========================================================================
// JSON value extraction helpers
// ==========================================================================

func strVal(m map[string]interface{}, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func intVal(m map[string]interface{}, key string, def int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func int64Val(m map[string]interface{}, key string, def int64) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return def
}

func floatVal(m map[string]interface{}, key string, def float64) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return def
}

func boolVal(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func strSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		if raw, ok := v.(json.RawMessage); ok {
			var arr []string
			if err := json.Unmarshal(raw, &arr); err == nil {
				return arr
			}
			var iarr []interface{}
			if err := json.Unmarshal(raw, &iarr); err == nil {
				out := make([]string, len(iarr))
				for i, e := range iarr {
					out[i] = fmt.Sprint(e)
				}
				return out
			}
			return nil
		}
		if arr, ok := v.([]interface{}); ok {
			out := make([]string, len(arr))
			for i, e := range arr {
				out[i] = fmt.Sprint(e)
			}
			return out
		}
		if s, ok := v.(string); ok {
			return []string{s}
		}
	}
	return nil
}

func stringMap(m map[string]interface{}, key string) map[string]string {
	if v, ok := m[key]; ok {
		if sm, ok := v.(map[string]interface{}); ok {
			out := make(map[string]string)
			for k, val := range sm {
				out[k] = fmt.Sprint(val)
			}
			return out
		}
	}
	return nil
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonField(dm map[string]interface{}, key, defaultJSON string) string {
	v, ok := dm[key]
	if !ok || v == nil {
		return defaultJSON
	}
	return jsonString(v)
}

// ==========================================================================
// Audit and redaction helpers
// ==========================================================================

func sensitiveKeys() []string {
	return []string{
		"KEY", "TOKEN", "PASSWORD", "PASSWD", "PWD",
		"SECRET", "AUTH", "CREDENTIAL", "ACCESS",
		"API_KEY", "APIKEY", "ACCESS_KEY", "SECRET_KEY",
		"AUTHORIZATION", "BEARER",
		"HF_TOKEN", "DASHSCOPE_API_KEY", "OPENAI_API_KEY",
		"AK", "SK", "PRIVATE",
	}
}

func isSensitive(key string) bool {
	upper := strings.ToUpper(key)
	for _, sk := range sensitiveKeys() {
		if strings.Contains(upper, sk) {
			return true
		}
	}
	return false
}

func redactDetailString(s string) string {
	result := s
	for _, sk := range sensitiveKeys() {
		upper := strings.ToUpper(sk)
		lower := strings.ToLower(sk)
		result = strings.ReplaceAll(result, upper, "<redacted>")
		result = strings.ReplaceAll(result, lower, "<redacted>")
	}
	return result
}

func redactEnvMap(env map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range env {
		if isSensitive(k) {
			out[k] = "<redacted>"
		} else {
			out[k] = v
		}
	}
	return out
}

func redactStringMap(env map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range env {
		if isSensitive(k) {
			out[k] = "<redacted>"
		} else {
			out[k] = v
		}
	}
	return out
}
