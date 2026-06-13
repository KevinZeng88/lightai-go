package api

import (
	"encoding/json"
	"net/http"
)

// HandleObservabilityStatus returns Prometheus/Grafana readiness status.
// Used by the Web frontend to avoid cross-origin fetch to P+G directly.
func HandleObservabilityStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"url":   "http://127.0.0.1:19090",
			"ready": probeHTTP("http://127.0.0.1:19090/-/ready"),
		},
		"grafana": map[string]interface{}{
			"url":   "http://127.0.0.1:13000",
			"ready": probeHTTP("http://127.0.0.1:13000/api/health"),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func probeHTTP(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
