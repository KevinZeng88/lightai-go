package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// HandleProxyNodeFiles proxies file browsing requests to the agent's /files endpoint.
func (h *AgentHandler) HandleProxyNodeFiles(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	var ip string
	var port int
	h.DB.QueryRow("SELECT primary_ip, metrics_port FROM nodes WHERE id = ?", nodeID).Scan(&ip, &port)
	if port == 0 {
		port = 19091
	}
	agentURL := fmt.Sprintf("http://%s:%d/files?root=%s&path=%s&limit=%s",
		ip, port,
		r.URL.Query().Get("root"),
		r.URL.Query().Get("path"),
		r.URL.Query().Get("limit"))
	resp, err := http.Get(agentURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "agent unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// HandleProxyNodeModelScan proxies model scan requests to the agent's /model-paths/scan endpoint.
func (h *AgentHandler) HandleProxyNodeModelScan(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	var ip string
	var port int
	h.DB.QueryRow("SELECT primary_ip, metrics_port FROM nodes WHERE id = ?", nodeID).Scan(&ip, &port)
	if port == 0 {
		port = 19091
	}
	agentURL := fmt.Sprintf("http://%s:%d/model-paths/scan", ip, port)
	bodyBytes, _ := io.ReadAll(r.Body)
	resp, err := http.Post(agentURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		writeError(w, http.StatusBadGateway, "agent unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
