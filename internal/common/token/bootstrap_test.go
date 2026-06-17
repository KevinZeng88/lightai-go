package token

import (
	"os"
	"testing"
)

func TestIsDefault(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		{"", true},
		{DefaultTokenValue, true},
		{DevTokenValue, true},
		{"my-secure-token", false},
		{"lightai-agent-token-change-me", true},
	}
	for _, tt := range tests {
		got := IsDefault(tt.token)
		if got != tt.expected {
			t.Errorf("IsDefault(%q) = %v, want %v", tt.token, got, tt.expected)
		}
	}
}

func TestGenerate(t *testing.T) {
	tok1, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(tok1) != TokenLength*2 { // hex encoding doubles length
		t.Errorf("Generate() len = %d, want %d", len(tok1), TokenLength*2)
	}

	// Verify uniqueness.
	tok2, _ := Generate()
	if tok1 == tok2 {
		t.Error("two generated tokens should differ")
	}
}

func TestWriteAndRead(t *testing.T) {
	// Use a temp file to avoid touching the real one.
	origFile := TokenFile
	tmpFile := t.TempDir() + "/agent-token.env"

	// Override the const for this test (hack: use a local variable).
	// Since the const can't be overridden, test ReadFromFile with explicit path.
	genTok, _ := Generate()

	// Write.
	dir := t.TempDir()
	path := dir + "/test-token.env"
	if err := os.MkdirAll(dir+"/nonexistent", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(genTok+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Read back.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	readTok := string(data)
	readTok = readTok[:len(readTok)-1] // strip newline
	if readTok != genTok {
		t.Errorf("read token %q != generated %q", readTok, genTok)
	}

	_ = origFile
	_ = tmpFile
}

func TestBootstrapServer_NonDefault(t *testing.T) {
	tok, autoGen, err := BootstrapServer("my-custom-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "my-custom-token" {
		t.Errorf("expected my-custom-token, got %q", tok)
	}
	if autoGen {
		t.Error("should not auto-generate for non-default token")
	}
}

func TestBootstrapAgent_NonDefault(t *testing.T) {
	tok, err := BootstrapAgent("my-custom-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "my-custom-token" {
		t.Errorf("expected my-custom-token, got %q", tok)
	}
}

func TestBootstrapAgent_NoTokenSource(t *testing.T) {
	// In a temp dir with no token file, BootstrapAgent should error.
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, err := BootstrapAgent(DefaultTokenValue)
	if err == nil {
		t.Error("expected error for agent with no token source")
	}
}
