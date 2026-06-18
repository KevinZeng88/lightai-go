package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lightai-go/internal/common/log"

	"github.com/google/uuid"
)

var defaultDeniedModelRoots = []string{
	"/", "/etc", "/root", "/boot", "/proc", "/sys", "/dev", "/run", "/var/run", "/var/lib/docker",
}

func normalizeAllowedModelRoot(path string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("model root is required")
	}
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("model root must be an absolute path")
	}
	for _, denied := range defaultDeniedModelRoots {
		if pathWithinRoot(clean, denied) {
			return "", fmt.Errorf("model root is not allowed")
		}
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("model root does not exist")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("model root must be a directory")
	}
	if realPath, err := filepath.EvalSymlinks(clean); err == nil {
		realClean := filepath.Clean(realPath)
		for _, denied := range defaultDeniedModelRoots {
			if pathWithinRoot(realClean, denied) {
				return "", fmt.Errorf("model root is not allowed")
			}
		}
	}
	return clean, nil
}

func pathWithinRoot(path, root string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	if cleanRoot == string(os.PathSeparator) {
		return cleanPath == cleanRoot
	}
	return cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator))
}

func safeRelativePath(value string) (string, error) {
	rel := filepath.Clean(strings.TrimSpace(value))
	if rel == "." || rel == "" {
		return "", nil
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return "", fmt.Errorf("path traversal blocked")
	}
	return rel, nil
}

