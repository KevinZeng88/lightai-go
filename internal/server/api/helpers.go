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
	writeJSON(w, status, map[string]string{
		"error":   msg,
		"code":    apiErrorCode(status, msg),
		"message": msg,
	})
}

func apiErrorCode(status int, msg string) string {
	normalized := strings.ToLower(strings.TrimSpace(msg))
	switch {
	case strings.Contains(normalized, "display_name already exists"):
		return "display_name_exists"
	case strings.Contains(normalized, "already exists"):
		return "already_exists"
	case strings.Contains(normalized, "invalid request"):
		return "invalid_request"
	case strings.Contains(normalized, "not found"):
		return "not_found"
	case strings.Contains(normalized, "unauthorized"):
		return "unauthorized"
	case strings.Contains(normalized, "permission"):
		return "permission_denied"
	case strings.Contains(normalized, "read-only") || strings.Contains(normalized, "readonly"):
		return "readonly"
	case strings.Contains(normalized, "required"):
		return "required"
	case strings.Contains(normalized, "agent unreachable"):
		return "agent_unreachable"
	case strings.Contains(normalized, "node backend runtime is not deployable"):
		return "node_runtime_not_deployable"
	case strings.Contains(normalized, "model_artifact_id not found"):
		return "model_artifact_not_found"
	case strings.Contains(normalized, "node_backend_runtime_id not found"):
		return "node_runtime_not_found"
	case strings.Contains(normalized, "model root already exists"):
		return "model_root_exists"
	case strings.Contains(normalized, "root not allowed"):
		return "root_not_allowed"
	case strings.Contains(normalized, "node runtime is used by deployments"):
		return "node_runtime_in_use"
	case strings.Contains(normalized, "node runtime is used by active instances"):
		return "node_runtime_active_instances"
	case status == http.StatusConflict:
		return "conflict"
	case status == http.StatusBadRequest:
		return "bad_request"
	case status == http.StatusForbidden:
		return "forbidden"
	case status == http.StatusNotFound:
		return "not_found"
	case status >= 500:
		return "server_error"
	default:
		return "request_failed"
	}
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

// rawJSONBytes returns the raw JSON bytes for a value that may be a DB string,
// json.RawMessage, or already-parsed map/slice. Avoids double-encoding that
// json.Marshal would produce on a string (which adds escape quotes).
func rawJSONBytes(v interface{}) []byte {
	switch raw := v.(type) {
	case json.RawMessage:
		return []byte(raw)
	case string:
		return []byte(raw)
	case []byte:
		return raw
	default:
		b, _ := json.Marshal(v)
		return b
	}
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
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return s
	}
	redacted := false
	for key := range m {
		if isSensitiveKey(key) {
			m[key] = "<redacted>"
			redacted = true
		}
	}
	if !redacted {
		return s
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	fragments := []string{"KEY", "TOKEN", "PASSWORD", "SECRET", "AUTH", "CREDENTIAL", "ACCESS"}
	for _, f := range fragments {
		if strings.Contains(upper, f) {
			return true
		}
	}
	return false
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
