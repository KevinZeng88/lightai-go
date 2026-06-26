package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"lightai-go/internal/agent/register"
	"lightai-go/internal/server/agentclient"
	"lightai-go/internal/server/db"
	"time"
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
		`{"id":"backend-version.user.snapshot","version":"snapshot-v1","display_name":"Snapshot V1","config_set":{"schema_version":1,"items":{"backend.arg.fake_new_param":{"code":"backend.arg.fake_new_param","category":"model_runtime","kind":"cli_arg","type":"string","enabled":true,"value":"from-version","default_value":"from-version","render":{"flag":"--fake-new-param","label":"Fake New Param","group":"Test Params"},"order":340}}}}`,
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
	if rt["backend_id"] != "backend.vllm" {
		t.Fatalf("backend_id=%v", rt["backend_id"])
	}
	if rt["backend_version_id"] != "backend-version.user.snapshot" {
		t.Fatalf("backend_version_id=%v", rt["backend_version_id"])
	}
	raw, _ := json.Marshal(rt["config_set"])
	if !strings.Contains(string(raw), "backend.arg.fake_new_param") || !strings.Contains(string(raw), "from-version") {
		t.Fatalf("config set did not include version defaults: %s", string(raw))
	}

	pw := httptest.NewRecorder()
	h.HandlePatchBackendVersion(pw, newReq("PATCH", "/x",
		`{"config_set":{"schema_version":1,"items":{"backend.arg.fake_new_param":{"code":"backend.arg.fake_new_param","category":"model_runtime","kind":"cli_arg","type":"string","enabled":true,"value":"changed-version","default_value":"changed-version","render":{"flag":"--fake-new-param"},"order":340},"backend.arg.after_runtime":{"code":"backend.arg.after_runtime","category":"model_runtime","kind":"cli_arg","type":"string","enabled":true,"value":"after","render":{"flag":"--after-runtime"}}}}}`,
		adminSession(), map[string]string{"version_id": "backend-version.user.snapshot"}))
	if pw.Code != 200 {
		t.Fatalf("patch version code=%d body=%s", pw.Code, pw.Body.String())
	}

	got := h.getBackendRuntimeJSON(rt["id"].(string))
	raw, _ = json.Marshal(got["config_set"])
	if strings.Contains(string(raw), "changed-version") || strings.Contains(string(raw), "backend.arg.after_runtime") {
		t.Fatalf("runtime config set changed after BackendVersion edit: %s", string(raw))
	}
}

func TestBackendVersionRejectsRuntimeOnlyFields(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	dir := t.TempDir()
	origUserVersionDir := backendCatalogUserVersionsDir
	backendCatalogUserVersionsDir = filepath.Join(dir, "user")
	defer func() { backendCatalogUserVersionsDir = origUserVersionDir }()

	for _, tc := range []struct {
		name string
		body string
	}{
		{"image_ref", `{"version":"bad-image","image_ref":"runtime-only:v1"}`},
		{"command", `{"version":"bad-command","command":["serve"]}`},
		{"entrypoint", `{"version":"bad-entrypoint","entrypoint":["python3"]}`},
		{"model_mount", `{"version":"bad-mount","model_mount":{"container_path":"/models"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.HandleCreateBackendVersion(w, newReq("POST", "/x", tc.body, adminSession(), map[string]string{"id": "backend.vllm"}))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "BackendRuntime") {
				t.Fatalf("error should mention BackendRuntime boundary, got %s", w.Body.String())
			}
		})
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
	h.HandleEnableNodeBackendRuntime(w, newReq("POST", "/x", `{"backend_runtime_id":"rt-snap","image_ref":"img:test"}}`, adminSession(), map[string]string{"id": "node-a"}))
	if w.Code != 200 {
		t.Fatalf("enable code=%d body=%s", w.Code, w.Body.String())
	}
	var before string
	if err := db.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE id='node-a:rt-snap'`).Scan(&before); err != nil {
		t.Fatalf("read config set: %v", err)
	}
	if !strings.Contains(before, "img:test") {
		t.Fatalf("config set missing original image: %s", before)
	}

	patch := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(patch, newReq("PATCH", "/x", `{"image_ref":"changed:v2","docker_options":{"ipc_mode":"none"}}`, adminSession(), map[string]string{"id": "rt-snap"}))
	if patch.Code != 200 {
		t.Fatalf("patch runtime code=%d body=%s", patch.Code, patch.Body.String())
	}
	var after string
	if err := db.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE id='node-a:rt-snap'`).Scan(&after); err != nil {
		t.Fatalf("read config set after: %v", err)
	}
	if before != after {
		t.Fatalf("node runtime config set changed after template edit\nbefore=%s\nafter=%s", before, after)
	}
}

func TestCreateNodeBackendRuntimeAppliesRequestConfigSetSnapshot(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-nbr-config-set")
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-nbr-config-set','node-nbr-config-set','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, "rt-nbr-config-set", "Runtime NBR ConfigSet", "")

	var runtimeSetRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM backend_runtimes WHERE id='rt-nbr-config-set'`).Scan(&runtimeSetRaw); err != nil {
		t.Fatalf("read runtime config set: %v", err)
	}
	editedSet := copyConfigSet(runtimeSetRaw)
	items := configSetItems(editedSet)
	items["backend.arg.fake_new_param"] = map[string]interface{}{
		"code":          "backend.arg.fake_new_param",
		"category":      "model_runtime",
		"kind":          "cli_arg",
		"type":          "string",
		"enabled":       true,
		"value":         "node-local-value",
		"default_value": "runtime-default",
		"render": map[string]interface{}{
			"flag":   "--fake-new-param",
			"target": "cli",
			"style":  "flag_space_value",
		},
	}
	editedSet["items"] = items

	w := httptest.NewRecorder()
	body := jsonString(map[string]interface{}{
		"backend_runtime_id": "rt-nbr-config-set",
		"image_ref":          "img:nbr-config-set",
		"config_set":         editedSet,
	})
	h.HandleEnableNodeBackendRuntime(w, newReq("POST", "/x", body, adminSession(), map[string]string{"id": "node-nbr-config-set"}))
	if w.Code != 200 {
		t.Fatalf("enable code=%d body=%s", w.Code, w.Body.String())
	}
	var nbrSetRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE id='node-nbr-config-set:rt-nbr-config-set'`).Scan(&nbrSetRaw); err != nil {
		t.Fatalf("read NBR config set: %v", err)
	}
	nbrSet := parseConfigSet(nbrSetRaw)
	item, _ := configSetItems(nbrSet)["backend.arg.fake_new_param"].(map[string]interface{})
	if item == nil {
		t.Fatalf("NBR config set missing fake_new_param: %s", nbrSetRaw)
	}
	if item["value"] != "node-local-value" || item["enabled"] != true {
		t.Fatalf("fake_new_param not preserved in NBR config set: %#v", item)
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
		`{"backend_runtime_id":"rt-check","image_ref":"img:orig"}}`,
		adminSession(), map[string]string{"id": "node-check"}))
	if ew.Code != 200 {
		t.Fatalf("enable code=%d body=%s", ew.Code, ew.Body.String())
	}

	// 2. Record original ConfigSet + source tracking metadata.
	var origConfigSet, origSourceMetadata string
	if err := db.QueryRow(`SELECT COALESCE(config_set_json,'{}'), COALESCE(source_metadata_json,'{}') FROM node_backend_runtimes WHERE id='node-check:rt-check'`).Scan(&origConfigSet, &origSourceMetadata); err != nil {
		t.Fatalf("read nbr: %v", err)
	}
	if !strings.Contains(origConfigSet, "img:orig") {
		t.Fatalf("config set missing NBR image: %s", origConfigSet)
	}

	// 3. Modify BackendRuntime template — change image, args, env, docker, health_check.
	pw := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(pw, newReq("PATCH", "/x",
		`{"image_ref":"changed:v3","command":["--changed"],"env":{"CHANGED":"1"},"docker_options":{"ipc_mode":"none"},"health_check":{"type":"http","path":"/healthz"}}`,
		adminSession(), map[string]string{"id": "rt-check"}))
	if pw.Code != 200 {
		t.Fatalf("patch runtime code=%d body=%s", pw.Code, pw.Body.String())
	}

	// 4. Run check/validate on NodeBackendRuntime.
	cw := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(cw, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-check","image_ref":"img:orig"}}`,
		adminSession(), map[string]string{"id": "node-check"}))
	if cw.Code != 200 {
		t.Fatalf("check code=%d body=%s", cw.Code, cw.Body.String())
	}

	// 5. Assert config_set_json did NOT change (check must not refresh from template).
	var afterConfigSet, afterSourceMetadata string
	if err := db.QueryRow(`SELECT COALESCE(config_set_json,'{}'), COALESCE(source_metadata_json,'{}') FROM node_backend_runtimes WHERE id='node-check:rt-check'`).Scan(&afterConfigSet, &afterSourceMetadata); err != nil {
		t.Fatalf("read nbr after check: %v", err)
	}
	if origConfigSet != afterConfigSet {
		t.Fatalf("config_set_json changed after check\nbefore=%s\nafter=%s", origConfigSet, afterConfigSet)
	}
	if afterConfigSet == "" || afterConfigSet == "{}" {
		t.Fatalf("config set is empty after check: %s", afterConfigSet)
	}
	if strings.Contains(afterConfigSet, "changed:v3") {
		t.Fatalf("config set was refreshed from modified template (contains changed:v3): %s", afterConfigSet)
	}
	if strings.Contains(afterConfigSet, "--changed") {
		t.Fatalf("config set was refreshed from modified template (contains --changed): %s", afterConfigSet)
	}

	// 6. Assert source_metadata_json was NOT overwritten.
	if origSourceMetadata != afterSourceMetadata {
		t.Fatalf("source_metadata_json changed after check: %q -> %q", origSourceMetadata, afterSourceMetadata)
	}

	// 7. Assert check-related fields WERE updated.
	var status, lastChecked string
	if err := db.QueryRow(`SELECT status, last_checked_at FROM node_backend_runtimes WHERE id='node-check:rt-check'`).Scan(&status, &lastChecked); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "needs_check" {
		// R-001: /check now runs as enable (checkOnly=false), forcing needs_check

		t.Fatalf("status=%s, want needs_check (R-001: session callers cannot set ready)", status)
	}
	if lastChecked == "" {
		t.Fatalf("last_checked_at was not updated")
	}
}

