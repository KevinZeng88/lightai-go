package runplan

import (
	"os"
	"testing"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/runtime-logs/" + name)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return string(data)
}

func TestClassifySGLangTorchaoSyntaxWarning(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := loadFixture(t, "sglang-torchao-syntax-warning.log")
	events := c.ClassifyLogText(log)
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	found := false
	for _, ev := range events {
		if ev.RuleID == "sglang.torchao.syntax_warning" {
			found = true
			if ev.Severity != LogSeverityNoise {
				t.Errorf("expected noise severity, got %s", ev.Severity)
			}
			if ev.Category != LogCategoryDependencyWarning {
				t.Errorf("expected dependency_warning category, got %s", ev.Category)
			}
		}
	}
	if !found {
		t.Error("expected sglang.torchao.syntax_warning rule to match")
	}
}

func TestClassifySGLangAttentionBackendDefault(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := loadFixture(t, "sglang-attention-backend-default.log")
	events := c.ClassifyLogText(log)
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	found := false
	for _, ev := range events {
		if ev.RuleID == "sglang.attention_backend.default" {
			found = true
			if ev.Severity != LogSeverityAdvisory {
				t.Errorf("expected advisory severity, got %s", ev.Severity)
			}
			if ev.Category != LogCategoryDefaultSelection {
				t.Errorf("expected default_selection category, got %s", ev.Category)
			}
		}
	}
	if !found {
		t.Error("expected sglang.attention_backend.default rule to match")
	}
}

func TestClassifyLlamaCppEnvHostOverwritten(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := loadFixture(t, "llamacpp-env-host-overwritten.log")
	events := c.ClassifyLogText(log)
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	found := false
	for _, ev := range events {
		if ev.RuleID == "llamacpp.env_overwritten.host" {
			found = true
			if ev.Severity != LogSeverityWarning {
				t.Errorf("expected warning severity, got %s", ev.Severity)
			}
			if ev.Category != LogCategoryArgConflict {
				t.Errorf("expected arg_conflict category, got %s", ev.Category)
			}
		}
	}
	if !found {
		t.Error("expected llamacpp.env_overwritten.host rule to match")
	}
}

func TestClassifyCUDAOOM(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := loadFixture(t, "cuda-oom.log")
	events := c.ClassifyLogText(log)
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	found := false
	for _, ev := range events {
		if ev.RuleID == "cuda.oom" {
			found = true
			if ev.Severity != LogSeverityError {
				t.Errorf("expected error severity, got %s", ev.Severity)
			}
			if ev.Category != LogCategoryOOM {
				t.Errorf("expected oom category, got %s", ev.Category)
			}
		}
	}
	if !found {
		t.Error("expected cuda.oom rule to match")
	}
}

func TestClassifyMultipleEvents(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := `warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host
RuntimeError: CUDA out of memory. Tried to allocate 2.00 GiB.
Some other line`
	events := c.ClassifyLogText(log)
	ruleIDs := make(map[string]bool)
	for _, ev := range events {
		ruleIDs[ev.RuleID] = true
	}
	if !ruleIDs["llamacpp.env_overwritten.host"] {
		t.Error("expected llamacpp.env_overwritten.host")
	}
	if !ruleIDs["cuda.oom"] {
		t.Error("expected cuda.oom")
	}
}

func TestClassifyNoMatch(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := "INFO: Server started on port 8080\nINFO: Ready to accept connections"
	events := c.ClassifyLogText(log)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestIsNonFatal(t *testing.T) {
	events := []RuntimeLogEvent{
		{Severity: LogSeverityNoise},
		{Severity: LogSeverityAdvisory},
	}
	if !IsNonFatal(events) {
		t.Error("noise + advisory should be non-fatal")
	}

	events = append(events, RuntimeLogEvent{Severity: LogSeverityWarning})
	if IsNonFatal(events) {
		t.Error("warning should NOT be non-fatal")
	}
}

func TestFilterBySeverity(t *testing.T) {
	events := []RuntimeLogEvent{
		{Severity: LogSeverityNoise},
		{Severity: LogSeverityAdvisory},
		{Severity: LogSeverityWarning},
		{Severity: LogSeverityError},
	}
	filtered := FilterBySeverity(events, LogSeverityWarning)
	if len(filtered) != 2 {
		t.Errorf("expected 2 events (warning + error), got %d", len(filtered))
	}
}

func TestOccurrences(t *testing.T) {
	c := NewRuntimeLogClassifier()
	log := `warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host
warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host
warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host`
	events := c.ClassifyLogText(log)
	found := false
	for _, ev := range events {
		if ev.RuleID == "llamacpp.env_overwritten.host" {
			found = true
			if ev.Occurrences != 3 {
				t.Errorf("expected 3 occurrences, got %d", ev.Occurrences)
			}
		}
	}
	if !found {
		t.Error("expected llamacpp.env_overwritten.host rule to match")
	}
}

func TestFormatEventsForDisplay(t *testing.T) {
	events := []RuntimeLogEvent{
		{
			RuleID:      "test.rule",
			Severity:    LogSeverityWarning,
			Message:     "Test message",
			Suggestion:  "Fix it",
			Occurrences: 2,
		},
	}
	display := FormatEventsForDisplay(events)
	if display == "" {
		t.Error("expected non-empty display")
	}
}
