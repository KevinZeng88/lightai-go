package api

import (
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestWorkflowModelWizardCreateArtifactWithLocation(t *testing.T) {
	app := newWorkflowTestApp(t)
	fixture := newWorkflowModelWizardFixture(t, app, "create", workflowModelScanPayload("create", "/models/create/Qwen3-0.6B-Instruct"))
	app.Client.LoginAsAdmin(t)
	fixture.ensureRoot(t, app)

	workflowAssertNodeVisible(t, app, fixture.NodeID)
	workflowBrowseNodeFiles(t, app, fixture.NodeID, fixture.RootID, "")
	scan := workflowScanModelPath(t, app, fixture.NodeID, fixture.RootID, "Qwen3-0.6B-Instruct")
	workflowAssertFakeAgentCalls(t, fixture.Agent, "/files", 1)
	workflowAssertFakeAgentCalls(t, fixture.Agent, "/model-paths/scan", 1)

	artifact, markDeleted := workflowCreateModelArtifactFromScan(t, app, "create", scan)
	artifactID := workflowStringField(t, artifact, "id")
	location := workflowCreateModelLocationFromScan(t, app, artifactID, fixture.NodeID, scan, "create")

	detail := workflowGetModelArtifact(t, app, artifactID, http.StatusOK)
	workflowAssertModelArtifactListDetailConsistent(t, app, detail)
	workflowAssertArtifactFieldsFromScan(t, detail, scan)
	workflowAssertLocationFromScan(t, location, fixture.NodeID, scan, "create")
	workflowAssertArtifactDetailHasLocation(t, detail, location)

	workflowDeleteModelArtifact(t, app, artifactID)
	markDeleted()
	workflowGetModelArtifact(t, app, artifactID, http.StatusNotFound)
	workflowAssertModelArtifactNotListed(t, app, artifactID)
}

func TestWorkflowModelWizardScanMetadataPreserved(t *testing.T) {
	app := newWorkflowTestApp(t)
	scanPayload := workflowModelScanPayload("metadata", "/models/metadata/Qwen3-0.6B-Instruct")
	fixture := newWorkflowModelWizardFixture(t, app, "metadata", scanPayload)
	app.Client.LoginAsAdmin(t)
	fixture.ensureRoot(t, app)

	scan := workflowScanModelPath(t, app, fixture.NodeID, fixture.RootID, "Qwen3-0.6B-Instruct")
	workflowAssertScanMetadata(t, scan, scanPayload)

	artifact, markDeleted := workflowCreateModelArtifactFromScan(t, app, "metadata", scan)
	artifactID := workflowStringField(t, artifact, "id")
	location := workflowCreateModelLocationFromScan(t, app, artifactID, fixture.NodeID, scan, "metadata")

	metadata := workflowMapField(t, location, "discovered_metadata_json")
	workflowAssertScanMetadataEmbedded(t, metadata, scanPayload)

	detail := workflowGetModelArtifact(t, app, artifactID, http.StatusOK)
	storedLocation := workflowArtifactLocationByID(t, detail, workflowStringField(t, location, "id"))
	storedMetadata := workflowMapField(t, storedLocation, "discovered_metadata_json")
	workflowAssertScanMetadataEmbedded(t, storedMetadata, scanPayload)

	workflowDeleteModelArtifact(t, app, artifactID)
	markDeleted()
}

func TestWorkflowModelWizardMultipleLocationsDoNotMix(t *testing.T) {
	app := newWorkflowTestApp(t)
	firstScanPayload := workflowModelScanPayload("multi-a", "/models/multi-a/Qwen3-0.6B-Instruct")
	secondScanPayload := workflowModelScanPayload("multi-b", "/models/multi-b/Qwen3-0.6B-Instruct")
	first := newWorkflowModelWizardFixture(t, app, "multi-a", firstScanPayload)
	second := newWorkflowModelWizardFixture(t, app, "multi-b", secondScanPayload)
	app.Client.LoginAsAdmin(t)
	first.ensureRoot(t, app)
	second.ensureRoot(t, app)

	firstScan := workflowScanModelPath(t, app, first.NodeID, first.RootID, "Qwen3-0.6B-Instruct")
	secondScan := workflowScanModelPath(t, app, second.NodeID, second.RootID, "Qwen3-0.6B-Instruct")

	artifact, markDeleted := workflowCreateModelArtifactFromScan(t, app, "multi", firstScan)
	artifactID := workflowStringField(t, artifact, "id")
	firstLocation := workflowCreateModelLocationFromScan(t, app, artifactID, first.NodeID, firstScan, "multi-a")
	secondLocation := workflowCreateModelLocationFromScan(t, app, artifactID, second.NodeID, secondScan, "multi-b")

	detail := workflowGetModelArtifact(t, app, artifactID, http.StatusOK)
	locations := workflowLocationsField(t, detail)
	if len(locations) != 2 {
		t.Fatalf("locations length=%d want 2 detail=%#v", len(locations), detail)
	}
	storedFirst := workflowArtifactLocationByID(t, detail, workflowStringField(t, firstLocation, "id"))
	storedSecond := workflowArtifactLocationByID(t, detail, workflowStringField(t, secondLocation, "id"))
	workflowAssertLocationFromScan(t, storedFirst, first.NodeID, firstScan, "multi-a")
	workflowAssertLocationFromScan(t, storedSecond, second.NodeID, secondScan, "multi-b")
	if storedFirst["node_id"] == storedSecond["node_id"] || storedFirst["absolute_path"] == storedSecond["absolute_path"] || storedFirst["checksum"] == storedSecond["checksum"] {
		t.Fatalf("multiple locations are mixed or overwritten: first=%#v second=%#v", storedFirst, storedSecond)
	}

	workflowDeleteModelLocation(t, app, artifactID, workflowStringField(t, firstLocation, "id"))
	afterLocationDelete := workflowGetModelArtifact(t, app, artifactID, http.StatusOK)
	remaining := workflowLocationsField(t, afterLocationDelete)
	if len(remaining) != 1 || remaining[0]["id"] != workflowStringField(t, secondLocation, "id") {
		t.Fatalf("location cleanup removed wrong location: %#v", remaining)
	}

	workflowDeleteModelArtifact(t, app, artifactID)
	markDeleted()
}

func TestWorkflowModelWizardDeleteCleanup(t *testing.T) {
	app := newWorkflowTestApp(t)
	fixture := newWorkflowModelWizardFixture(t, app, "delete", workflowModelScanPayload("delete", "/models/delete/Qwen3-0.6B-Instruct"))
	app.Client.LoginAsAdmin(t)
	fixture.ensureRoot(t, app)

	scan := workflowScanModelPath(t, app, fixture.NodeID, fixture.RootID, "Qwen3-0.6B-Instruct")
	artifact, markDeleted := workflowCreateModelArtifactFromScan(t, app, "delete", scan)
	artifactID := workflowStringField(t, artifact, "id")
	location := workflowCreateModelLocationFromScan(t, app, artifactID, fixture.NodeID, scan, "delete")

	workflowDeleteModelLocation(t, app, artifactID, workflowStringField(t, location, "id"))
	afterLocationDelete := workflowGetModelArtifact(t, app, artifactID, http.StatusOK)
	if locations := workflowLocationsField(t, afterLocationDelete); len(locations) != 0 {
		t.Fatalf("location still visible after delete: %#v", locations)
	}

	workflowDeleteModelArtifact(t, app, artifactID)
	markDeleted()
	workflowGetModelArtifact(t, app, artifactID, http.StatusNotFound)
	workflowAssertModelArtifactNotListed(t, app, artifactID)
}

type workflowModelWizardFixture struct {
	NodeID string
	RootID string
	Root   string
	Agent  *fakeAgent
}

func newWorkflowModelWizardFixture(t *testing.T, app *workflowTestApp, suffix string, scan map[string]interface{}) workflowModelWizardFixture {
	t.Helper()

	files := map[string]interface{}{
		"entries": []map[string]interface{}{
			{
				"name":          "Qwen3-0.6B-Instruct",
				"path":          "Qwen3-0.6B-Instruct",
				"type":          "directory",
				"size_bytes":    scan["size_bytes"],
				"modified_time": "2026-06-20T00:00:00Z",
			},
		},
	}
	agent := newFakeAgent(t, fakeAgentScenario{
		Files: files,
		Scan:  scan,
	})
	nodeID := "workflow-model-node-" + suffix
	app.InsertOnlineNode(t, nodeID, agent)
	root := t.TempDir()
	return workflowModelWizardFixture{
		NodeID: nodeID,
		Root:   root,
		Agent:  agent,
	}
}

func workflowModelScanPayload(suffix, absolutePath string) map[string]interface{} {
	return map[string]interface{}{
		"discovered_name": "Qwen3-0.6B-Instruct-" + suffix,
		"format":          "huggingface",
		"architecture":    "qwen3",
		"size_bytes":      float64(123456789 + len(suffix)),
		"checksum":        "sha256:workflow-" + suffix,
		"capabilities":    []interface{}{"chat", "text-generation"},
		"metadata": map[string]interface{}{
			"source":         "fake-agent",
			"workflow":       suffix,
			"parameter_size": "0.6B",
			"dtype":          "bf16",
		},
		"absolute_path": absolutePath,
	}
}

func workflowAssertNodeVisible(t *testing.T, app *workflowTestApp, nodeID string) {
	t.Helper()
	nodesResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes", nil, http.StatusOK)
	var nodes []map[string]interface{}
	nodesResp.Decode(t, &nodes)
	if !workflowListContainsID(nodes, nodeID) {
		t.Fatalf("node %q missing from GET /api/v1/nodes: %#v", nodeID, nodes)
	}
}

func workflowAddModelRoot(t *testing.T, app *workflowTestApp, nodeID, root string) string {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/model-roots", map[string]interface{}{
		"path":        root,
		"description": "workflow model root",
	}, http.StatusCreated)
	var payload map[string]interface{}
	resp.Decode(t, &payload)
	id := workflowStringField(t, payload, "id")
	if payload["path"] != root {
		t.Fatalf("model root path=%#v want %#v response=%#v", payload["path"], root, payload)
	}
	return id
}