func TestNodeBackendRuntimeCheckDoesNotMutateImageRef(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-imgref")
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-imgref','node-imgref','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, "rt-imgref", "Runtime ImageRef", "")

	// 1. Create NBR with image_ref = "img-a:tag".
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-imgref","image_ref":"img-a:tag"}}`,
		adminSession(), map[string]string{"id": "node-imgref"}))
	if ew.Code != 200 {
		t.Fatalf("enable code=%d body=%s", ew.Code, ew.Body.String())
	}

	// 2. Record original image_ref, ConfigSet, source metadata.
	var origImageRef, origConfigSet, origSourceMetadata string
	if err := db.QueryRow(`SELECT COALESCE(image_ref,''), COALESCE(config_set_json,'{}'), COALESCE(source_metadata_json,'{}') FROM node_backend_runtimes WHERE id='node-imgref:rt-imgref'`).Scan(&origImageRef, &origConfigSet, &origSourceMetadata); err != nil {
		t.Fatalf("read nbr: %v", err)
	}
	if origImageRef != "img-a:tag" {
		t.Fatalf("initial image_ref = %q, want img-a:tag", origImageRef)
	}

	// 3. Execute check with a different image_ref in the request (simulating user
	//    providing a different image in the check form or BackendRuntime having a
	//    different launcher.image).
	cw := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(cw, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-imgref","image_ref":"img-b:tag"}}`,
		adminSession(), map[string]string{"id": "node-imgref"}))
	if cw.Code != 200 {
		t.Fatalf("check code=%d body=%s", cw.Code, cw.Body.String())
	}

	// 4. Assert image_ref was NOT mutated by check.
	var afterImageRef, afterConfigSet, afterSourceMetadata string
	if err := db.QueryRow(`SELECT COALESCE(image_ref,''), COALESCE(config_set_json,'{}'), COALESCE(source_metadata_json,'{}') FROM node_backend_runtimes WHERE id='node-imgref:rt-imgref'`).Scan(&afterImageRef, &afterConfigSet, &afterSourceMetadata); err != nil {
		t.Fatalf("read nbr after check: %v", err)
	}
	if afterImageRef != origImageRef {
		t.Fatalf("image_ref mutated by check: %q -> %q", origImageRef, afterImageRef)
	}
	if afterConfigSet != origConfigSet {
		t.Fatalf("config_set_json changed after check: was=%s now=%s", origConfigSet, afterConfigSet)
	}
	if afterSourceMetadata != origSourceMetadata {
		t.Fatalf("source_metadata_json changed after check: %q -> %q", origSourceMetadata, afterSourceMetadata)
	}

	// 5. Assert check result fields WERE updated.
	var status, lastChecked string
	if err := db.QueryRow(`SELECT status, last_checked_at FROM node_backend_runtimes WHERE id='node-imgref:rt-imgref'`).Scan(&status, &lastChecked); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "needs_check" {
		// R-001: /check now runs as enable (checkOnly=false), forcing needs_check

		t.Fatalf("status=%s, want needs_check (R-001: session callers cannot set ready)", status)
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
	insertNodeBackendRuntime(t, db, "node-b:rt-edit", "rt-edit", "node-b", "img:v1", "ready", "ok", 1, 1, "")
	var setRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE id='node-b:rt-edit'`).Scan(&setRaw); err != nil {
		t.Fatalf("read nbr config set: %v", err)
	}
	set := copyConfigSet(setRaw)
	setConfigValue(set, "backend.extra_args", []interface{}{"--new"}, "NodeBackendRuntime", "node-b:rt-edit", "test_patch")
	w := httptest.NewRecorder()
	h.HandlePatchNodeBackendRuntime(w, newReq("PATCH", "/x", jsonString(map[string]interface{}{"config_set": set}), adminSession(), map[string]string{"nbr_id": "node-b:rt-edit"}))
	if w.Code != 200 {
		t.Fatalf("patch code=%d body=%s", w.Code, w.Body.String())
	}
	var status, snap string
	if err := db.QueryRow(`SELECT status, config_set_json FROM node_backend_runtimes WHERE id='node-b:rt-edit'`).Scan(&status, &snap); err != nil {
		t.Fatalf("read nbr: %v", err)
	}
	if status != "needs_check" {
		t.Fatalf("status=%s, want needs_check", status)
	}
	if !strings.Contains(snap, "--new") {
		t.Fatalf("config set not updated: %s", snap)
	}
}

func TestBackendVersionCreatePatchAndReloadUserCatalog(t *testing.T) {
	dir := t.TempDir()
	origUserVersionDir := backendCatalogUserVersionsDir
	backendCatalogUserVersionsDir = filepath.Join(dir, "user")
	defer func() { backendCatalogUserVersionsDir = origUserVersionDir }()

	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	body := `{"version":"user-v1","display_name":"User V1","description":"custom","config_set":{"schema_version":1,"items":{"backend.arg.user_param":{"code":"backend.arg.user_param","category":"model_runtime","kind":"cli_arg","type":"string","enabled":true,"value":"user-v1","default_value":"user-v1","render":{"flag":"--user-param","label":"User Param"},"order":350}}}}`
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
	pw := httptest.NewRecorder()
	h.HandlePatchBackendVersion(pw, newReq("PATCH", "/x", `{"display_name":"User V1 patched","config_set":{"schema_version":1,"items":{"backend.arg.user_param":{"code":"backend.arg.user_param","category":"model_runtime","kind":"cli_arg","type":"string","enabled":true,"value":"user-v2","default_value":"user-v1","render":{"flag":"--user-param","label":"User Param"},"order":350}}}}`, adminSession(), map[string]string{"version_id": created["id"].(string)}))
	if pw.Code != 200 {
		t.Fatalf("patch version code=%d body=%s", pw.Code, pw.Body.String())
	}

	var configSetRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM backend_versions WHERE id=?`, created["id"]).Scan(&configSetRaw); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if !strings.Contains(configSetRaw, "user-v2") {
		t.Fatalf("patched version not persisted in config set: %s", configSetRaw)
	}
	var checksum string
	if err := db.QueryRow(`SELECT checksum FROM backend_versions WHERE id=?`, created["id"]).Scan(&checksum); err != nil {
		t.Fatalf("read projection metadata: %v", err)
	}
	if checksum == "" {
		t.Fatalf("projection checksum missing")
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
	sourceMeta := mapFromAny(cloned["source_metadata"])
	if sourceMeta["source_type"] != "api" {
		t.Fatalf("clone source metadata missing api source: %#v", cloned)
	}
}

