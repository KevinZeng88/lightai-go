package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	"lightai-go/internal/server/rbac"

	"golang.org/x/time/rate"
)

const (
	workflowAdminUsername = "admin"
	workflowAdminPassword = "test1234"
	workflowAgentToken    = "workflow-agent-token"
)

type workflowTestApp struct {
	DB     *db.DB
	Mux    *http.ServeMux
	Server *httptest.Server
	Client *WorkflowClient
}

func newWorkflowTestApp(t *testing.T) *workflowTestApp {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	bootstrapCfg := auth.BootstrapConfig{
		Username:            workflowAdminUsername,
		Password:            workflowAdminPassword,
		ForceChangePassword: false,
	}
	initWorkflowBootstrap(t, database, bootstrapCfg)

	sessionCfg := auth.DefaultSessionConfig()
	sessionStore := auth.NewSessionStore(database, sessionCfg)
	authHandler := &auth.AuthHandler{
		DB:           database,
		SessionStore: sessionStore,
		SessionCfg:   sessionCfg,
		RateLimiter:  auth.NewLoginRateLimiter(rate.Limit(1000), 1000),
		BootstrapCfg: bootstrapCfg,
	}

	agentHandler := NewAgentHandler(database, nil)
	if _, err := agentHandler.ReloadBackendCatalogProjection(); err != nil {
		t.Fatalf("reload backend catalog projection: %v", err)
	}

	mux := http.NewServeMux()
	SetupRoutes(mux, RouterConfig{
		DB:              database,
		AgentToken:      workflowAgentToken,
		SessionStore:    sessionStore,
		SessionCfg:      sessionCfg,
		AuthHandler:     authHandler,
		RBACHandler:     rbac.NewHandler(database),
		AgentHandler:    agentHandler,
		ResourceHandler: NewResourceHandler(database, nil),
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &workflowTestApp{
		DB:     database,
		Mux:    mux,
		Server: server,
		Client: newWorkflowClient(t, server.URL),
	}
}

func initWorkflowBootstrap(t *testing.T, database *db.DB, cfg auth.BootstrapConfig) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("enter temp cwd for bootstrap: %v", err)
	}
	if err := auth.InitBootstrap(database, cfg); err != nil {
		_ = os.Chdir(origDir)
		t.Fatalf("bootstrap auth: %v", err)
	}
	if err := os.Chdir(origDir); err != nil {
		t.Fatalf("restore cwd after bootstrap: %v", err)
	}
}

func (app *workflowTestApp) InsertOnlineNode(t *testing.T, nodeID string, fakeAgent *fakeAgent) string {
	t.Helper()

	advertisedAddress := "127.0.0.1"
	primaryIP := "127.0.0.1"
	metricsPort := 19091
	if fakeAgent != nil {
		host, port := fakeAgent.HostPort(t)
		advertisedAddress = host
		primaryIP = host
		metricsPort = port
	}

	now := time.Now().Format(time.RFC3339)
	_, err := app.DB.Exec(
		`INSERT INTO nodes (
			id, agent_id, hostname, primary_ip, advertised_address,
			metrics_enabled, metrics_scheme, metrics_port, metrics_path,
			status, last_heartbeat_at, tenant_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 1, 'http', ?, '/metrics', 'online', ?, ?, ?, ?)`,
		nodeID,
		"agent-"+nodeID,
		"host-"+nodeID,
		primaryIP,
		advertisedAddress,
		metricsPort,
		now,
		app.DB.DefaultTenantID(),
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert online node: %v", err)
	}
	return nodeID
}

func (app *workflowTestApp) InsertGPU(t *testing.T, nodeID, vendor string) string {
	t.Helper()

	gpuID := "gpu-" + nodeID + "-" + vendor
	now := time.Now().Format(time.RFC3339)
	_, err := app.DB.Exec(
		`INSERT INTO gpu_devices (
			id, node_id, vendor, index_num, name, uuid, tenant_id,
			memory_total_bytes, health, status, collected_at, reported_at, created_at, updated_at
		) VALUES (?, ?, ?, 0, ?, ?, ?, 1024, 'healthy', 'available', ?, ?, ?, ?)`,
		gpuID,
		nodeID,
		vendor,
		"Workflow "+vendor+" GPU",
		"uuid-"+gpuID,
		app.DB.DefaultTenantID(),
		now,
		now,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert GPU: %v", err)
	}
	return gpuID
}

