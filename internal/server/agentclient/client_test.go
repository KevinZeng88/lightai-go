package agentclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestValidateAgentAddress_Localhost(t *testing.T) {
	if err := ValidateAgentAddress("127.0.0.1"); err != nil {
		t.Errorf("localhost should be allowed: %v", err)
	}
}

func TestValidateAgentAddress_PrivateIP(t *testing.T) {
	for _, addr := range []string{"10.0.0.1", "192.168.1.1", "172.16.0.1"} {
		if err := ValidateAgentAddress(addr); err != nil {
			t.Errorf("private IP %s should be allowed: %v", addr, err)
		}
	}
}

func TestValidateAgentAddress_Metadata(t *testing.T) {
	if err := ValidateAgentAddress("169.254.169.254"); err == nil {
		t.Error("metadata endpoint should be denied")
	}
}

func TestValidateAgentAddress_LinkLocal(t *testing.T) {
	if err := ValidateAgentAddress("169.254.0.1"); err == nil {
		t.Error("link-local should be denied")
	}
}

func TestValidateAgentAddress_Unspecified(t *testing.T) {
	if err := ValidateAgentAddress("0.0.0.0"); err == nil {
		t.Error("unspecified should be denied")
	}
}

func TestValidateAgentAddress_Multicast(t *testing.T) {
	if err := ValidateAgentAddress("224.0.0.1"); err == nil {
		t.Error("multicast should be denied")
	}
}

func TestValidateAgentAddress_Hostname(t *testing.T) {
	if err := ValidateAgentAddress("my-agent.local"); err != nil {
		t.Errorf("hostname should be allowed: %v", err)
	}
}

func TestGetJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := New("test-token", 5*time.Second)
	// Use the test server address
	body, code, err := c.GetJSON(context.Background(), "127.0.0.1", extractPort(srv.URL), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 200 {
		t.Errorf("expected 200, got %d", code)
	}
	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestGetJSON_DeniedAddress(t *testing.T) {
	c := New("test-token", 5*time.Second)
	_, _, err := c.GetJSON(context.Background(), "169.254.169.254", 80, "/test", nil)
	if err == nil {
		t.Error("expected error for metadata address")
	}
}

func TestGetJSON_URLEncoding(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New("test-token", 5*time.Second)
	params := url.Values{"path": {"/models/test dir"}, "glob": {"*.gguf"}}
	_, _, err := c.GetJSON(context.Background(), "127.0.0.1", extractPort(srv.URL), "/files", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedQuery == "" {
		t.Error("query not received")
	}
}

func TestGetJSON_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New("test-token", 100*time.Millisecond)
	_, _, err := c.GetJSON(context.Background(), "127.0.0.1", extractPort(srv.URL), "/test", nil)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestPostJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"result":"created"}`))
	}))
	defer srv.Close()

	c := New("test-token", 5*time.Second)
	body, _, err := c.PostJSON(context.Background(), "127.0.0.1", extractPort(srv.URL), "/scan", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"result":"created"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func extractPort(serverURL string) int {
	u, _ := url.Parse(serverURL)
	port := 80
	if p := u.Port(); p != "" {
		// Parse port
		switch p {
		case "80":
			port = 80
		default:
			// Just use a dummy port for testing
			port = 19091
		}
	}
	// For httptest, extract actual port from URL
	u2, _ := url.Parse(serverURL)
	if u2.Port() != "" {
		p := 0
		for _, c := range u2.Port() {
			p = p*10 + int(c-'0')
		}
		if p > 0 {
			return p
		}
	}
	return port
}