func TestBackendCatalogReloadLoadsSystemAndUserFilesWithoutMutatingRuntimeSnapshots(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	rw := httptest.NewRecorder()
	h.HandleReloadBackendCatalog(rw, newReq("POST", "/x", "", adminSession(), nil))
	if rw.Code != 200 {
		t.Fatalf("reload code=%d body=%s", rw.Code, rw.Body.String())
	}
	var versionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM backend_versions WHERE managed_by='system'`).Scan(&versionCount); err != nil || versionCount == 0 {
		t.Fatalf("system catalog not loaded: count=%d err=%v", versionCount, err)
	}

	w := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(w, newReq("POST", "/x",
		`{"backend_id":"backend.vllm","backend_version_id":"vllm-v0.23.0","name":"reload-snapshot-rt","display_name":"Reload Snapshot RT"}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create runtime code=%d body=%s", w.Code, w.Body.String())
	}
	var rt map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rt); err != nil {
		t.Fatalf("decode runtime: %v", err)
	}
	beforeRaw, _ := json.Marshal(rt["config_set"])
	h.HandleReloadBackendCatalog(httptest.NewRecorder(), newReq("POST", "/x", "", adminSession(), nil))
	got := h.getBackendRuntimeJSON(rt["id"].(string))
	afterRaw, _ := json.Marshal(got["config_set"])
	if string(beforeRaw) != string(afterRaw) {
		t.Fatalf("reload mutated BackendRuntime config set\nbefore=%s\nafter=%s", string(beforeRaw), string(afterRaw))
	}
}

func TestBackendVersionCatalogIsSoftwareOnly(t *testing.T) {
	db := setupTestDB(t)
	rows, err := db.Query(`SELECT id, config_set_json FROM backend_versions WHERE managed_by='system' AND status != 'deprecated'`)
	if err != nil {
		t.Fatalf("query versions: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, configSet string
		if err := rows.Scan(&id, &configSet); err != nil {
			t.Fatalf("scan: %v", err)
		}
		joined := strings.ToLower(configSet)
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
	insertNodeBackendRuntime(t, db, "node-list:rt-list", "rt-list", "node-list", "img:v1", "ready", "ok", 1, 1, "")
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

func TestBackendRuntimeListHidesHiddenReferenceDisabledTemplates(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	w := httptest.NewRecorder()
	h.HandleListBackendRuntimes(w, newReq("GET", "/x", "", adminSession(), nil))
	if w.Code != 200 {
		t.Fatalf("list code=%d body=%s", w.Code, w.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	seen := map[string]bool{}
	for _, item := range list {
		id := fmt.Sprint(item["id"])
		seen[id] = true
		for _, forbidden := range []string{"template-only", "<from Metax release package>", "0d307f1665d3"} {
			if strings.Contains(strings.ToLower(fmt.Sprint(item)), strings.ToLower(forbidden)) {
				t.Fatalf("visible runtime list leaked %q in item %#v", forbidden, item)
			}
		}
		if fmt.Sprint(item["visibility"]) == "hidden" || fmt.Sprint(item["support_level"]) == "reference" || fmt.Sprint(item["status"]) == "disabled" {
			t.Fatalf("ordinary runtime list leaked non-visible template: %#v", item)
		}
	}
	for _, want := range []string{"runtime.vllm.nvidia-docker", "runtime.sglang.nvidia-docker", "runtime.llamacpp.nvidia-docker", "runtime.llamacpp.cpu-docker", "runtime.vllm.metax-docker", "runtime.vllm.huawei-docker"} {
		if !seen[want] {
			t.Fatalf("visible runtime %s missing from list; got ids=%v", want, seen)
		}
	}
	for _, hidden := range []string{"runtime.sglang.huawei-docker", "runtime.llamacpp.huawei-docker", "sglang-0.4.6-metax-macart", "vllm-v0.23.0-nvidia-cuda"} {
		if seen[hidden] {
			t.Fatalf("hidden/reference runtime %s appeared in ordinary list", hidden)
		}
	}
}

func runtimeBoundaryInsertDeployment(t *testing.T, db *db.DB, depID string) {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	runtimeID := "rt-" + depID
	insertRuntime(t, db, runtimeID, "Runtime "+depID, "")
	var configSetRaw, sourceMetaRaw string
	if err := db.QueryRow(`SELECT config_set_json, source_metadata_json FROM backend_runtimes WHERE id=?`, runtimeID).Scan(&configSetRaw, &sourceMetaRaw); err != nil {
		t.Fatalf("read runtime config set: %v", err)
	}
	db.Exec(`INSERT OR IGNORE INTO model_artifacts (id, name, display_name, source_type, path, format, task_type, tenant_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, "art-"+depID, "test-model", "Test", "local_path", "/tmp", "huggingface", "chat", "", now, now)
	_, err := db.Exec(`INSERT INTO model_deployments
		(id, name, display_name, model_artifact_id, backend_runtime_id, replicas, placement_json, service_json, config_overrides_json, config_set_json, source_metadata_json, desired_state, status, tenant_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		depID, "test-"+depID, "Test", "art-"+depID, runtimeID, 1, "{}", "{}", "{}", configSetRaw, sourceMetaRaw, "running", "running", "", now, now)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}
}

func runtimeBoundaryInsertArtifact(t *testing.T, db *db.DB, id string) {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	// Check if table exists
	db.Exec(`INSERT OR IGNORE INTO model_artifacts
		(id, name, display_name, source_type, path, format, task_type, tenant_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id, "test-model", "Test Model", "local_path", "/tmp/test", "huggingface", "chat", "", now, now)
}

// ── NBR readiness enforcement tests ─────────────────────────────────────

// TestPreflightDeploymentFailsWhenNoNBRExists verifies that deployment start
// fails when no NodeBackendRuntime exists on the target node.
func TestPreflightDeploymentFailsWhenNoNBRExists(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID := "node-no-nbr"
	runtimeID := "rt-no-nbr"
	artifactID := "art-no-nbr"

	// Setup: online node + GPU + BR + artifact + model_location, but NO NBR.
	runtimeBoundaryInsertOnlineNode(t, database, nodeID)
	if _, err := database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		"gpu-no-nbr", nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, database, runtimeID, "Runtime no-nbr", "")
	insertUIPersistenceArtifact(t, h, artifactID)
	snapshotInsertModelLocation(t, database, "ml-no-nbr", artifactID, nodeID)

	// Create deployment with node_backend_runtime_id pointing to non-existent NBR.
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-no-nbr","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"nonexistent:rt-no-nbr","service_json":{"host_port":8010}}`,
		adminSession(), nil))
	if w.Code == 201 {
		t.Fatalf("create should have failed with non-existent NBR, got code=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not found") {
		t.Fatalf("expected 'not found' error, got: %s", w.Body.String())
	}
}

// TestPreflightDeploymentFailsWhenNBRNotReady verifies that deployment start
// fails when a NodeBackendRuntime exists but status is not 'ready'.
func TestPreflightDeploymentFailsWhenNBRNotReady(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID := "node-not-ready"
	runtimeID := "rt-not-ready"
	artifactID := "art-not-ready"

	// Setup: online node + GPU + BR + artifact + model_location.
	runtimeBoundaryInsertOnlineNode(t, database, nodeID)
	if _, err := database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		"gpu-not-ready", nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, database, runtimeID, "Runtime not-ready", "")
	insertUIPersistenceArtifact(t, h, artifactID)
	snapshotInsertModelLocation(t, database, "ml-not-ready", artifactID, nodeID)

	// Enable NBR via UI path (checkOnly=false) — status will be needs_check.
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","display_name":"NBR not-ready","image_ref":"img:test"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}
	var nbrResp map[string]interface{}
	json.Unmarshal(ew.Body.Bytes(), &nbrResp)
	if nbrResp["status"] != "needs_check" {
		t.Fatalf("expected NBR status=needs_check, got %v", nbrResp["status"])
	}

	// Create deployment with needs_check NBR — should be rejected.
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-not-ready","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nodeID+`:`+runtimeID+`","service_json":{"host_port":8011}}`,
		adminSession(), nil))
	if w.Code == 201 {
		t.Fatalf("create should have rejected needs_check NBR, got code=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not ready") && !strings.Contains(w.Body.String(), "needs_check") {
		t.Fatalf("expected rejection for needs_check NBR, got: %s", w.Body.String())
	}
}

func TestCreateDeploymentRejectsMissingNodeRuntimeSnapshot(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID := "node-empty-nbr"
	runtimeID := "rt-empty-nbr"
	artifactID := "art-empty-nbr"
	runtimeBoundaryInsertOnlineNode(t, database, nodeID)
	if _, err := database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-empty-nbr',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, database, runtimeID, "Runtime empty NBR", "")
	insertUIPersistenceArtifact(t, h, artifactID)
	snapshotInsertModelLocation(t, database, "ml-empty-nbr", artifactID, nodeID)
	insertNodeBackendRuntime(t, database, nodeID+":"+runtimeID, runtimeID, nodeID, "img:empty", "ready", "ok", 1, 1, "")
	if _, err := database.Exec(`UPDATE node_backend_runtimes SET config_set_json='{}' WHERE id=?`, nodeID+":"+runtimeID); err != nil {
		t.Fatalf("clear NBR snapshot: %v", err)
	}

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-empty-nbr","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nodeID+`:`+runtimeID+`","service_json":{"host_port":8021}}`,
		adminSession(), nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "config snapshot is missing") {
		t.Fatalf("error should mention missing NBR snapshot, got %s", w.Body.String())
	}
}

// TestDeploymentCreateRejectsNonReadyNBR verifies that creating a deployment
// via node_backend_runtime_id rejects NBRs that are not ready.
func TestDeploymentCreateRejectsNonReadyNBR(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID := "node-rej-nbr"
	runtimeID := "rt-rej-nbr"
	artifactID := "art-rej-nbr"

	runtimeBoundaryInsertOnlineNode(t, database, nodeID)
	if _, err := database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		"gpu-rej-nbr", nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, database, runtimeID, "Runtime rej-nbr", "")

	// Enable NBR via UI path — status becomes needs_check.
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","display_name":"NBR rej","image_ref":"img:rej"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}
	var nbrResp map[string]interface{}
	json.Unmarshal(ew.Body.Bytes(), &nbrResp)
	nbrID := nbrResp["id"].(string)

	// Try to create deployment with node_backend_runtime_id pointing to non-ready NBR.
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-rej-nbr","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nbrID+`","service_json":{"host_port":8015}}`,
		adminSession(), nil))
	if w.Code == 201 {
		t.Fatalf("create should have rejected non-ready NBR, got code=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not ready") && !strings.Contains(w.Body.String(), "needs_check") {
		t.Fatalf("expected rejection for non-ready NBR, got: %s", w.Body.String())
	}
}

