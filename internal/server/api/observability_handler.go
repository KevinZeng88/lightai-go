package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// HandleObservabilityStatus returns Prometheus/Grafana readiness status.
// Internal probes always use 127.0.0.1. External URLs use request Host.
func HandleObservabilityStatus(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	// Strip port if present, we'll add the observability ports.
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}
	if host == "" {
		host = "127.0.0.1"
	}

	status := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"url":   "http://" + host + ":19090",
			"ready": probeHTTP("http://127.0.0.1:19090/-/ready"),
		},
		"grafana": map[string]interface{}{
			"url":   "http://" + host + ":13000",
			"ready": probeHTTP("http://127.0.0.1:13000/api/health"),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

var probeClient = &http.Client{Timeout: 5 * time.Second}

func probeHTTP(url string) bool {
	resp, err := probeClient.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
