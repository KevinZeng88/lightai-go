package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"lightai-go/internal/server/authz"
)

// HandleProxyNodeFiles proxies file browsing requests to the agent's /files endpoint.
func (h *AgentHandler) HandleProxyNodeFiles(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	if !authz.CheckNodeTenant(r, h.DB.DB, nodeID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var ip string
	var port int
	h.DB.QueryRow("SELECT primary_ip, metrics_port FROM nodes WHERE id = ?", nodeID).Scan(&ip, &port)
	if port == 0 {
		port = 19091
	}
	rootID := r.URL.Query().Get("root_id")
	rootPath := r.URL.Query().Get("root")
	if rootID == "" && rootPath == "" {
		roots, err := h.listNodeModelRoots(nodeID, false)
		if err != nil {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}
		out := []map[string]interface{}{}
		for _, root := range roots {
			out = append(out, root.jsonMap())
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"allowed_roots": out, "entries": []map[string]interface{}{}})
		return
	}
	root, err := h.resolveNodeModelRoot(nodeID, rootID, rootPath)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"entries": []map[string]interface{}{}, "error": "root_not_allowed"})
		return
	}
	rel, err := safeRelativePath(r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"entries": []map[string]interface{}{}, "error": "path traversal blocked"})
		return
	}
	q := url.Values{}
	q.Set("root", root.Path)
	q.Set("path", rel)
	q.Set("limit", r.URL.Query().Get("limit"))
	q.Set("extra_roots", root.Path)
	ac := h.requireAgentClient(w)
	if ac == nil {
		return
	}
	body, statusCode, err := ac.GetJSON(r.Context(), ip, port, "/files", q)
	if err != nil {
		writeError(w, http.StatusBadGateway, "agent unreachable: "+err.Error())
		return
	}
	if statusCode >= 200 && statusCode < 300 {
		out := map[string]interface{}{}
		if json.Unmarshal(body, &out) == nil {
			out["root_id"] = root.ID
			out["root"] = root.Path
			out["model_root"] = root.Path
			out["relative_path"] = rel
			out["absolute_path"] = root.Path
			if rel != "" {
				out["absolute_path"] = root.Path + "/" + rel
			}
			body, _ = json.Marshal(out)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(body)
}

// HandleProxyNodeModelScan proxies model scan requests to the agent's /model-paths/scan endpoint.
func (h *AgentHandler) HandleProxyNodeModelScan(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	if !authz.CheckNodeTenant(r, h.DB.DB, nodeID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var ip string
	var port int
	h.DB.QueryRow("SELECT primary_ip, metrics_port FROM nodes WHERE id = ?", nodeID).Scan(&ip, &port)
	if port == 0 {
		port = 19091
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	reqMap := map[string]interface{}{}
	if err := json.Unmarshal(bodyBytes, &reqMap); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	root, err := h.resolveNodeModelRoot(nodeID, strVal(reqMap, "root_id", ""), strVal(reqMap, "root", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, "root not allowed")
		return
	}
	rel, err := safeRelativePath(strVal(reqMap, "relative_path", ""))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	reqMap["root"] = root.Path
	reqMap["root_id"] = root.ID
	reqMap["relative_path"] = rel
	bodyBytes, _ = json.Marshal(reqMap)
	q := url.Values{}
	q.Set("extra_roots", root.Path)
	ac := h.requireAgentClient(w)
	if ac == nil {
		return
	}
	body, statusCode, err := ac.PostJSON(r.Context(), ip, port, "/model-paths/scan", bytes.NewReader(bodyBytes), q)
	if err != nil {
		writeError(w, http.StatusBadGateway, "agent unreachable: "+err.Error())
		return
	}
	if statusCode >= 200 && statusCode < 300 {
		out := map[string]interface{}{}
		if json.Unmarshal(body, &out) == nil {
			out["root_id"] = root.ID
			out["root"] = root.Path
			out["model_root"] = root.Path
			out["scan_root"] = root.Path
			// Compute the canonical server-side absolute path from the validated root + relative path.
			// The agent may return its own paths; for top-level we trust the server's resolution.
			out["relative_path"] = rel
			out["absolute_path"] = root.Path
			if rel != "" {
				out["absolute_path"] = root.Path + "/" + rel
			}
			// For candidate-based responses, preserve each candidate's own specific path
			// (e.g., a .gguf file within a directory). This is essential for llama.cpp
			// which needs the exact file path in -m, not the parent directory (WEB-AI-LW-003).
			// The agent's scanner sets candidate.path to the discovered file/directory.
			body, _ = json.Marshal(out)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(body)
}
