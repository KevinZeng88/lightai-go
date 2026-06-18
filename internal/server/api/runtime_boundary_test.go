package api

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lightai-go/internal/server/db"
)

func runtimeBoundaryInsertOnlineNode(t *testing.T, database *db.DB, nodeID string) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES (?, ?, ?, 'online', '', datetime('now'), datetime('now'), datetime('now'))`, nodeID, "agent-"+nodeID, "host-"+nodeID)
	if err != nil {
		t.Fatalf("insert node: %v", err)
	}
}

func TestCreateBackendRuntimeCopiesBackendVersionSnapshot(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	dir := t.TempDir()
	origUserVersionDir := backendCatalogUserVersionsDir
	backendCatalogUserVersionsDir = filepath.Join(dir, "user")
	defer func() { backendCatalogUserVersionsDir = origUserVersionDir }()

	vw := httptest.NewRecorder()
	h.HandleCreateBackendVersion(vw, newReq("POST", "/x",
		`{"id":"backend-version.user.snapshot","version":"snapshot-v1","display_name":"Snapshot V1","default_images_json":{"nvidia":"snapshot:v1"},"default_args_json":["serve"]}`,
		adminSession(), map[string]string{"id": "backend.vllm"}))
	if vw.Code != 201 {
		t.Fatalf("create version code=%d body=%s", vw.Code, vw.Body.String())
	}

	w := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(w, newReq("POST", "/x",
		`{"backend_id":"backend.vllm","backend_version_id":"backend-version.user.snapshot","name":"snapshot-rt","display_name":"Snapshot RT","vendor":"nvidia"}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create code=%d body=%s", w.Code, w.Body.String())
	}
	var rt map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rt); err != nil {
		t.Fatalf("decode runtime: %v", err)
	}
	if rt["source_backend_id"] != "backend.vllm" {
		t.Fatalf("source_backend_id=%v", rt["source_backend_id"])
	}
	if rt["source_backend_version_id"] != "backend-version.user.snapshot" {
		t.Fatalf("source_backend_version_id=%v", rt["source_backend_version_id"])
	}
	if rt["source_version_revision"] == "" {
		t.Fatalf("source_version_revision is empty")
	}
	if rt["image_name"] != "snapshot:v1" {
		t.Fatalf("image_name=%v", rt["image_name"])
	}
	snap := rt["version_snapshot_json"]
	raw, _ := json.Marshal(snap)
	if !strings.Contains(string(raw), "default_args_json") || !strings.Contains(string(raw), "snapshot:v1") {
		t.Fatalf("version snapshot did not include version defaults: %s", string(raw))
	}

	pw := httptest.NewRecorder()
	h.HandlePatchBackendVersion(pw, newReq("PATCH", "/x", `{"default_images_json":{"nvidia":"changed:v2"},"default_args_json":["changed"]}`, adminSession(), map[string]string{"version_id": "backend-version.user.snapshot"}))
	if pw.Code != 200 {
		t.Fatalf("patch version code=%d body=%s", pw.Code, pw.Body.String())
	}

	got := h.getBackendRuntimeJSON(rt["id"].(string))
	if got["image_name"] != "snapshot:v1" {
		t.Fatalf("runtime image changed after BackendVersion edit: %v", got["image_name"])
	}
	raw, _ = json.Marshal(got["version_snapshot_json"])
	if strings.Contains(string(raw), "changed:v2") {
		t.Fatalf("runtime version snapshot changed after BackendVersion edit: %s", string(raw))
	}
}

func TestNodeBackendRuntimeCopiesTemplateSnapshotAndTemplateEditDoesNotChangeIt(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-a")
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-a','node-a','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, "rt-snap", "Runtime Snap", "")

	w := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(w, newReq("POST", "/x", `{"backend_runtime_id":"rt-snap","image_ref":"img:test","image_present":true,"docker_available":true}`, adminSession(), map[string]string{"id": "node-a"}))
	if w.Code != 200 {
		t.Fatalf("enable code=%d body=%s", w.Code, w.Body.String())
	}
	var before string
	if err := db.QueryRow(`SELECT config_snapshot_json FROM node_backend_runtimes WHERE id='node-a:rt-snap'`).Scan(&before); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !strings.Contains(before, "img:test") {
		t.Fatalf("snapshot missing original image: %s", before)
	}

	patch := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(patch, newReq("PATCH", "/x", `{"image_name":"changed:v2","docker_json":{"ipc_mode":"none"}}`, adminSession(), map[string]string{"id": "rt-snap"}))
	if patch.Code != 200 {
		t.Fatalf("patch runtime code=%d body=%s", patch.Code, patch.Body.String())
	}
	var after string
	if err := db.QueryRow(`SELECT config_snapshot_json FROM node_backend_runtimes WHERE id='node-a:rt-snap'`).Scan(&after); err != nil {
		t.Fatalf("read snapshot after: %v", err)
	}
	if before != after {
		t.Fatalf("node runtime snapshot changed after template edit\nbefore=%s\nafter=%s", before, after)
	}
}

