package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCatalogSeedDrift verifies that ConfigSet catalog loading has entries for
// all current YAML catalog versions.
func TestCatalogSeedDrift(t *testing.T) {
	versionYAMLs, _ := filepath.Glob("../../configs/backend-catalog/versions/*/*.yaml")
	if len(versionYAMLs) == 0 {
		t.Skip("no catalog YAML files found")
	}

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

// TestCapabilitiesNotArrayFormat verifies backend capabilities are structured
// ConfigSet objects rather than old array-style fields.
func TestCapabilitiesNotArrayFormat(t *testing.T) {
	db := setupTestDB(t)
	rows, err := db.Query(`SELECT id, config_set_json FROM backend_versions WHERE is_deprecated=0`)
	if err != nil {
		t.Skip("DB not available")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, configSetRaw string
		rows.Scan(&id, &configSetRaw)
		caps := configObject(parseConfigSet(configSetRaw), "backend.capabilities")
		if len(caps) == 0 {
			t.Errorf("backend version %s has empty ConfigSet backend.capabilities", id)
		}
	}
}