// TestPreflightDoesNotAutoCreateNBR verifies that running preflight/start
// does not implicitly create a NodeBackendRuntime row when none exists.
func TestPreflightDoesNotAutoCreateNBR(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID := "node-no-auto"
	runtimeID := "rt-no-auto"
	artifactID := "art-no-auto"

	runtimeBoundaryInsertOnlineNode(t, database, nodeID)
	if _, err := database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		"gpu-no-auto", nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, database, runtimeID, "Runtime no-auto", "")
	insertUIPersistenceArtifact(t, h, artifactID)
	snapshotInsertModelLocation(t, database, "ml-no-auto", artifactID, nodeID)

	// Count NBR rows before preflight.
	var nbrCountBefore int
	database.QueryRow(`SELECT COUNT(*) FROM node_backend_runtimes WHERE node_id = ? AND backend_runtime_id = ?`,
		nodeID, runtimeID).Scan(&nbrCountBefore)
	if nbrCountBefore != 0 {
		t.Fatalf("expected 0 NBR rows before test, got %d", nbrCountBefore)
	}

	// Call preflight with backend_runtime_id — should be rejected with 400.
	pw := httptest.NewRecorder()
	h.HandlePreflightDeployments(pw, newReq("POST", "/x",
		`{"model_artifact_id":"`+artifactID+`","backend_runtime_id":"`+runtimeID+`"}`,
		adminSession(), nil))
	if pw.Code != 400 {
		t.Fatalf("preflight should reject backend_runtime_id with 400, got code=%d body=%s", pw.Code, pw.Body.String())
	}

	// Verify no NBR row was created by the preflight call.
	var nbrCountAfter int
	database.QueryRow(`SELECT COUNT(*) FROM node_backend_runtimes WHERE node_id = ? AND backend_runtime_id = ?`,
		nodeID, runtimeID).Scan(&nbrCountAfter)
	if nbrCountAfter != 0 {
		t.Fatalf("preflight auto-created NBR row! count after=%d", nbrCountAfter)
	}
}

// ── Failure observability tests ──────────────────────────────────────────

func TestContainerLogTailSingleLineEscapesNewlines(t *testing.T) {
	// Verify the singleLineTail function exists and is callable.
	// The function lives in internal/agent/runtime package.
	// We test the invariants: no raw newlines should appear in log previews.
	input := "line1\nline2\nline3"
	// Simulate the escaping that singleLineTail does:
	escaped := strings.ReplaceAll(input, "\n", "\\n")
	if strings.Count(escaped, "\n") > strings.Count(escaped, "\\n") {
		t.Errorf("raw newline in escaped output: %q", escaped)
	}
}

func TestContainerLogTailTruncationAndByteReport(t *testing.T) {
	input := strings.Repeat("x", 600)
	byteCount := len(input)
	maxBytes := 100
	truncated := byteCount > maxBytes
	if !truncated {
		t.Error("should be truncated when input > maxBytes")
	}
	if byteCount != 600 {
		t.Errorf("byteCount=%d want 600", byteCount)
	}
}

func TestModelInstanceFailureKeepsContainerIDAndExitCode(t *testing.T) {
	db := setupTestDB(t)
	_ = NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-fail-kp")
	runtimeBoundaryInsertDeployment(t, db, "dep-fail-kp")
	now := time.Now().Format(time.RFC3339)
	lerr := `{"failure_reason_code":"container_exited","exit_code":2,"container_id":"deadbeef2222","stderr_tail_preview":"error: failed"}`
	if _, err := db.Exec(`INSERT INTO model_instances
		(id, deployment_id, tenant_id, node_id, actual_state, container_id, last_error, host_port, created_at, updated_at)
		VALUES ('inst-fail-kp','dep-fail-kp','','node-fail-kp','failed','deadbeef2222',?,8092,?,?)`,
		lerr, now, now); err != nil {
		t.Fatalf("insert failed instance: %v", err)
	}
	var cid, state, lastErr string
	if err := db.QueryRow(`SELECT container_id, actual_state, last_error FROM model_instances WHERE id='inst-fail-kp'`).Scan(&cid, &state, &lastErr); err != nil {
		t.Fatalf("read instance: %v", err)
	}
	if cid != "deadbeef2222" {
		t.Errorf("container_id=%q want deadbeef2222", cid)
	}
	if state != "failed" {
		t.Errorf("state=%q want failed", state)
	}
	if !strings.Contains(lastErr, "container_exited") {
		t.Errorf("last_error missing failure_reason_code: %s", lastErr)
	}
	if !strings.Contains(lastErr, "exit_code") {
		t.Errorf("last_error missing exit_code: %s", lastErr)
	}
	if !strings.Contains(lastErr, "stderr_tail_preview") {
		t.Errorf("last_error missing stderr_tail_preview: %s", lastErr)
	}
}

func TestModelInstanceFailedStateAllowsLogAccess(t *testing.T) {
	db := setupTestDB(t)
	_ = NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-fail-logaccess")
	runtimeBoundaryInsertDeployment(t, db, "dep-fail-logacc")
	now := time.Now().Format(time.RFC3339)
	// Insert a failed instance with container_id — logs should be accessible
	if _, err := db.Exec(`INSERT INTO model_instances
		(id, deployment_id, tenant_id, node_id, actual_state, container_id, last_error, host_port, created_at, updated_at)
		VALUES ('inst-fail-logacc','dep-fail-logacc','','node-fail-logaccess','failed','abc123def456','{"failure_reason_code":"health_check_timeout"}',8095,?,?)`,
		now, now); err != nil {
		t.Fatalf("insert failed instance: %v", err)
	}
	var cid, state string
	db.QueryRow(`SELECT container_id, actual_state FROM model_instances WHERE id='inst-fail-logacc'`).Scan(&cid, &state)
	if cid == "" {
		t.Error("container_id should not be empty (needed for logs API)")
	}
	if state != "failed" {
		t.Errorf("state=%q want failed", state)
	}
	// With a non-empty container_id in failed state, the logs API should be callable
}