func (app *workflowTestApp) FindBackendRuntimeID(t *testing.T, backendName, vendor string) string {
	t.Helper()

	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes", nil, http.StatusOK)
	var runtimes []map[string]interface{}
	resp.Decode(t, &runtimes)
	for _, runtime := range runtimes {
		id, _ := runtime["id"].(string)
		runtimeVendor, _ := runtime["vendor"].(string)
		if runtimeVendor == vendor && strings.Contains(strings.ToLower(id), strings.ToLower(backendName)) {
			return id
		}
	}
	t.Fatalf("backend runtime not found backend=%q vendor=%q in %#v", backendName, vendor, runtimes)
	return ""
}

func (app *workflowTestApp) EnableNodeBackendRuntime(t *testing.T, nodeID, runtimeID, imageRef string) string {
	t.Helper()

	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          imageRef,
	}, http.StatusOK)
	var enabled map[string]interface{}
	resp.Decode(t, &enabled)
	id, _ := enabled["id"].(string)
	if id == "" {
		t.Fatalf("enable node backend runtime response missing id: %#v", enabled)
	}
	return id
}

type WorkflowClient struct {
	baseURL   string
	origin    string
	http      *http.Client
	csrfToken string
}

type WorkflowResponse struct {
	StatusCode int
	Body       []byte
}

func newWorkflowClient(t *testing.T, baseURL string) *WorkflowClient {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	return &WorkflowClient{
		baseURL: baseURL,
		origin:  baseURL,
		http:    &http.Client{Jar: jar},
	}
}

func (c *WorkflowClient) LoginAsAdmin(t *testing.T) {
	t.Helper()

	resp := c.JSON(t, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{
		"username": workflowAdminUsername,
		"password": workflowAdminPassword,
	}, http.StatusOK)

	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	resp.Decode(t, &body)
	if body.CSRFToken == "" {
		t.Fatalf("login response missing csrf_token: %s", string(resp.Body))
	}
	c.csrfToken = body.CSRFToken
}

func (c *WorkflowClient) JSON(t *testing.T, method, path string, body interface{}, wantStatus int) WorkflowResponse {
	t.Helper()
	return c.json(t, method, path, body, wantStatus, true)
}

func (c *WorkflowClient) JSONWithoutCSRF(t *testing.T, method, path string, body interface{}, wantStatus int) WorkflowResponse {
	t.Helper()
	return c.json(t, method, path, body, wantStatus, false)
}

func (c *WorkflowClient) json(t *testing.T, method, path string, body interface{}, wantStatus int, includeCSRF bool) WorkflowResponse {
	t.Helper()

	bodyReader, err := marshalWorkflowBody(body)
	if err != nil {
		t.Fatalf("%s %s marshal body: %v", method, path, err)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("%s %s build request: %v", method, path, err)
	}
	req.Header.Set("Origin", c.origin)
	req.Header.Set("Content-Type", "application/json")
	if includeCSRF && c.csrfToken != "" && method != http.MethodGet {
		req.Header.Set(auth.CSRFHeader, c.csrfToken)
	}

	httpResp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s request failed: %v", method, path, err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("%s %s read response: %v", method, path, err)
	}

	if httpResp.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, httpResp.StatusCode, wantStatus, string(respBody))
	}
	return WorkflowResponse{StatusCode: httpResp.StatusCode, Body: respBody}
}

func marshalWorkflowBody(body interface{}) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	switch v := body.(type) {
	case string:
		return strings.NewReader(v), nil
	case []byte:
		return bytes.NewReader(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}
}

func (r WorkflowResponse) Decode(t *testing.T, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(r.Body, out); err != nil {
		t.Fatalf("decode response JSON: %v body=%s", err, string(r.Body))
	}
}

func splitWorkflowHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL %q: %v", rawURL, err)
	}
	host := parsed.Hostname()
	portText := parsed.Port()
	if host == "" || portText == "" {
		t.Fatalf("URL %q missing host or port", rawURL)
	}
	var port int
	if _, err := fmt.Sscanf(portText, "%d", &port); err != nil {
		t.Fatalf("parse port %q: %v", portText, err)
	}
	return host, port
}
