package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigEditTemplatesListIncludesMaterializedRegistryData(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)

	w := httptest.NewRecorder()
	h.HandleListConfigEditTemplates(w, newReq("GET", "/x", "", adminSession(), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Templates []struct {
			TemplateID string `json:"template_id"`
			Source     string `json:"source"`
			AppliesTo  struct {
				Backend string   `json:"backend"`
				Vendors []string `json:"vendors"`
			} `json:"applies_to"`
			Fields []struct {
				Key     string `json:"key"`
				Section string `json:"section"`
				Tier    string `json:"tier"`
				View    string `json:"view"`
				Risk    string `json:"risk"`
			} `json:"fields"`
		} `json:"templates"`
		Issues []any `json:"issues"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Issues) > 0 {
		t.Fatalf("unexpected template registry issues: %s", w.Body.String())
	}

	backends := map[string]bool{}
	foundRuntimeArg := false
	foundDockerSecurity := false
	for _, tmpl := range response.Templates {
		if tmpl.Source != "catalog_materialized" {
			continue
		}
		backends[tmpl.AppliesTo.Backend] = true
		for _, field := range tmpl.Fields {
			if strings.HasPrefix(field.Key, "model_runtime.") {
				foundRuntimeArg = true
			}
			if field.Key == "docker.privileged" {
				if field.Risk != "high" || field.Tier != "expert" || field.Section != "security_high_risk" {
					t.Fatalf("privileged field not marked high-risk expert security: %+v", field)
				}
				foundDockerSecurity = true
			}
		}
	}
	for _, backend := range []string{"vllm", "sglang", "llamacpp"} {
		if !backends[backend] {
			t.Fatalf("materialized ConfigEdit template missing backend %s; backends=%v body=%s", backend, backends, w.Body.String())
		}
	}
	if !foundRuntimeArg {
		t.Fatalf("materialized templates do not include backend runtime args: %s", w.Body.String())
	}
	if !foundDockerSecurity {
		t.Fatalf("materialized templates do not include high-risk Docker security fields: %s", w.Body.String())
	}
}