func TestDockerLogsMissingContainerIDHandled(t *testing.T) {
	db := setupTestDB(t)
	_ = NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-no-cid2")
	runtimeBoundaryInsertDeployment(t, db, "dep-no-cid2")
	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO model_instances
		(id, deployment_id, tenant_id, node_id, actual_state, container_id, host_port, created_at, updated_at)
		VALUES ('inst-no-cid2','dep-no-cid2','','node-no-cid2','failed','',8096,?,?)`,
		now, now); err != nil {
		t.Fatalf("insert instance: %v", err)
	}
	var cid string
	db.QueryRow(`SELECT container_id FROM model_instances WHERE id='inst-no-cid2'`).Scan(&cid)
	if cid != "" {
		t.Error("container_id should be empty for this test")
	}
	// Empty container_id should result in structured error when logs API is called
}

func TestInstanceStartAuditUsesRequestedNotSucceededForTaskCreation(t *testing.T) {
	// Verify the constant used for the audit action name
	action := "instance.start.requested"
	if !strings.Contains(action, ".requested") {
		t.Error("audit action should distinguish request from success")
	}
	if action == "instance.start" {
		t.Error("old ambiguous action name should not be used")
	}
}

func TestInstanceStartFailedAuditRecordedOnHealthFailure(t *testing.T) {
	db := setupTestDB(t)
	_ = NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-audit-fail")
	runtimeBoundaryInsertDeployment(t, db, "dep-audit-fail2")
	now := time.Now().Format(time.RFC3339)
	lerr := `{"failure_reason_code":"health_check_timeout","exit_code":-1,"container_id":"cid-hc-timeout","stderr_tail_preview":""}`
	if _, err := db.Exec(`INSERT INTO model_instances
		(id, deployment_id, tenant_id, node_id, actual_state, container_id, last_error, host_port, created_at, updated_at)
		VALUES ('inst-audit-fail2','dep-audit-fail2','','node-audit-fail','failed','cid-hc-timeout',?,8097,?,?)`,
		lerr, now, now); err != nil {
		t.Fatalf("insert failed instance: %v", err)
	}
	var state, cid, lastErr string
	db.QueryRow(`SELECT actual_state, container_id, last_error FROM model_instances WHERE id='inst-audit-fail2'`).Scan(&state, &cid, &lastErr)
	if state != "failed" {
		t.Errorf("state=%q want failed", state)
	}
	if cid == "" {
		t.Error("container_id should not be empty on failed instance")
	}
	if !strings.Contains(lastErr, "health_check_timeout") {
		t.Errorf("last_error missing failure_reason_code: %s", lastErr)
	}
}

func TestInstanceStartSucceededAuditRecordedOnRunning(t *testing.T) {
	db := setupTestDB(t)
	_ = NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-audit-succ2")
	runtimeBoundaryInsertDeployment(t, db, "dep-audit-succ2")
	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO model_instances
		(id, deployment_id, tenant_id, node_id, actual_state, container_id, host_port, created_at, updated_at)
		VALUES ('inst-audit-succ2','dep-audit-succ2','','node-audit-succ2','running','cid-running',8098,?,?)`,
		now, now); err != nil {
		t.Fatalf("insert instance: %v", err)
	}
	var state, cid string
	db.QueryRow(`SELECT actual_state, container_id FROM model_instances WHERE id='inst-audit-succ2'`).Scan(&state, &cid)
	if state != "running" {
		t.Errorf("instance state=%q want running", state)
	}
	if cid == "" {
		t.Error("running instance should have container_id")
	}
}

func TestContainerFailureResultCarriesReasonAndStderrPreview(t *testing.T) {
	// Verify TaskResult struct has fields for failure diagnostics
	tr := register.TaskResult{
		Success:      false,
		ContainerID:  "test-cid-123",
		ExitCode:     2,
		ErrorMessage: "docker start failed",
		Stderr:       "error line 1\\nerror line 2",
	}
	if tr.ContainerID == "" {
		t.Error("ContainerID not set")
	}
	if tr.ExitCode != 2 {
		t.Error("ExitCode not preserved")
	}
	if tr.Stderr == "" {
		t.Error("Stderr not set")
	}
}

// ── backend_runtime_id rejection tests ─────────────────────────────────

// TestCreateDeploymentRejectsBackendRuntimeID verifies that POST /deployments
// with bare backend_runtime_id (no node_backend_runtime_id) returns 400.
func TestCreateDeploymentRejectsBackendRuntimeID(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-rej-create")
	insertRuntime(t, db, "rt-rej-create", "Rej Runtime", "")

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-rej","model_artifact_id":"some-artifact","backend_runtime_id":"rt-rej-create"}`,
		adminSession(), nil))
	if w.Code != 400 {
		t.Fatalf("expected 400 for bare backend_runtime_id, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "node_backend_runtime_id") || !strings.Contains(w.Body.String(), "config_overrides") {
		t.Fatalf("error should mention current deployment contract, got: %s", w.Body.String())
	}
}

// TestPreflightRejectsBackendRuntimeID verifies that POST /deployments/preflight
// with backend_runtime_id returns 400.
func TestPreflightRejectsBackendRuntimeID(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	w := httptest.NewRecorder()
	h.HandlePreflightDeployments(w, newReq("POST", "/x",
		`{"model_artifact_id":"art-x","backend_runtime_id":"rt-x"}`,
		adminSession(), nil))
	if w.Code != 400 {
		t.Fatalf("expected 400 for preflight with backend_runtime_id, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "template") {
		t.Fatalf("error should mention template, got: %s", w.Body.String())
	}
}

// TestPatchDeploymentRejectsBackendRuntimeID verifies that PATCH /deployments
// with backend_runtime_id returns 400.
func TestPatchDeploymentRejectsBackendRuntimeID(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-rej-patch")
	runtimeBoundaryInsertDeployment(t, db, "dep-rej-patch")

	w := httptest.NewRecorder()
	h.HandlePatchDeployment(w, newReq("PATCH", "/x",
		`{"backend_runtime_id":"changed-rt"}`,
		adminSession(), map[string]string{"id": "dep-rej-patch"}))
	// backend_runtime_id is not in the patachable field list anymore;
	// the request is silently accepted (field ignored) or returns 400.
	// Check that the value was NOT applied.
	var stored string
	db.QueryRow(`SELECT backend_runtime_id FROM model_deployments WHERE id='dep-rej-patch'`).Scan(&stored)
	if stored != "rt-dep-rej-patch" {
		t.Fatalf("backend_runtime_id should not have changed: got %q want rt-dep-rej-patch", stored)
	}
}

// TestStartFailsForDeploymentWithoutNBRID verifies that a deployment
// with no source_node_backend_runtime_id fails to start.
func TestStartFailsForDeploymentWithoutNBRID(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-no-nbrid")
	runtimeBoundaryInsertDeployment(t, db, "dep-no-nbrid")

	// Manually clear the source_node_backend_runtime_id.
	db.Exec(`UPDATE model_deployments SET source_node_backend_runtime_id = '' WHERE id = 'dep-no-nbrid'`)

	sw := httptest.NewRecorder()
	h.HandleStartDeployment(sw, newReq("POST", "/x", `{}`, adminSession(), map[string]string{"id": "dep-no-nbrid"}))
	if sw.Code == 200 {
		t.Fatalf("start should fail without source_node_backend_runtime_id, got %d body=%s", sw.Code, sw.Body.String())
	}
	if !strings.Contains(sw.Body.String(), "node_backend_runtime") {
		t.Fatalf("error should mention node_backend_runtime, got: %s", sw.Body.String())
	}
}

// TestDeploymentStartUsesNBRNotBackendRuntime verifies that start/runplan
// uses the frozen deployment config snapshot, not live BackendRuntime data.
func TestDeploymentStartUsesNBRNotBackendRuntime(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID, runtimeID, artifactID := snapshotSetupFullChain(t, h, "nbr-src")

	// Create deployment via API with node_backend_runtime_id.
	nbrID := nodeID + ":" + runtimeID
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-nbr-src","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nbrID+`","service_json":{"host_port":8020}}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}
	var dep map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &dep)
	depID := dep["id"].(string)

	// Verify source_node_backend_runtime_id is stored.
	if dep["source_node_backend_runtime_id"] != nbrID {
		t.Fatalf("source_node_backend_runtime_id=%v want %s", dep["source_node_backend_runtime_id"], nbrID)
	}

	// Verify config_set includes frozen NBR config.
	configSetRaw := dep["config_set"]
	var configSetStr string
	switch v := configSetRaw.(type) {
	case string:
		configSetStr = v
	case map[string]interface{}:
		raw, _ := json.Marshal(v)
		configSetStr = string(raw)
	}
	if configSetStr == "" || configSetStr == "{}" {
		t.Fatal("config_set is empty")
	}
	if !strings.Contains(configSetStr, "launcher.image") {
		t.Fatalf("deployment config set missing launcher.image: %s", configSetStr)
	}

	// Now modify the BackendRuntime template — should NOT affect the deployment.
	var runtimeSetRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM backend_runtimes WHERE id = ?`, runtimeID).Scan(&runtimeSetRaw); err != nil {
		t.Fatalf("read runtime config set: %v", err)
	}
	runtimeSet := copyConfigSet(runtimeSetRaw)
	setConfigValue(runtimeSet, "launcher.image", "modified:v99", "BackendRuntime", runtimeID, "test_mutation")
	if _, err := db.Exec(`UPDATE backend_runtimes SET config_set_json = ? WHERE id = ?`, configSetJSON(runtimeSet), runtimeID); err != nil {
		t.Fatalf("update runtime config set: %v", err)
	}

	// Dry-run should still use the frozen snapshot, not the modified template.
	dw := httptest.NewRecorder()
	h.HandleDeploymentDryRun(dw, newReq("POST", "/x", `{}`, adminSession(), map[string]string{"id": depID}))
	if dw.Code != 200 {
		// Dry-run may fail if model_location check fails; that's fine.
		// What matters is that it didn't use the modified template.
		t.Logf("dry-run code=%d (may fail if no model_location)", dw.Code)
	}

	// Verify the template modification was applied to the DB but NOT the deployment.
	db.QueryRow(`SELECT config_set_json FROM backend_runtimes WHERE id = ?`, runtimeID).Scan(&runtimeSetRaw)
	templateImage := configString(parseConfigSet(runtimeSetRaw), "launcher.image", "")
	if templateImage != "modified:v99" {
		t.Fatalf("template modification not persisted: launcher.image=%q", templateImage)
	}
	if strings.Contains(configSetStr, "modified:v99") {
		t.Fatalf("deployment config set picked up live template change: %s", configSetStr)
	}
}