func TestNodeBackendRuntimeCheckDoesNotRefreshSnapshot(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-check")
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-check','node-check','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, "rt-check", "Runtime Check", "")

	// 1. Create NodeBackendRuntime via enable (snapshot captured from BackendRuntime).
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-check","image_ref":"img:orig","image_present":true,"docker_available":true}`,
		adminSession(), map[string]string{"id": "node-check"}))
	if ew.Code != 200 {
		t.Fatalf("enable code=%d body=%s", ew.Code, ew.Body.String())
	}

	// 2. Record original snapshot + source tracking fields.
	var origSnapshot, origSourceName, origSourceRevision string
	if err := db.QueryRow(`SELECT COALESCE(config_snapshot_json,'{}'), COALESCE(source_runtime_name,''), COALESCE(source_runtime_revision,'') FROM node_backend_runtimes WHERE id='node-check:rt-check'`).Scan(&origSnapshot, &origSourceName, &origSourceRevision); err != nil {
		t.Fatalf("read nbr: %v", err)
	}
	if !strings.Contains(origSnapshot, "img:test") {
		t.Fatalf("snapshot missing template image: %s", origSnapshot)
	}

	// 3. Modify BackendRuntime template — change image, args, env, docker, health_check.
	pw := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(pw, newReq("PATCH", "/x",
		`{"image_name":"changed:v3","args_override_json":["--changed"],"default_env_json":{"CHANGED":"1"},"docker_json":{"ipc_mode":"none"},"health_check_override_json":{"type":"http","path":"/healthz"}}`,
		adminSession(), map[string]string{"id": "rt-check"}))
	if pw.Code != 200 {
		t.Fatalf("patch runtime code=%d body=%s", pw.Code, pw.Body.String())
	}

	// 4. Run check/validate on NodeBackendRuntime.
	cw := httptest.NewRecorder()
	h.HandleCheckNodeBackendRuntime(cw, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-check","image_ref":"img:orig","image_present":true,"docker_available":true}`,
		adminSession(), map[string]string{"id": "node-check"}))
	if cw.Code != 200 {
		t.Fatalf("check code=%d body=%s", cw.Code, cw.Body.String())
	}

	// 5. Assert config_snapshot_json did NOT change (check must not refresh from template).
	var afterSnapshot, afterSourceName, afterSourceRevision string
	if err := db.QueryRow(`SELECT COALESCE(config_snapshot_json,'{}'), COALESCE(source_runtime_name,''), COALESCE(source_runtime_revision,'') FROM node_backend_runtimes WHERE id='node-check:rt-check'`).Scan(&afterSnapshot, &afterSourceName, &afterSourceRevision); err != nil {
		t.Fatalf("read nbr after check: %v", err)
	}
	if origSnapshot != afterSnapshot {
		t.Fatalf("config_snapshot_json changed after check\nbefore=%s\nafter=%s", origSnapshot, afterSnapshot)
	}
	if afterSnapshot == "" || afterSnapshot == "{}" {
		t.Fatalf("snapshot is empty after check: %s", afterSnapshot)
	}
	if strings.Contains(afterSnapshot, "changed:v3") {
		t.Fatalf("snapshot was refreshed from modified template (contains changed:v3): %s", afterSnapshot)
	}
	if strings.Contains(afterSnapshot, "--changed") {
		t.Fatalf("snapshot was refreshed from modified template (contains --changed): %s", afterSnapshot)
	}

	// 6. Assert source_runtime_name and source_runtime_revision were NOT overwritten.
	if origSourceName != afterSourceName {
		t.Fatalf("source_runtime_name changed after check: %q -> %q", origSourceName, afterSourceName)
	}
	if origSourceRevision != afterSourceRevision {
		t.Fatalf("source_runtime_revision changed after check: %q -> %q", origSourceRevision, afterSourceRevision)
	}

	// 7. Assert check-related fields WERE updated.
	var status, lastChecked string
	if err := db.QueryRow(`SELECT status, last_checked_at FROM node_backend_runtimes WHERE id='node-check:rt-check'`).Scan(&status, &lastChecked); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "ready" {
		t.Fatalf("status=%s, want ready", status)
	}
	if lastChecked == "" {
		t.Fatalf("last_checked_at was not updated")
	}
}