func workflowBrowseNodeFiles(t *testing.T, app *workflowTestApp, nodeID, rootID, rel string) map[string]interface{} {
	t.Helper()
	path := "/api/v1/nodes/" + nodeID + "/files?root_id=" + url.QueryEscape(rootID) + "&path=" + url.QueryEscape(rel) + "&limit=20"
	resp := app.Client.JSON(t, http.MethodGet, path, nil, http.StatusOK)
	var payload map[string]interface{}
	resp.Decode(t, &payload)
	entries, ok := payload["entries"].([]interface{})
	if !ok || len(entries) == 0 {
		t.Fatalf("file browse returned no entries: %#v", payload)
	}
	if payload["root_id"] != rootID {
		t.Fatalf("file browse root_id=%#v want %#v payload=%#v", payload["root_id"], rootID, payload)
	}
	return payload
}

func workflowScanModelPath(t *testing.T, app *workflowTestApp, nodeID, rootID, rel string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/model-paths/scan", map[string]interface{}{
		"root_id":       rootID,
		"relative_path": rel,
	}, http.StatusOK)
	var payload map[string]interface{}
	resp.Decode(t, &payload)
	if payload["root_id"] != rootID {
		t.Fatalf("scan root_id=%#v want %#v payload=%#v", payload["root_id"], rootID, payload)
	}
	if payload["relative_path"] != rel {
		t.Fatalf("scan relative_path=%#v want %#v payload=%#v", payload["relative_path"], rel, payload)
	}
	workflowStringField(t, payload, "absolute_path")
	workflowStringField(t, payload, "format")
	workflowStringField(t, payload, "architecture")
	workflowMapField(t, payload, "metadata")
	if _, ok := payload["capabilities"].([]interface{}); !ok {
		t.Fatalf("scan capabilities missing or not array: %#v", payload)
	}
	return payload
}