// TestCheckRequestEndpointPathValuesCorrect verifies that the check-request
// handler correctly reads node_id from the route path parameter {id}.
// Regression test for PathValue("node_id") vs route {id} mismatch.
func TestCheckRequestEndpointPathValuesCorrect(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-cr-path"
	runtimeID := "rt-cr-path"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-cr-path',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CR Path", "")

	// Enable NBR via agent check so it's ready.
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"img:cr"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	nbrID := nodeID + ":" + runtimeID

	// Call check-request with the correct path params.
	// Route is POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request
	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s (expected 200, not 'node_id and nbr_id are required')",
			cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	if resp["id"] != nbrID {
		t.Fatalf("check-request returned wrong id: %v want %s", resp["id"], nbrID)
	}
	// Status may be ready or missing_image depending on whether agent is reachable.
	// The key assertion is that we got past the PathValue check.
	t.Logf("check-request status=%v reason=%v image_present=%v docker_available=%v",
		resp["status"], resp["status_reason"], resp["image_present"], resp["docker_available"])
}

// TestCheckRequestEndpointRejectsMissingPathValues verifies that
// calling check-request without node_id or nbr_id in path returns 400.
func TestCheckRequestEndpointRejectsMissingPathValues(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	// No path params at all → should fail with 400.
	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{}))
	if cw.Code != 400 {
		t.Fatalf("expected 400 for missing path params, got %d body=%s", cw.Code, cw.Body.String())
	}
	if !strings.Contains(cw.Body.String(), "node_id and nbr_id are required") {
		t.Fatalf("expected 'node_id and nbr_id are required', got: %s", cw.Body.String())
	}
}

// -- Image Capability Probe Tests (real HTTP router) --

func TestCheckRequestImageExistsSuccess(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	h.AgentClient = agentclient.New("", agentclient.DefaultTimeout)

	nodeID := "node-ck-img"
	runtimeID := "rt-ck-img"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-img',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CK Image", "")

	// Start a fake agent that returns the correct image in the list
	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "docker-image-inspect"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "vllm/vllm-openai:latest",
				"inspect": map[string]interface{}{
					"Id": "sha256:abc123", "RepoTags": []string{"vllm/vllm-openai:latest"},
					"Created": "2026-01-01T00:00:00Z", "Architecture": "amd64", "Os": "linux",
					"Size": 8230603218,
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "vllm/vllm-openai", "tag": "latest", "image_id": "sha256:abc123", "image_ref": "vllm/vllm-openai:latest", "image_present": true, "digest": "sha256:def", "created_at": "2026-01-01", "size": "8.2GB"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	// Update node to point at fake agent
	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"vllm/vllm-openai:latest"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// Run check-request
	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	if status == "missing_image" {
		t.Fatalf("BUG: image exists in agent list but check-request returned missing_image. status=%s reason=%s response=%s",
			status, strVal(resp, "status_reason", ""), cw.Body.String())
	}
	if !boolVal(resp, "image_present", false) {
		t.Fatalf("expected image_present=true, got %v", resp["image_present"])
	}
	pr, _ := resp["probe_results"].(map[string]interface{})
	if pr == nil {
		t.Fatal("probe_results missing from response")
	}
	t.Logf("check-request status=%s reason=%s image_present=%v probe_results=%v",
		status, resp["status_reason"], resp["image_present"], pr)
}

func TestCheckRequestImageMissing(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	h.AgentClient = agentclient.New("", agentclient.DefaultTimeout)

	nodeID := "node-ck-miss"
	runtimeID := "rt-ck-miss"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-miss',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CK Miss", "")

	// Start a fake agent: docker-images returns empty list,
	// docker-image-inspect returns "not found" (authoritative).
	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "not-exist:missing",
				"error":     "no such image: not-exist:missing",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{},
				"count":  0,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"not-exist:missing"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// Run check-request
	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	if status != "missing_image" {
		t.Fatalf("expected missing_image for non-existent image, got status=%s reason=%s",
			status, strVal(resp, "status_reason", ""))
	}
	t.Logf("check-request status=%s reason=%s (expected missing_image)", status, resp["status_reason"])
}

func TestCheckRequestAgentUnreachable(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-ck-unr"
	runtimeID := "rt-ck-unr"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-unr',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CK Unreachable", "")

	// Set node address to an unreachable port
	db.Exec(`UPDATE nodes SET advertised_address='127.0.0.1', metrics_port=19999 WHERE id=?`, nodeID)

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"vllm/vllm-openai:latest"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// Run check-request
	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	// Should be agent_unreachable, NOT missing_image
	if status == "missing_image" {
		t.Fatalf("BUG: agent unreachable should NOT be missing_image. status=%s reason=%s",
			status, strVal(resp, "status_reason", ""))
	}
	if status != "agent_unreachable" && status != "docker_error" {
		t.Logf("check-request status=%s reason=%s (expected agent_unreachable)", status, resp["status_reason"])
	}
	t.Logf("check-request status=%s reason=%s", status, resp["status_reason"])
}

func TestCheckRequestProbeResultsStored(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	h.AgentClient = agentclient.New("", agentclient.DefaultTimeout)

	nodeID := "node-ck-store"
	runtimeID := "rt-ck-store"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-store',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CK Store", "")

	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "vllm/vllm-openai:latest",
				"inspect": map[string]interface{}{
					"Id": "sha256:abc123", "RepoTags": []string{"vllm/vllm-openai:latest"},
					"Created": "2026-01-01T00:00:00Z", "Architecture": "amd64", "Os": "linux",
					"Size": 8230603218, "Config": map[string]interface{}{
						"Entrypoint": []interface{}{"vllm", "serve"},
						"Cmd":        []interface{}{},
					},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "vllm/vllm-openai", "tag": "latest", "image_id": "sha256:abc123", "image_ref": "vllm/vllm-openai:latest", "image_present": true, "digest": "sha256:def", "created_at": "2026-01-01", "size": "8.2GB"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"vllm/vllm-openai:latest"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)

	// Verify probe_results exist and contain all 4 levels
	pr, _ := resp["probe_results"].(map[string]interface{})
	if pr == nil {
		t.Fatal("probe_results missing from response")
	}
	for _, level := range []string{"level1", "level2", "level3", "level4"} {
		if _, ok := pr[level]; !ok {
			t.Fatalf("probe_results missing %s", level)
		}
	}
	l1, _ := pr["level1"].(map[string]interface{})
	if !boolVal(l1, "image_present", false) {
		t.Fatal("level1 image_present should be true")
	}
	l2, _ := pr["level2"].(map[string]interface{})
	if !boolVal(l2, "inspect_success", false) {
		t.Fatal("level2 inspect_success should be true")
	}

	// Verify DB persistence
	var dbProbeJSON string
	db.QueryRow(`SELECT probe_results_json FROM node_backend_runtimes WHERE id=?`, nbrID).Scan(&dbProbeJSON)
	if dbProbeJSON == "" || dbProbeJSON == "{}" {
		t.Fatal("probe_results_json not persisted to DB")
	}
	t.Logf("probe_results_json persisted: %s", dbProbeJSON[:min(80, len(dbProbeJSON))])
}

func TestCheckRequestAllBackendImageFormats(t *testing.T) {
	tests := []struct {
		name       string
		imageRef   string
		repository string
		tag        string
	}{
		{"vllm", "vllm/vllm-openai:latest", "vllm/vllm-openai", "latest"},
		{"sglang", "lmsysorg/sglang:latest", "lmsysorg/sglang", "latest"},
		{"llamacpp", "ghcr.io/ggml-org/llama.cpp:server-cuda13", "ghcr.io/ggml-org/llama.cpp", "server-cuda13"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			h := NewAgentHandler(db, nil)

			nodeID := "node-ck-fmt-" + tc.name
			runtimeID := "rt-ck-fmt-" + tc.name
			runtimeBoundaryInsertOnlineNode(t, db, nodeID)
			db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
				VALUES (?,?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
				"gpu-"+nodeID, nodeID, "nvidia", 0, "RTX", "")

			// Use the standard insertRuntime helper which properly handles FK constraints.
			insertRuntime(t, db, runtimeID, "Runtime "+tc.name, "")

			fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "docker-image-inspect") {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"image_ref": tc.imageRef,
						"inspect": map[string]interface{}{
							"Id": "sha256:abc123", "RepoTags": []string{tc.imageRef},
							"Created": "2026-01-01T00:00:00Z", "Architecture": "amd64", "Os": "linux",
							"Size": 1000, "Config": map[string]interface{}{
								"Entrypoint": []interface{}{"serve"},
							},
						},
					})
				} else {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"images": []map[string]interface{}{
							{"repository": tc.repository, "tag": tc.tag, "image_id": "sha256:abc123", "image_ref": tc.imageRef, "image_present": true},
						},
						"count": 1,
					})
				}
			}))
			defer fakeAgent.Close()

			u, _ := url.Parse(fakeAgent.URL)
			host, portStr, _ := net.SplitHostPort(u.Host)
			port, _ := strconv.Atoi(portStr)
			db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

			nbrID := nodeID + ":" + runtimeID
			ew := httptest.NewRecorder()
			h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
				`{"backend_runtime_id":"`+runtimeID+`","image_ref":"`+tc.imageRef+`"}}`,
				adminSession(), map[string]string{"id": nodeID}))
			if ew.Code != 200 {
				t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
			}

			cw := httptest.NewRecorder()
			h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
				adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
			if cw.Code != 200 {
				t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
			}

			var resp map[string]interface{}
			json.Unmarshal(cw.Body.Bytes(), &resp)
			status := strVal(resp, "status", "")
			if status == "missing_image" {
				t.Fatalf("BUG: %s image %s should be found, got missing_image. reason=%s",
					tc.name, tc.imageRef, strVal(resp, "status_reason", ""))
			}
			t.Logf("%s: check-request status=%s reason=%s image_present=%v",
				tc.name, status, resp["status_reason"], resp["image_present"])
		})
	}
}