func TestPatchNodeBackendRuntimeSnapshotFieldsNeedRecheck(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-b")
	insertRuntime(t, db, "rt-edit", "Runtime Edit", "")
	if _, err := db.Exec(`INSERT INTO node_backend_runtimes
		(id, backend_runtime_id, node_id, runner_type, image_ref, image_present, docker_available, config_snapshot_json, status, status_reason, tenant_id, created_at, updated_at)
		VALUES ('node-b:rt-edit','rt-edit','node-b','docker','img:v1',1,1,'{}','ready','ok','',datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert nbr: %v", err)
	}
	w := httptest.NewRecorder()
	h.HandlePatchNodeBackendRuntime(w, newReq("PATCH", "/x", `{"config_snapshot_json":{"args_override_json":["--new"]}}`, adminSession(), map[string]string{"nbr_id": "node-b:rt-edit"}))
	if w.Code != 200 {
		t.Fatalf("patch code=%d body=%s", w.Code, w.Body.String())
	}
	var status, snap string
	if err := db.QueryRow(`SELECT status, config_snapshot_json FROM node_backend_runtimes WHERE id='node-b:rt-edit'`).Scan(&status, &snap); err != nil {
		t.Fatalf("read nbr: %v", err)
	}
	if status != "needs_check" {
		t.Fatalf("status=%s, want needs_check", status)
	}
	if !strings.Contains(snap, "--new") {
		t.Fatalf("snapshot not updated: %s", snap)
	}
}

func TestBackendVersionCreatePatchAndReloadUserCatalog(t *testing.T) {
	dir := t.TempDir()
	origUserVersionDir := backendCatalogUserVersionsDir
	backendCatalogUserVersionsDir = filepath.Join(dir, "user")
	defer func() { backendCatalogUserVersionsDir = origUserVersionDir }()

	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	body := `{"version":"user-v1","display_name":"User V1","default_images_json":{"nvidia":"user:v1"},"default_args_json":["serve"],"capabilities_json":{"formats":["huggingface"]},"docker_options_json":{"ipc_mode":"host"},"model_mount_json":{"container_path":"/models","readonly":true},"description":"custom"}`
	w := httptest.NewRecorder()
	h.HandleCreateBackendVersion(w, newReq("POST", "/x", body, adminSession(), map[string]string{"id": "backend.vllm"}))
	if w.Code != 201 {
		t.Fatalf("create version code=%d body=%s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	if created["managed_by"] != "user" {
		t.Fatalf("managed_by=%v", created["managed_by"])
	}
	if created["readonly"] != false {
		t.Fatalf("readonly=%v, want false", created["readonly"])
	}
	if _, err := os.Stat(filepath.Join(backendCatalogUserVersionsDir, "vllm", "user-v1.yaml")); err != nil {
		t.Fatalf("user catalog file missing: %v", err)
	}

	pw := httptest.NewRecorder()
	h.HandlePatchBackendVersion(pw, newReq("PATCH", "/x", `{"display_name":"User V1 patched","default_images_json":{"nvidia":"user:v2"}}`, adminSession(), map[string]string{"version_id": created["id"].(string)}))
	if pw.Code != 200 {
		t.Fatalf("patch version code=%d body=%s", pw.Code, pw.Body.String())
	}

	rw := httptest.NewRecorder()
	h.HandleReloadBackendCatalog(rw, newReq("POST", "/x", "", adminSession(), nil))
	if rw.Code != 200 {
		t.Fatalf("reload code=%d body=%s", rw.Code, rw.Body.String())
	}
	var img string
	if err := db.QueryRow(`SELECT default_images_json FROM backend_versions WHERE id=?`, created["id"]).Scan(&img); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if !strings.Contains(img, "user:v2") {
		t.Fatalf("patched version not persisted/reloaded: %s", img)
	}
	fileData, err := os.ReadFile(filepath.Join(backendCatalogUserVersionsDir, "vllm", "user-v1.yaml"))
	if err != nil {
		t.Fatalf("read user catalog file: %v", err)
	}
	if !strings.Contains(string(fileData), "user:v2") {
		t.Fatalf("patched user catalog file missing new image: %s", string(fileData))
	}
	var loadedFrom, configHash string
	if err := db.QueryRow(`SELECT loaded_from, config_hash FROM backend_versions WHERE id=?`, created["id"]).Scan(&loadedFrom, &configHash); err != nil {
		t.Fatalf("read projection metadata: %v", err)
	}
	if loadedFrom == "" || configHash == "" {
		t.Fatalf("projection metadata missing loaded_from=%q config_hash=%q", loadedFrom, configHash)
	}
}

func TestSystemBackendVersionReadOnlyAndCloneable(t *testing.T) {
	dir := t.TempDir()
	origUserVersionDir := backendCatalogUserVersionsDir
	backendCatalogUserVersionsDir = filepath.Join(dir, "user")
	defer func() { backendCatalogUserVersionsDir = origUserVersionDir }()

	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	patch := httptest.NewRecorder()
	h.HandlePatchBackendVersion(patch, newReq("PATCH", "/x", `{"display_name":"bad"}`, adminSession(), map[string]string{"version_id": "vllm-v0.23.0"}))
	if patch.Code != 409 {
		t.Fatalf("system patch code=%d body=%s", patch.Code, patch.Body.String())
	}

	clone := httptest.NewRecorder()
	h.HandleCloneBackendVersion(clone, newReq("POST", "/x", "", adminSession(), map[string]string{"version_id": "vllm-v0.23.0"}))
	if clone.Code != 201 {
		t.Fatalf("clone code=%d body=%s", clone.Code, clone.Body.String())
	}
	var cloned map[string]interface{}
	json.Unmarshal(clone.Body.Bytes(), &cloned)
	if cloned["managed_by"] != "user" || cloned["readonly"] != false {
		t.Fatalf("clone mutability = managed_by:%v readonly:%v", cloned["managed_by"], cloned["readonly"])
	}
	loadedFrom, _ := cloned["loaded_from"].(string)
	if loadedFrom == "" {
		t.Fatalf("clone did not reload from user catalog file: %#v", cloned)
	}
	if _, err := os.Stat(loadedFrom); err != nil {
		t.Fatalf("clone catalog file missing: %v", err)
	}
}

func TestBackendCatalogReloadLoadsSystemAndUserFilesWithoutMutatingRuntimeSnapshots(t *testing.T) {
	dir := t.TempDir()
	systemDir := filepath.Join(dir, "system")
	userDir := filepath.Join(dir, "user")
	if err := os.MkdirAll(filepath.Join(systemDir, "vllm"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(userDir, "vllm"), 0755); err != nil {
		t.Fatal(err)
	}
	systemFile := filepath.Join(systemDir, "vllm", "sys.yaml")
	userFile := filepath.Join(userDir, "vllm", "user.yaml")
	if err := os.WriteFile(systemFile, []byte(`id: test-system-v1
backend_id: backend.vllm
version: sys-v1
source: system
readonly: true
protocol: openai-compatible
image_candidates:
  - sys:v1
default_port: 8000
default_host: 0.0.0.0
default_model_mount:
  container_path: /models
  readonly: true
default_endpoints:
  models: /v1/models
capabilities:
  - models
health_check:
  type: http
  path: /v1/models
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte(`id: test-user-v1
backend_id: backend.vllm
version: user-file-v1
source: user
readonly: false
image_candidates:
  - user:file-v1
default_port: 8001
`), 0644); err != nil {
		t.Fatal(err)
	}
	origSystemDir := backendCatalogSystemVersionsDir
	origUserDir := backendCatalogUserVersionsDir
	backendCatalogSystemVersionsDir = systemDir
	backendCatalogUserVersionsDir = userDir
	defer func() {
		backendCatalogSystemVersionsDir = origSystemDir
		backendCatalogUserVersionsDir = origUserDir
	}()

	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	rw := httptest.NewRecorder()
	h.HandleReloadBackendCatalog(rw, newReq("POST", "/x", "", adminSession(), nil))
	if rw.Code != 200 {
		t.Fatalf("reload code=%d body=%s", rw.Code, rw.Body.String())
	}
	var sysReadonly int
	if err := db.QueryRow(`SELECT readonly FROM backend_versions WHERE id='test-system-v1'`).Scan(&sysReadonly); err != nil {
		t.Fatalf("system catalog not loaded: %v", err)
	}
	if sysReadonly != 1 {
		t.Fatalf("system readonly=%d, want 1", sysReadonly)
	}
	var userReadonly int
	if err := db.QueryRow(`SELECT readonly FROM backend_versions WHERE id='test-user-v1'`).Scan(&userReadonly); err != nil {
		t.Fatalf("user catalog not loaded: %v", err)
	}
	if userReadonly != 0 {
		t.Fatalf("user readonly=%d, want 0", userReadonly)
	}

	w := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(w, newReq("POST", "/x",
		`{"backend_id":"backend.vllm","backend_version_id":"test-user-v1","name":"reload-snapshot-rt","display_name":"Reload Snapshot RT"}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create runtime code=%d body=%s", w.Code, w.Body.String())
	}
	var rt map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rt); err != nil {
		t.Fatalf("decode runtime: %v", err)
	}
	if err := os.WriteFile(userFile, []byte(`id: test-user-v1
backend_id: backend.vllm
version: user-file-v1
source: user
readonly: false
image_candidates:
  - user:file-v2
default_port: 8001
`), 0644); err != nil {
		t.Fatal(err)
	}
	h.HandleReloadBackendCatalog(httptest.NewRecorder(), newReq("POST", "/x", "", adminSession(), nil))
	got := h.getBackendRuntimeJSON(rt["id"].(string))
	raw, _ := json.Marshal(got["version_snapshot_json"])
	if strings.Contains(string(raw), "user:file-v2") {
		t.Fatalf("reload mutated BackendRuntime snapshot: %s", string(raw))
	}
}

func TestBackendVersionCatalogIsSoftwareOnly(t *testing.T) {
	db := setupTestDB(t)
	rows, err := db.Query(`SELECT id, default_images_json, image_candidates_json, capabilities_json, docker_options_json, model_mount_json, vendor_options_json FROM backend_versions WHERE managed_by='system' AND status != 'deprecated'`)
	if err != nil {
		t.Fatalf("query versions: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		fields := make([]string, 6)
		if err := rows.Scan(&id, &fields[0], &fields[1], &fields[2], &fields[3], &fields[4], &fields[5]); err != nil {
			t.Fatalf("scan: %v", err)
		}
		joined := strings.ToLower(strings.Join(fields, "\n"))
		for _, forbidden := range []string{"node_id", "image_present", "needs_check", "ready", "cuda_visible_devices", "--gpus", "/dev/dri", "/dev/mxcd", "/dev/infiniband", "/usr/local/ascend", "host_path"} {
			if strings.Contains(joined, forbidden) {
				t.Fatalf("system BackendVersion %s contains hardware/node field %q in %s", id, forbidden, joined)
			}
		}
	}
}

func TestBackendRuntimeListShowsTemplatesWithNodeAggregatesOnly(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-list")
	insertRuntime(t, db, "rt-list", "Runtime List", "")
	if _, err := db.Exec(`INSERT INTO node_backend_runtimes
		(id, backend_runtime_id, node_id, runner_type, image_ref, image_present, docker_available, config_snapshot_json, status, status_reason, tenant_id, created_at, updated_at)
		VALUES ('node-list:rt-list','rt-list','node-list','docker','img:v1',1,1,'{}','ready','ok','',datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert nbr: %v", err)
	}
	w := httptest.NewRecorder()
	h.HandleListBackendRuntimes(w, newReq("GET", "/x", "", adminSession(), nil))
	if w.Code != 200 {
		t.Fatalf("list code=%d body=%s", w.Code, w.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	for _, item := range list {
		if item["id"] == "node-list:rt-list" {
			t.Fatalf("BackendRuntime list leaked NodeBackendRuntime row: %v", item)
		}
		if item["id"] == "rt-list" {
			if item["node_count"].(float64) != 1 || item["ready_count"].(float64) != 1 {
				t.Fatalf("aggregate counts = %v/%v", item["node_count"], item["ready_count"])
			}
			return
		}
	}
	t.Fatalf("runtime rt-list missing from list")
}