func workflowCreateModelArtifactFromScan(t *testing.T, app *workflowTestApp, suffix string, scan map[string]interface{}) (map[string]interface{}, func()) {
	t.Helper()
	name := "workflow-model-" + suffix
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":                   name,
		"display_name":           "Workflow Model " + suffix,
		"source_type":            "local_path",
		"path":                   workflowStringField(t, scan, "absolute_path"),
		"format":                 workflowStringField(t, scan, "format"),
		"task_type":              "chat",
		"architecture":           workflowStringField(t, scan, "architecture"),
		"size_label":             "0.6B",
		"quantization":           "bf16",
		"default_context_length": 32768,
		"estimated_vram_bytes":   2147483648,
		"required_gpu_count":     1,
	}, http.StatusCreated)
	var artifact map[string]interface{}
	resp.Decode(t, &artifact)
	id := workflowStringField(t, artifact, "id")

	deleted := false
	markDeleted := func() {
		deleted = true
	}
	t.Cleanup(func() {
		if deleted {
			return
		}
		app.Client.JSON(t, http.MethodDelete, "/api/v1/model-artifacts/"+id, nil, http.StatusOK)
	})
	return artifact, markDeleted
}

func workflowCreateModelLocationFromScan(t *testing.T, app *workflowTestApp, artifactID, nodeID string, scan map[string]interface{}, suffix string) map[string]interface{} {
	t.Helper()
	metadata := map[string]interface{}{
		"scan_metadata": workflowMapField(t, scan, "metadata"),
		"capabilities":  scan["capabilities"],
		"format":        scan["format"],
		"architecture":  scan["architecture"],
	}
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts/"+artifactID+"/locations", map[string]interface{}{
		"node_id":                  nodeID,
		"path_type":                "directory",
		"absolute_path":            workflowStringField(t, scan, "absolute_path"),
		"size_bytes":               scan["size_bytes"],
		"checksum":                 workflowStringField(t, scan, "checksum"),
		"manifest_digest":          "manifest-" + suffix,
		"discovered_metadata_json": metadata,
		"match_status":             "exact_match",
		"verification_status":      "verified",
	}, http.StatusCreated)
	var location map[string]interface{}
	resp.Decode(t, &location)
	return location
}