type nodeModelRoot struct {
	ID            string         `json:"id"`
	NodeID        string         `json:"node_id"`
	Path          string         `json:"path"`
	Status        string         `json:"status"`
	Source        string         `json:"source"`
	Description   string         `json:"description"`
	CreatedBy     string         `json:"created_by"`
	TenantID      string         `json:"tenant_id"`
	LastCheckedAt sql.NullString `json:"-"`
	LastError     string         `json:"last_error"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

func (r nodeModelRoot) jsonMap() map[string]interface{} {
	out := map[string]interface{}{
		"id":          r.ID,
		"node_id":     r.NodeID,
		"path":        r.Path,
		"root":        r.Path,
		"label":       r.Path,
		"status":      r.Status,
		"source":      r.Source,
		"description": r.Description,
		"created_by":  r.CreatedBy,
		"tenant_id":   r.TenantID,
		"last_error":  r.LastError,
		"created_at":  r.CreatedAt,
		"updated_at":  r.UpdatedAt,
	}
	if r.LastCheckedAt.Valid {
		out["last_checked_at"] = r.LastCheckedAt.String
	} else {
		out["last_checked_at"] = ""
	}
	return out
}

func (h *AgentHandler) listNodeModelRoots(nodeID string, includeDisabled bool) ([]nodeModelRoot, error) {
	q := `SELECT id, node_id, path, status, source, description, created_by, tenant_id, last_checked_at, last_error, created_at, updated_at
		FROM node_model_roots WHERE node_id = ?`
	args := []interface{}{nodeID}
	if !includeDisabled {
		q += ` AND status = 'enabled'`
	}
	q += ` ORDER BY path`
	rows, err := h.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []nodeModelRoot{}
	for rows.Next() {
		var r nodeModelRoot
		if err := rows.Scan(&r.ID, &r.NodeID, &r.Path, &r.Status, &r.Source, &r.Description, &r.CreatedBy, &r.TenantID, &r.LastCheckedAt, &r.LastError, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (h *AgentHandler) resolveNodeModelRoot(nodeID, rootID, rootPath string) (nodeModelRoot, error) {
	var r nodeModelRoot
	base := `SELECT id, node_id, path, status, source, description, created_by, tenant_id, last_checked_at, last_error, created_at, updated_at
		FROM node_model_roots WHERE node_id = ? AND status = 'enabled'`
	var row *sql.Row
	if rootID != "" {
		row = h.DB.QueryRow(base+` AND id = ?`, nodeID, rootID)
	} else {
		clean := filepath.Clean(strings.TrimSpace(rootPath))
		row = h.DB.QueryRow(base+` AND path = ?`, nodeID, clean)
	}
	if err := row.Scan(&r.ID, &r.NodeID, &r.Path, &r.Status, &r.Source, &r.Description, &r.CreatedBy, &r.TenantID, &r.LastCheckedAt, &r.LastError, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return r, fmt.Errorf("root not allowed")
	}
	return r, nil
}

func (h *AgentHandler) nodeTenant(nodeID string) (string, error) {
	var tid string
	if err := h.DB.QueryRow(`SELECT tenant_id FROM nodes WHERE id = ?`, nodeID).Scan(&tid); err != nil {
		return "", err
	}
	return tid, nil
}

// HandleListNodeModelRoots returns persisted allowed model roots for a node.
func (h *AgentHandler) HandleListNodeModelRoots(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	includeDisabled := r.URL.Query().Get("include_disabled") == "true"
	if _, err := h.nodeTenant(nodeID); err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	roots, err := h.listNodeModelRoots(nodeID, includeDisabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := []map[string]interface{}{}
	for _, root := range roots {
		out = append(out, root.jsonMap())
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleAddNodeModelRoot adds a persisted allowed model root for a node.
func (h *AgentHandler) HandleAddNodeModelRoot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	tid, err := h.nodeTenant(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	rawPath := strVal(req, "path", strVal(req, "root", ""))
	clean, err := normalizeAllowedModelRoot(rawPath)
	if err != nil {
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{TenantID: tid, ActorID: actorIDFromSession(r), Action: "node_model_root.add", ResourceType: "node", ResourceID: nodeID, Result: "failure", Error: err.Error(), RequestID: log.RequestIDFromContext(r.Context())})
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	id := uuid.NewString()
	actorID := actorIDFromSession(r)
	_, err = h.DB.Exec(`INSERT INTO node_model_roots
		(id, node_id, path, status, source, description, created_by, tenant_id, last_checked_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		id, nodeID, clean, "enabled", strVal(req, "source", "user"), strVal(req, "description", ""), actorID, tid, now, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			var existingID string
			if scanErr := h.DB.QueryRow(`SELECT id FROM node_model_roots WHERE node_id = ? AND path = ?`, nodeID, clean).Scan(&existingID); scanErr == nil {
				if _, updateErr := h.DB.Exec(`UPDATE node_model_roots SET status = 'enabled', description = ?, updated_at = ? WHERE id = ?`,
					strVal(req, "description", ""), now, existingID); updateErr != nil {
					writeError(w, http.StatusInternalServerError, "internal error")
					return
				}
				root, _ := h.resolveNodeModelRoot(nodeID, existingID, "")
				writeJSON(w, http.StatusOK, root.jsonMap())
				return
			}
			writeError(w, http.StatusConflict, "model root already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	WriteAudit(r.Context(), h.DB.DB, AuditEntry{TenantID: tid, ActorID: actorID, Action: "node_model_root.add", ResourceType: "node_model_root", ResourceID: id, Result: "success", Detail: "node_id=" + nodeID + " path=" + clean, RequestID: log.RequestIDFromContext(r.Context())})
	root, _ := h.resolveNodeModelRoot(nodeID, id, "")
	writeJSON(w, http.StatusCreated, root.jsonMap())
}

func (h *AgentHandler) HandlePatchNodeModelRoot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	rootID := r.PathValue("root_id")
	root, err := h.resolveNodeModelRoot(nodeID, rootID, "")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	status := strVal(req, "status", root.Status)
	if status != "enabled" && status != "disabled" {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := h.DB.Exec(`UPDATE node_model_roots SET status = ?, description = ?, updated_at = ? WHERE id = ?`,
		status, strVal(req, "description", root.Description), now, rootID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	WriteAudit(r.Context(), h.DB.DB, AuditEntry{TenantID: root.TenantID, ActorID: actorIDFromSession(r), Action: "node_model_root.patch", ResourceType: "node_model_root", ResourceID: rootID, Result: "success", Detail: "status=" + status, RequestID: log.RequestIDFromContext(r.Context())})
	updated, _ := h.resolveNodeModelRoot(nodeID, rootID, "")
	writeJSON(w, http.StatusOK, updated.jsonMap())
}

func (h *AgentHandler) HandleDeleteNodeModelRoot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	rootID := r.PathValue("root_id")
	root, err := h.resolveNodeModelRoot(nodeID, rootID, "")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var refs int
	h.DB.QueryRow(`SELECT COUNT(*) FROM model_locations WHERE node_id = ? AND model_root = ?`, nodeID, root.Path).Scan(&refs)
	if refs > 0 {
		writeError(w, http.StatusConflict, "model root is still referenced by model locations")
		return
	}
	if root.Source == "config" {
		if _, err := h.DB.Exec(`UPDATE node_model_roots SET status = 'disabled', updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), rootID); err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	} else if _, err := h.DB.Exec(`DELETE FROM node_model_roots WHERE id = ?`, rootID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	WriteAudit(r.Context(), h.DB.DB, AuditEntry{TenantID: root.TenantID, ActorID: actorIDFromSession(r), Action: "node_model_root.delete", ResourceType: "node_model_root", ResourceID: rootID, Result: "success", Detail: "path=" + root.Path, RequestID: log.RequestIDFromContext(r.Context())})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Legacy compatibility wrappers for /model-browser/roots.
func (h *AgentHandler) HandleListNodeModelBrowserRoots(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	roots, err := h.listNodeModelRoots(nodeID, false)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	extra := []string{}
	for _, root := range roots {
		extra = append(extra, root.Path)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"extra_roots": extra})
}

func (h *AgentHandler) HandleAddNodeModelBrowserRoot(w http.ResponseWriter, r *http.Request) {
	h.HandleAddNodeModelRoot(w, r)
}

func (h *AgentHandler) HandleDeleteNodeModelBrowserRoot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	rootPath := r.URL.Query().Get("root")
	root, err := h.resolveNodeModelRoot(nodeID, "", rootPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	r.SetPathValue("root_id", root.ID)
	h.HandleDeleteNodeModelRoot(w, r)
}