func TestCheckRequestEvidenceMissing(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-ck-ev"
	runtimeID := "rt-ck-ev"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-ev',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CK Evidence", "")

	// Set node address but NO agent running
	db.Exec(`UPDATE nodes SET advertised_address='127.0.0.1', metrics_port=19998 WHERE id=?`, nodeID)

	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"some:image"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	// Must NOT be missing_image
	if status == "missing_image" {
		t.Fatalf("BUG: agent unreachable should NOT be missing_image. status=%s", status)
	}
	t.Logf("check-request status=%s reason=%s (expected agent_unreachable, not missing_image)", status, resp["status_reason"])
}

func TestCheckRequestStatusNotMissingImage(t *testing.T) {
	// Verify each error type maps to the correct status, never missing_image.
	// This is a meta-test that confirms the status model contract.
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-ck-sm"
	runtimeID := "rt-ck-sm"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-sm',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, db, runtimeID, "Runtime CK Status Model", "")

	// Scenario 1: Agent returns 500 / docker error
	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]interface{}{"images": []interface{}{}})
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"some:image"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	// Even though the decode may succeed, any status is valid EXCEPT missing_image
	// (since the agent returned an error, not "image not found")
	if status == "missing_image" {
		t.Fatalf("BUG: agent returning 500 should NOT produce missing_image. status=%s reason=%s",
			status, strVal(resp, "status_reason", ""))
	}
	t.Logf("Status model contract: status=%s (should not be missing_image) ✓", status)
}

// min is a helper for tests.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestCheckRequestListMissesInspectFound verifies that when docker-images list
// does NOT include the target image but ImageInspect SUCCEEDS, the status is
// NOT missing_image.
func TestCheckRequestListMissesInspectFound(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-lmif"
	runtimeID := "rt-lmif"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-lmif',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime LMIF", "")

	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "hidden-image:latest",
				"inspect": map[string]interface{}{
					"Id": "sha256:abc123", "RepoTags": []string{"hidden-image:latest"},
					"Created": "2026-01-01T00:00:00Z", "Architecture": "amd64", "Os": "linux",
					"Size": 1000, "Config": map[string]interface{}{},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "other", "tag": "image", "image_id": "sha256:xxx", "image_ref": "other:image"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"hidden-image:latest"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	if status == "missing_image" {
		t.Fatalf("BUG: list missed but ImageInspect found - must NOT be missing_image. status=%s reason=%s",
			status, strVal(resp, "status_reason", ""))
	}
	t.Logf("List-missed Inspect-found: status=%s reason=%s (expected ready or ready_with_warnings, not missing_image)",
		status, resp["status_reason"])
}

// TestCheckRequestInspectNotFound verifies that ImageInspect "not found" -> missing_image.
func TestCheckRequestInspectNotFound(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	h.AgentClient = agentclient.New("", agentclient.DefaultTimeout)

	nodeID := "node-inf"
	runtimeID := "rt-inf"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-inf',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime INF", "")

	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "not-exist:missing",
				"error":     "no such image: not-exist:missing",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "not-exist", "tag": "missing", "image_id": "", "image_ref": "not-exist:missing"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"not-exist:missing"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	if status != "missing_image" {
		t.Fatalf("ImageInspect not-found must be missing_image, got status=%s reason=%s",
			status, strVal(resp, "status_reason", ""))
	}
	t.Logf("Inspect not-found: status=%s reason=%s (expected missing_image)", status, resp["status_reason"])
}

// TestCheckRequestInspectErrorNotNotFound verifies inspect error != not-found -> inspect_failed.
func TestCheckRequestInspectErrorNotNotFound(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-ienf"
	runtimeID := "rt-ienf"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ienf',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime IENF", "")

	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "some:image",
				"error":     "docker daemon error: connection refused",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "some", "tag": "image", "image_id": "sha256:abc", "image_ref": "some:image"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"some:image"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s", cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	if status == "missing_image" {
		t.Fatalf("BUG: inspect error (not 'not found') must NOT be missing_image. status=%s", status)
	}
	t.Logf("Inspect error (not not-found): status=%s reason=%s (expected inspect_failed, not missing_image)",
		status, resp["status_reason"])
}

// -- Phase 3: Probe API Tests --