func workflowGetModelArtifact(t *testing.T, app *workflowTestApp, id string, wantStatus int) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts/"+id, nil, wantStatus)
	if wantStatus != http.StatusOK {
		return nil
	}
	var artifact map[string]interface{}
	resp.Decode(t, &artifact)
	return artifact
}

func workflowDeleteModelLocation(t *testing.T, app *workflowTestApp, artifactID, locationID string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodDelete, "/api/v1/model-artifacts/"+artifactID+"/locations/"+locationID, nil, http.StatusOK)
	var payload map[string]interface{}
	resp.Decode(t, &payload)
	if payload["status"] != "deleted" {
		t.Fatalf("location delete response=%#v", payload)
	}
}

func workflowDeleteModelArtifact(t *testing.T, app *workflowTestApp, id string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodDelete, "/api/v1/model-artifacts/"+id, nil, http.StatusOK)
	var payload map[string]interface{}
	resp.Decode(t, &payload)
	if payload["status"] != "deleted" {
		t.Fatalf("artifact delete response=%#v", payload)
	}
}

func workflowAssertModelArtifactListDetailConsistent(t *testing.T, app *workflowTestApp, detail map[string]interface{}) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts", nil, http.StatusOK)
	var artifacts []map[string]interface{}
	resp.Decode(t, &artifacts)
	listItem := workflowFindByID(t, artifacts, workflowStringField(t, detail, "id"))
	for _, field := range []string{
		"id",
		"name",
		"display_name",
		"source_type",
		"path",
		"format",
		"task_type",
		"architecture",
		"size_label",
		"quantization",
		"default_context_length",
		"estimated_vram_bytes",
		"required_gpu_count",
		"tenant_id",
	} {
		if !reflect.DeepEqual(listItem[field], detail[field]) {
			t.Fatalf("model artifact list/detail mismatch field %q: list=%#v detail=%#v", field, listItem[field], detail[field])
		}
	}
}

func workflowAssertArtifactFieldsFromScan(t *testing.T, artifact map[string]interface{}, scan map[string]interface{}) {
	t.Helper()
	if artifact["path"] != scan["absolute_path"] {
		t.Fatalf("artifact path=%#v want scan absolute_path=%#v artifact=%#v scan=%#v", artifact["path"], scan["absolute_path"], artifact, scan)
	}
	if artifact["format"] != scan["format"] || artifact["architecture"] != scan["architecture"] {
		t.Fatalf("artifact format/architecture not from scan: artifact=%#v scan=%#v", artifact, scan)
	}
	if artifact["required_gpu_count"] != float64(1) || artifact["default_context_length"] != float64(32768) {
		t.Fatalf("artifact numeric fields not preserved: %#v", artifact)
	}
}

func workflowAssertLocationFromScan(t *testing.T, location map[string]interface{}, nodeID string, scan map[string]interface{}, suffix string) {
	t.Helper()
	if location["node_id"] != nodeID {
		t.Fatalf("location node_id=%#v want %#v location=%#v", location["node_id"], nodeID, location)
	}
	if location["absolute_path"] != scan["absolute_path"] {
		t.Fatalf("location absolute_path=%#v want %#v location=%#v", location["absolute_path"], scan["absolute_path"], location)
	}
	if location["checksum"] != scan["checksum"] {
		t.Fatalf("location checksum=%#v want %#v location=%#v", location["checksum"], scan["checksum"], location)
	}
	if location["manifest_digest"] != "manifest-"+suffix {
		t.Fatalf("location manifest_digest=%#v suffix=%s location=%#v", location["manifest_digest"], suffix, location)
	}
	workflowAssertNumberField(t, location, "size_bytes", scan["size_bytes"])
	metadata := workflowMapField(t, location, "discovered_metadata_json")
	workflowAssertCapabilities(t, metadata["capabilities"], scan["capabilities"])
	scanMetadata := workflowMapField(t, metadata, "scan_metadata")
	if !reflect.DeepEqual(scanMetadata, workflowMapField(t, scan, "metadata")) {
		t.Fatalf("location metadata mismatch: got=%#v want=%#v", scanMetadata, scan["metadata"])
	}
}

