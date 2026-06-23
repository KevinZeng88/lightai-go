package runtimecontract

import "testing"

func TestIsValidFormat(t *testing.T) {
	valid := AllFormats()
	for _, f := range valid {
		if !IsValidFormat(f) {
			t.Errorf("IsValidFormat(%q) = false, want true", f)
		}
	}
	invalid := []string{"", "unknown_format", "pytorch", "GGUF", "HuggingFace"}
	for _, f := range invalid {
		if IsValidFormat(f) {
			t.Errorf("IsValidFormat(%q) = true, want false", f)
		}
	}
}

func TestIsValidTask(t *testing.T) {
	valid := AllTasks()
	for _, v := range valid {
		if !IsValidTask(v) {
			t.Errorf("IsValidTask(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "invalid_task", "CHAT", "Chat"}
	for _, v := range invalid {
		if IsValidTask(v) {
			t.Errorf("IsValidTask(%q) = true, want false", v)
		}
	}
}

func TestIsValidCapability(t *testing.T) {
	valid := AllCapabilities()
	for _, c := range valid {
		if !IsValidCapability(c) {
			t.Errorf("IsValidCapability(%q) = false, want true", c)
		}
	}
	invalid := []string{"", "unknown_cap", "CHAT"}
	for _, c := range invalid {
		if IsValidCapability(c) {
			t.Errorf("IsValidCapability(%q) = true, want false", c)
		}
	}
}

func TestIsValidPathMode(t *testing.T) {
	valid := AllPathModes()
	for _, p := range valid {
		if !IsValidPathMode(p) {
			t.Errorf("IsValidPathMode(%q) = false, want true", p)
		}
	}
	invalid := []string{"", "block", "DIRECTORY"}
	for _, p := range invalid {
		if IsValidPathMode(p) {
			t.Errorf("IsValidPathMode(%q) = true, want false", p)
		}
	}
}

func TestIsValidCapabilitySource(t *testing.T) {
	valid := AllCapabilitySources()
	for _, s := range valid {
		if !IsValidCapabilitySource(s) {
			t.Errorf("IsValidCapabilitySource(%q) = false, want true", s)
		}
	}
	invalid := []string{"", "manual", "SCAN"}
	for _, s := range invalid {
		if IsValidCapabilitySource(s) {
			t.Errorf("IsValidCapabilitySource(%q) = true, want false", s)
		}
	}
}

func TestIsValidTestMode(t *testing.T) {
	valid := AllTestModes()
	for _, m := range valid {
		if !IsValidTestMode(m) {
			t.Errorf("IsValidTestMode(%q) = false, want true", m)
		}
	}
	invalid := []string{"", "invalid_mode", "AUTO"}
	for _, m := range invalid {
		if IsValidTestMode(m) {
			t.Errorf("IsValidTestMode(%q) = true, want false", m)
		}
	}
}

func TestIsValidServingProtocol(t *testing.T) {
	valid := AllServingProtocols()
	for _, p := range valid {
		if !IsValidServingProtocol(p) {
			t.Errorf("IsValidServingProtocol(%q) = false, want true", p)
		}
	}
	invalid := []string{"", "grpc", "OPENAI-COMPATIBLE"}
	for _, p := range invalid {
		if IsValidServingProtocol(p) {
			t.Errorf("IsValidServingProtocol(%q) = true, want false", p)
		}
	}
}

func TestFormatConstantsUniqueness(t *testing.T) {
	seen := map[string]string{}
	for _, f := range AllFormats() {
		if other, ok := seen[f]; ok {
			t.Errorf("duplicate format value %q (constants: %s and %s)", f, other, f)
		}
		seen[f] = f
	}
}

func TestAllFormatsIncludesOllama(t *testing.T) {
	found := false
	for _, f := range AllFormats() {
		if f == FormatOllama {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllFormats() should include FormatOllama")
	}
}

func TestAllPathModesIncludesOllamaManaged(t *testing.T) {
	found := false
	for _, p := range AllPathModes() {
		if p == PathModeOllamaManaged {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllPathModes() should include PathModeOllamaManaged")
	}
}

func TestAllServingProtocolsIncludesOllama(t *testing.T) {
	found := false
	for _, p := range AllServingProtocols() {
		if p == ServingProtocolOllama {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllServingProtocols() should include ServingProtocolOllama")
	}
}
