package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type fakeAgent struct {
	server   *httptest.Server
	mu       sync.Mutex
	counts   map[string]int
	scenario fakeAgentScenario
}

type fakeAgentScenario struct {
	Images       []fakeAgentImage
	Inspect      map[string]interface{}
	InspectCode  int
	Files        map[string]interface{}
	FilesCode    int
	Scan         map[string]interface{}
	ScanCode     int
	ResponseCode map[string]int
}

type fakeAgentImage struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ImageRef   string `json:"image_ref"`
	ImageID    string `json:"image_id"`
	Digest     string `json:"digest,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	Size       int64  `json:"size,omitempty"`
}

func newFakeAgent(t *testing.T, scenario fakeAgentScenario) *fakeAgent {
	t.Helper()

	agent := &fakeAgent{
		counts:   map[string]int{},
		scenario: scenario,
	}
	agent.server = httptest.NewServer(http.HandlerFunc(agent.handle))
	t.Cleanup(agent.server.Close)
	return agent
}

func (a *fakeAgent) HostPort(t *testing.T) (string, int) {
	t.Helper()
	return splitWorkflowHostPort(t, a.server.URL)
}

func (a *fakeAgent) RequestCount(path string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.counts[path]
}

func (a *fakeAgent) handle(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	a.counts[r.URL.Path]++
	a.mu.Unlock()

	switch r.URL.Path {
	case "/healthz":
		writeFakeAgentJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
	case "/docker-images":
		code := a.statusCode("/docker-images", http.StatusOK)
		writeFakeAgentJSON(w, code, map[string]interface{}{
			"images": a.scenario.Images,
			"count":  len(a.scenario.Images),
		})
	case "/docker-image-inspect":
		code := a.scenario.InspectCode
		if code == 0 {
			code = a.statusCode("/docker-image-inspect", http.StatusOK)
		}
		payload := a.scenario.Inspect
		if payload == nil {
			payload = defaultFakeAgentInspect(r.URL.Query().Get("ref"))
		}
		writeFakeAgentJSON(w, code, payload)
	case "/files":
		code := a.scenario.FilesCode
		if code == 0 {
			code = a.statusCode("/files", http.StatusOK)
		}
		payload := a.scenario.Files
		if payload == nil {
			payload = defaultFakeAgentFiles()
		}
		writeFakeAgentJSON(w, code, payload)
	case "/model-paths/scan":
		code := a.scenario.ScanCode
		if code == 0 {
			code = a.statusCode("/model-paths/scan", http.StatusOK)
		}
		payload := a.scenario.Scan
		if payload == nil {
			payload = defaultFakeAgentScan()
		}
		writeFakeAgentJSON(w, code, payload)
	default:
		writeFakeAgentJSON(w, http.StatusNotFound, map[string]interface{}{"error": "not found"})
	}
}

func (a *fakeAgent) statusCode(path string, fallback int) int {
	if a.scenario.ResponseCode == nil || a.scenario.ResponseCode[path] == 0 {
		return fallback
	}
	return a.scenario.ResponseCode[path]
}

func writeFakeAgentJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func defaultFakeAgentInspect(ref string) map[string]interface{} {
	if ref == "" {
		ref = "vllm/vllm-openai:latest"
	}
	return map[string]interface{}{
		"inspect": map[string]interface{}{
			"Id":       "sha256:workflow-fake-inspect",
			"RepoTags": []string{ref},
			"Config": map[string]interface{}{
				"Entrypoint": []string{"python3", "-m", "vllm.entrypoints.openai.api_server"},
				"Cmd":        []string{},
				"Env":        []string{"PATH=/usr/local/bin"},
			},
			"Size": 123456789,
		},
	}
}

func defaultFakeAgentFiles() map[string]interface{} {
	return map[string]interface{}{
		"entries": []map[string]interface{}{
			{
				"name":          "Qwen3-0.6B-Instruct-2512",
				"path":          "Qwen3-0.6B-Instruct-2512",
				"type":          "directory",
				"size_bytes":    0,
				"modified_time": "2026-06-20T00:00:00Z",
			},
		},
	}
}

func defaultFakeAgentScan() map[string]interface{} {
	return map[string]interface{}{
		"discovered_name": "Qwen3-0.6B-Instruct-2512",
		"format":          "huggingface",
		"architecture":    "qwen3",
		"size_bytes":      123456789,
		"capabilities":    []string{"chat"},
		"metadata": map[string]interface{}{
			"source": "fake-agent",
		},
	}
}
