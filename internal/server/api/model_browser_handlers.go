package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
)

// HandleListNodeModelBrowserRoots returns the dynamic extra roots for a node.
func (h *AgentHandler) HandleListNodeModelBrowserRoots(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	var extraJSON string
	if err := h.DB.QueryRow(`SELECT model_browser_extra_roots FROM nodes WHERE id = ?`, nodeID).Scan(&extraJSON); err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	var extra []string
	if extraJSON != "" {
		json.Unmarshal([]byte(extraJSON), &extra)
	}
	if extra == nil {
		extra = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"extra_roots": extra})
}

// HandleAddNodeModelBrowserRoot adds a dynamic root to a node.
func (h *AgentHandler) HandleAddNodeModelBrowserRoot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	newRoot := strings.TrimSpace(strVal(req, "root", ""))
	if newRoot == "" {
		writeError(w, http.StatusBadRequest, "root is required")
		return
	}
	if !strings.HasPrefix(newRoot, "/") {
		writeError(w, http.StatusBadRequest, "root must be an absolute path")
		return
	}
	clean := filepath.Clean(newRoot)
	if clean != newRoot && clean+"/" != newRoot {
		writeError(w, http.StatusBadRequest, "root path must be clean (no .. or .)")
		return
	}
	forbidden := []string{"/etc", "/root", "/proc", "/sys", "/dev", "/run", "/var/run", "/boot", "/lost+found"}
	for _, fb := range forbidden {
		if clean == fb || strings.HasPrefix(clean, fb+"/") {
			writeError(w, http.StatusBadRequest, "root path is in a forbidden system directory")
			return
		}
	}
	var extraJSON string
	h.DB.QueryRow(`SELECT model_browser_extra_roots FROM nodes WHERE id = ?`, nodeID).Scan(&extraJSON)
	var extra []string
	if extraJSON != "" {
		json.Unmarshal([]byte(extraJSON), &extra)
	}
	for _, e := range extra {
		if e == newRoot {
			writeJSON(w, http.StatusOK, map[string]interface{}{"extra_roots": extra, "status": "already_exists"})
			return
		}
	}
	extra = append(extra, newRoot)
	b, _ := json.Marshal(extra)
	h.DB.Exec(`UPDATE nodes SET model_browser_extra_roots = ? WHERE id = ?`, string(b), nodeID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"extra_roots": extra, "status": "added"})
}

// HandleDeleteNodeModelBrowserRoot removes a dynamic root from a node.
func (h *AgentHandler) HandleDeleteNodeModelBrowserRoot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	delRoot := r.URL.Query().Get("root")
	if delRoot == "" {
		writeError(w, http.StatusBadRequest, "root query param is required")
		return
	}
	var extraJSON string
	h.DB.QueryRow(`SELECT model_browser_extra_roots FROM nodes WHERE id = ?`, nodeID).Scan(&extraJSON)
	var extra []string
	if extraJSON != "" {
		json.Unmarshal([]byte(extraJSON), &extra)
	}
	var newExtra []string
	for _, e := range extra {
		if e != delRoot {
			newExtra = append(newExtra, e)
		}
	}
	b, _ := json.Marshal(newExtra)
	h.DB.Exec(`UPDATE nodes SET model_browser_extra_roots = ? WHERE id = ?`, string(b), nodeID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"extra_roots": newExtra, "status": "removed"})
}