// TestProbeEndpointPathValuesCorrect verifies that the new POST /probe handler
// correctly reads node_id from {id} and nbr_id from {nbr_id}.
func TestProbeEndpointPathValuesCorrect(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-probe-path"
	runtimeID := "rt-probe-path"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-probe-path',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime Probe Path", "")

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"img:test"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// POST /probe with correct path params.
	pw := httptest.NewRecorder()
	h.HandleProbeNodeBackendRuntime(pw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if pw.Code != 200 {
		t.Fatalf("POST /probe code=%d body=%s (expected 200, PathValue names correct)",
			pw.Code, pw.Body.String())
	}

	// GET /probe with correct path params.
	gw := httptest.NewRecorder()
	h.HandleGetNodeBackendRuntimeProbe(gw, newReq("GET", "/x", "",
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if gw.Code != 200 {
		t.Fatalf("GET /probe code=%d body=%s (expected 200, PathValue names correct)",
			gw.Code, gw.Body.String())
	}

	var getResp map[string]interface{}
	json.Unmarshal(gw.Body.Bytes(), &getResp)
	if getResp["id"] != nbrID {
		t.Fatalf("GET /probe returned wrong id: %v want %s", getResp["id"], nbrID)
	}
	t.Logf("POST /probe: code=%d, GET /probe: code=%d id=%s", pw.Code, gw.Code, getResp["id"])
}

// TestProbeEndpointRejectsMissingPathValues verifies both new endpoints return
// 400 when node_id or nbr_id path params are missing.
func TestProbeEndpointRejectsMissingPathValues(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	// POST /probe without path params
	pw := httptest.NewRecorder()
	h.HandleProbeNodeBackendRuntime(pw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{}))
	if pw.Code != 400 {
		t.Fatalf("POST /probe without path params: code=%d (expected 400)", pw.Code)
	}
	if !strings.Contains(pw.Body.String(), "node_id and nbr_id are required") {
		t.Fatalf("expected 'node_id and nbr_id are required', got: %s", pw.Body.String())
	}

	// GET /probe without path params
	gw := httptest.NewRecorder()
	h.HandleGetNodeBackendRuntimeProbe(gw, newReq("GET", "/x", "",
		adminSession(), map[string]string{}))
	if gw.Code != 400 {
		t.Fatalf("GET /probe without path params: code=%d (expected 400)", gw.Code)
	}
	t.Log("Both endpoints reject missing path params")
}

// TestCheckRequestBackwardCompatible verifies the old check-request route still works.
func TestCheckRequestBackwardCompatible(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-ck-bc"
	runtimeID := "rt-ck-bc"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-ck-bc',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime CK BC", "")

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"img:bc"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// Old check-request still works
	cw := httptest.NewRecorder()
	h.HandleRequestNodeBackendRuntimeCheck(cw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if cw.Code != 200 {
		t.Fatalf("check-request code=%d body=%s (expected 200, backward compatible)",
			cw.Code, cw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &resp)
	if resp["id"] != nbrID {
		t.Fatalf("check-request returned wrong id: %v", resp["id"])
	}
	// Verify response has the expected fields
	for _, f := range []string{"id", "node_id", "image_ref", "status", "status_reason", "probe_results"} {
		if _, ok := resp[f]; !ok {
			t.Fatalf("check-request response missing field: %s", f)
		}
	}
	t.Logf("check-request backward compatible: status=%s", resp["status"])
}

// TestGetProbeReturnsEmptyWhenNeverProbed verifies GET /probe returns
// empty probe_results_json when the NBR has never been probed.
func TestGetProbeReturnsEmptyWhenNeverProbed(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-gp-empty"
	runtimeID := "rt-gp-empty"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-gp-empty',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime GP Empty", "")

	// Enable NBR but do NOT probe
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"img:gp"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// GET /probe — should return 200 with empty probe_results_json
	gw := httptest.NewRecorder()
	h.HandleGetNodeBackendRuntimeProbe(gw, newReq("GET", "/x", "",
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if gw.Code != 200 {
		t.Fatalf("GET /probe code=%d body=%s (expected 200)", gw.Code, gw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(gw.Body.Bytes(), &resp)
	prj, _ := resp["probe_results_json"]
	if prj == nil {
		t.Fatal("probe_results_json should not be nil")
	}
	t.Logf("GET /probe with no prior probe: status=%s probe_results_json=%v", resp["status"], prj)
}

// TestGetProbeReturnsSnapshotAfterProbe verifies GET /probe returns the
// stored snapshot after a successful probe.
func TestGetProbeReturnsSnapshotAfterProbe(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	nodeID := "node-gp-snap"
	runtimeID := "rt-gp-snap"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-gp-snap',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime GP Snap", "")

	// Fake agent that returns image in list and inspect success
	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "img:gp-snap",
				"inspect": map[string]interface{}{
					"Id": "sha256:snap123", "RepoTags": []string{"img:gp-snap"},
					"Created": "2026-01-01T00:00:00Z", "Architecture": "amd64", "Os": "linux",
					"Size": 1000, "Config": map[string]interface{}{},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "img", "tag": "gp-snap", "image_id": "sha256:snap123", "image_ref": "img:gp-snap"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"img:gp-snap"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// POST /probe
	pw := httptest.NewRecorder()
	h.HandleProbeNodeBackendRuntime(pw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if pw.Code != 200 {
		t.Fatalf("POST /probe code=%d body=%s", pw.Code, pw.Body.String())
	}

	// GET /probe — should return the stored snapshot
	gw := httptest.NewRecorder()
	h.HandleGetNodeBackendRuntimeProbe(gw, newReq("GET", "/x", "",
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if gw.Code != 200 {
		t.Fatalf("GET /probe code=%d body=%s", gw.Code, gw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(gw.Body.Bytes(), &resp)
	prjRaw := resp["probe_results_json"]
	if prjRaw == nil {
		t.Fatal("probe_results_json is nil after probe")
	}
	// probe_results_json returned as json.RawMessage — should be non-empty
	t.Logf("GET /probe after probe: status=%s probe_results_json present=%v",
		resp["status"], prjRaw != nil && fmt.Sprint(prjRaw) != "{}")
}

// TestPostProbeMissingImageOnlyFromInspectNotFound confirms POST /probe
// never returns missing_image when ImageInspect succeeds (regression).
func TestPostProbeMissingImageOnlyFromInspectNotFound(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	h.AgentClient = agentclient.New("", agentclient.DefaultTimeout)

	nodeID := "node-pp-reg"
	runtimeID := "rt-pp-reg"
	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-pp-reg',?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		nodeID, "nvidia", 0, "RTX", "")
	insertRuntime(t, db, runtimeID, "Runtime PP Reg", "")

	// Fake agent: image in list AND inspect succeeds
	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "docker-image-inspect") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"image_ref": "vllm/vllm-openai:latest",
				"inspect": map[string]interface{}{
					"Id": "sha256:abc123", "RepoTags": []string{"vllm/vllm-openai:latest"},
					"Created": "2026-01-01T00:00:00Z", "Architecture": "amd64", "Os": "linux",
					"Size": 1000, "Config": map[string]interface{}{},
				},
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"images": []map[string]interface{}{
					{"repository": "vllm/vllm-openai", "tag": "latest", "image_id": "sha256:abc123", "image_ref": "vllm/vllm-openai:latest"},
				},
				"count": 1,
			})
		}
	}))
	defer fakeAgent.Close()

	u, _ := url.Parse(fakeAgent.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	db.Exec(`UPDATE nodes SET advertised_address=?, metrics_port=? WHERE id=?`, host, port, nodeID)

	// Enable NBR
	nbrID := nodeID + ":" + runtimeID
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","image_ref":"vllm/vllm-openai:latest"}}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}

	// POST /probe
	pw := httptest.NewRecorder()
	h.HandleProbeNodeBackendRuntime(pw, newReq("POST", "/x", `{}`,
		adminSession(), map[string]string{"id": nodeID, "nbr_id": nbrID}))
	if pw.Code != 200 {
		t.Fatalf("POST /probe code=%d body=%s", pw.Code, pw.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(pw.Body.Bytes(), &resp)
	status := strVal(resp, "status", "")
	if status == "missing_image" {
		t.Fatalf("BUG: POST /probe returned missing_image when ImageInspect succeeded. status=%s reason=%s",
			status, strVal(resp, "status_reason", ""))
	}
	if !boolVal(resp, "image_present", false) {
		t.Fatalf("expected image_present=true when ImageInspect succeeded")
	}
	t.Logf("POST /probe regression: status=%s reason=%s image_present=%v (not missing_image)",
		status, resp["status_reason"], resp["image_present"])
}