func workflowAssertArtifactDetailHasLocation(t *testing.T, detail, location map[string]interface{}) {
	t.Helper()
	stored := workflowArtifactLocationByID(t, detail, workflowStringField(t, location, "id"))
	if !reflect.DeepEqual(stored, location) {
		t.Fatalf("artifact detail location differs:\n got=%#v\nwant=%#v", stored, location)
	}
}

func workflowAssertScanMetadata(t *testing.T, scan, want map[string]interface{}) {
	t.Helper()
	workflowAssertCapabilities(t, scan["capabilities"], want["capabilities"])
	if !reflect.DeepEqual(workflowMapField(t, scan, "metadata"), workflowMapField(t, want, "metadata")) {
		t.Fatalf("scan metadata mismatch: got=%#v want=%#v", scan["metadata"], want["metadata"])
	}
}

func workflowAssertScanMetadataEmbedded(t *testing.T, metadata, scanPayload map[string]interface{}) {
	t.Helper()
	workflowAssertCapabilities(t, metadata["capabilities"], scanPayload["capabilities"])
	if metadata["format"] != scanPayload["format"] || metadata["architecture"] != scanPayload["architecture"] {
		t.Fatalf("embedded format/architecture mismatch: metadata=%#v scan=%#v", metadata, scanPayload)
	}
	scanMetadata := workflowMapField(t, metadata, "scan_metadata")
	if !reflect.DeepEqual(scanMetadata, workflowMapField(t, scanPayload, "metadata")) {
		t.Fatalf("embedded scan metadata mismatch: got=%#v want=%#v", scanMetadata, scanPayload["metadata"])
	}
}

func workflowLocationsField(t *testing.T, artifact map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw, ok := artifact["locations"].([]interface{})
	if !ok {
		t.Fatalf("artifact locations missing or not array: %#v", artifact)
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		location, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("artifact location not object: %#v", item)
		}
		out = append(out, location)
	}
	return out
}

func workflowArtifactLocationByID(t *testing.T, artifact map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	for _, location := range workflowLocationsField(t, artifact) {
		if location["id"] == id {
			return location
		}
	}
	t.Fatalf("location %q not found in artifact detail: %#v", id, artifact)
	return nil
}

func workflowAssertModelArtifactNotListed(t *testing.T, app *workflowTestApp, id string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts", nil, http.StatusOK)
	var artifacts []map[string]interface{}
	resp.Decode(t, &artifacts)
	if workflowListContainsID(artifacts, id) {
		t.Fatalf("model artifact %q still visible after delete: %#v", id, artifacts)
	}
}

func workflowAssertCapabilities(t *testing.T, got, want interface{}) {
	t.Helper()
	gotSlice, ok := got.([]interface{})
	if !ok || gotSlice == nil {
		t.Fatalf("capabilities missing or not array: %#v", got)
	}
	wantSlice, ok := want.([]interface{})
	if !ok || wantSlice == nil {
		t.Fatalf("expected capabilities not array: %#v", want)
	}
	if !reflect.DeepEqual(gotSlice, wantSlice) {
		t.Fatalf("capabilities mismatch: got=%#v want=%#v", gotSlice, wantSlice)
	}
}

func workflowAssertNumberField(t *testing.T, payload map[string]interface{}, field string, want interface{}) {
	t.Helper()
	gotValue, ok := payload[field].(float64)
	if !ok {
		t.Fatalf("field %q missing or not number in %#v", field, payload)
	}
	var wantValue float64
	switch v := want.(type) {
	case float64:
		wantValue = v
	case int:
		wantValue = float64(v)
	case int64:
		wantValue = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("parse expected number %q for field %q: %v", v, field, err)
		}
		wantValue = parsed
	default:
		t.Fatalf("unsupported expected number type %T for field %q", want, field)
	}
	if gotValue != wantValue {
		t.Fatalf("field %q=%v want %v in %#v", field, gotValue, wantValue, payload)
	}
}

func workflowAssertFakeAgentCalls(t *testing.T, agent *fakeAgent, path string, want int) {
	t.Helper()
	if got := agent.RequestCount(path); got != want {
		t.Fatalf("fake agent %s calls=%d want %d", path, got, want)
	}
}

func (f *workflowModelWizardFixture) ensureRoot(t *testing.T, app *workflowTestApp) {
	t.Helper()
	if f.RootID != "" {
		return
	}
	f.RootID = workflowAddModelRoot(t, app, f.NodeID, filepath.Clean(f.Root))
}
