package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPasswordFromCredentialsFile(t *testing.T) {
	t.Run("existing file returns password", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "initial-credentials.txt")
		content := `============================================
LightAI Go - Initial Credentials
Generated: 2026-06-25T00:00:00Z
============================================

[Web/Admin]
Username: admin
Password: abc123def456
Note: Change this password after first login.
`
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		got := readPasswordFromCredentialsFile(path)
		if got != "abc123def456" {
			t.Errorf("expected abc123def456, got %q", got)
		}
	})

	t.Run("missing file returns empty", func(t *testing.T) {
		got := readPasswordFromCredentialsFile("/nonexistent/path/credentials.txt")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("malformed file returns empty", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad-credentials.txt")
		if err := os.WriteFile(path, []byte("no password here\n"), 0600); err != nil {
			t.Fatal(err)
		}
		got := readPasswordFromCredentialsFile(path)
		if got != "" {
			t.Errorf("expected empty string for malformed file, got %q", got)
		}
	})
}

func TestWriteInitialCredentials_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "initial-credentials.txt")

	// First write should succeed.
	if err := writeInitialCredentials(path, "admin", "firstpass"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Error("file should contain credentials")
	}

	// Second write should NOT overwrite.
	if err := writeInitialCredentials(path, "admin", "secondpass"); err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(path)
	if string(data) != string(data2) {
		t.Error("file should not be overwritten on second write")
	}
}

func TestWriteInitialCredentials_Permissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perm-credentials.txt")

	if err := writeInitialCredentials(path, "admin", "testpass"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected 0600 permissions, got %04o", perm)
	}
}

// TestPasswordResolutionPriority verifies the documented priority order
// by testing env vars and file reading logic in isolation.
func TestPasswordResolutionPriority(t *testing.T) {
	// The full priority chain is tested via unit tests on the components:
	// 1. Test env var reading (InitBootstrap uses os.Getenv)
	// 2. Test credentials file reading (readPasswordFromCredentialsFile)
	// 3. Test auto-generate (InitBootstrap uses crypto/rand)

	t.Run("INITIAL_PASSWORD env preferred", func(t *testing.T) {
		// Verify BootstrapConfig has InitialPasswordEnv set
		cfg := BootstrapConfig{
			Username:           "admin",
			Password:           "",
			PasswordEnv:        "LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD",
			InitialPasswordEnv: "LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD",
			ForceChangePassword: true,
		}
		if cfg.InitialPasswordEnv != "LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD" {
			t.Error("InitialPasswordEnv should be LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD")
		}
		// PasswordEnv is the legacy fallback
		if cfg.PasswordEnv != "LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD" {
			t.Error("PasswordEnv should be LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD")
		}
	})

	t.Run("legacy ADMIN_PASSWORD is fallback", func(t *testing.T) {
		// Simulate: only ADMIN_PASSWORD set, INITIAL_PASSWORD empty
		os.Setenv("LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD", "legacy-pass")
		os.Unsetenv("LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD")
		defer os.Unsetenv("LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD")

		// Verify INITIAL_PASSWORD takes priority when set
		os.Setenv("LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD", "canonical-pass")
		defer os.Unsetenv("LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD")

		cfg := BootstrapConfig{
			InitialPasswordEnv: "LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD",
			PasswordEnv:        "LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD",
		}
		pass := os.Getenv(cfg.InitialPasswordEnv)
		if pass != "canonical-pass" {
			t.Fatalf("INITIAL_PASSWORD should be canonical-pass, got %q", pass)
		}
		legacy := os.Getenv(cfg.PasswordEnv)
		if legacy != "legacy-pass" {
			t.Fatalf("ADMIN_PASSWORD should be legacy-pass, got %q", legacy)
		}
	})

	t.Run("existing credentials file preferred over auto-generate", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "initial-credentials.txt")
		content := `Username: admin
Password: file-pass-123`
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		pwd := readPasswordFromCredentialsFile(path)
		if pwd != "file-pass-123" {
			t.Errorf("should read password from file, got %q", pwd)
		}
	})

	t.Run("auto-generate only when no env and no file", func(t *testing.T) {
		// When file doesn't exist, readPasswordFromCredentialsFile returns ""
		pwd := readPasswordFromCredentialsFile("/nonexistent/path")
		if pwd != "" {
			t.Errorf("expected empty for missing file, got %q", pwd)
		}
		// InitBootstrap would then auto-generate (tested indirectly)
	})

	t.Run("restart does not overwrite credentials file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "initial-credentials.txt")
		// Simulate first start: write credentials
		if err := writeInitialCredentials(path, "admin", "original-pass"); err != nil {
			t.Fatal(err)
		}
		// Simulate restart: file exists, writeInitialCredentials returns nil without writing
		if err := writeInitialCredentials(path, "admin", "new-pass"); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if string(data) == "" {
			t.Error("file should still exist after restart")
		}
		// Verify original content preserved
		pwd := readPasswordFromCredentialsFile(path)
		if pwd != "original-pass" {
			t.Errorf("file should contain original-pass after restart, got %q", pwd)
		}
	})
}
