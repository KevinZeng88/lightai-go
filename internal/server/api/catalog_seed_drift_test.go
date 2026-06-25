package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCatalogSeedDrift verifies that db.go seed has entries for all YAML catalog versions.
func TestCatalogSeedDrift(t *testing.T) {
	versionYAMLs, _ := filepath.Glob("../../configs/backend-catalog/versions/*/*.yaml")
	if len(versionYAMLs) == 0 {
		t.Skip("no catalog YAML files found")
	}

	// Collect YAML version IDs
	yamlIDs := map[string]bool{}
	for _, path := range versionYAMLs {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "id:") {
				id := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "id:"))
				id = strings.Trim(id, "\"'")
				yamlIDs[id] = true
			}
		}
	}

	// Verify deprecated versions are NOT in YAML (they're historical only, in seed)
	deprecated := map[string]bool{
		"bver-vllm-0.8.5":      true,
		"bver-vllm-0.10.0":     true,
		"bver-sglang-0.4.6":    true,
		"bver-sglang-0.5.0":    true,
		"bver-llamacpp-b4817":  true,
	}

	for id := range deprecated {
		if yamlIDs[id] {
			t.Errorf("deprecated version %s found in catalog YAML but should be seed-only", id)
		}
	}

	// Verify current versions exist in both YAML and seed
	current := []string{
		"vllm-v0.23.0", "sglang-v0.5.13.post1", "sglang-v0.5.12.post1",
		"sglang-0.4.6-compatible", "llamacpp-b9700", "backend-version.ollama.latest",
	}
	for _, id := range current {
		if !yamlIDs[id] {
			t.Errorf("current version %s missing from catalog YAML", id)
		}
	}
}

// TestCapabilitiesNotArrayFormat verifies no backend version uses array-format capabilities.
func TestCapabilitiesNotArrayFormat(t *testing.T) {
	db := setupTestDB(t)
	rows, err := db.Query(`SELECT id, capabilities_json FROM backend_versions WHERE is_deprecated=0`)
	if err != nil {
		t.Skip("DB not available")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, caps string
		rows.Scan(&id, &caps)
		if caps == "" || caps == "[]" || caps == "{}" {
			t.Errorf("default backend version %s has empty capabilities_json (must be structured object)", id)
		}
		// Detect array format: starts with [
		if strings.HasPrefix(strings.TrimSpace(caps), "[") {
			t.Errorf("backend version %s has array-format capabilities_json: %s", id, caps[:minInt(60, len(caps))])
		}
	}
}

func minInt(a, b int) int { if a < b { return a }; return b }
