package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorIncludesStableCode(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusConflict, "display_name already exists in runtime templates")

	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d want %d", w.Code, http.StatusConflict)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body["code"] != "display_name_exists" {
		t.Fatalf("code=%q want display_name_exists body=%#v", body["code"], body)
	}
	if body["message"] == "" || body["error"] == "" {
		t.Fatalf("debug message fields missing: %#v", body)
	}
}
