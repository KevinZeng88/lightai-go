package collector

import (
	"os"
	"testing"
)

func TestReadGGUFMeta_Qwen3_5_9B(t *testing.T) {
	path := "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("GGUF test file not found: %s", path)
	}

	meta, err := readGGUFMeta(path)
	if err != nil {
		t.Fatalf("readGGUFMeta failed: %v", err)
	}

	if meta.Architecture == "" || meta.Architecture == "unknown" {
		t.Errorf("expected architecture to be known, got: %s", meta.Architecture)
	}
	t.Logf("Architecture: %s", meta.Architecture)

	if meta.ContextLength == 0 {
		t.Errorf("expected context_length > 0, got: %d", meta.ContextLength)
	}
	t.Logf("ContextLength: %d", meta.ContextLength)

	if meta.Quantization == "" || meta.Quantization == "unknown" {
		t.Logf("WARNING: quantization could not be determined from tensor types")
	}
	t.Logf("Quantization: %s", meta.Quantization)

	t.Logf("BlockCount: %d", meta.BlockCount)
	t.Logf("VocabSize: %d", meta.VocabSize)
	t.Logf("EmbeddingLength: %d", meta.EmbeddingLength)
	t.Logf("HeadCount: %d", meta.HeadCount)
	t.Logf("FileSizeBytes: %d", meta.FileSizeBytes)
	t.Logf("Warnings: %v", meta.Warnings)
}

func TestReadGGUFMeta_Gemma(t *testing.T) {
	path := "/home/kzeng/models/gemma-4-31b-jang-crack-Q4_K_M.gguf"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("GGUF test file not found: %s", path)
	}

	meta, err := readGGUFMeta(path)
	if err != nil {
		t.Fatalf("readGGUFMeta failed: %v", err)
	}

	t.Logf("Architecture: %s", meta.Architecture)
	t.Logf("ContextLength: %d", meta.ContextLength)
	t.Logf("Quantization: %s", meta.Quantization)
	t.Logf("BlockCount: %d", meta.BlockCount)
	t.Logf("VocabSize: %d", meta.VocabSize)
	t.Logf("FileSizeBytes: %d", meta.FileSizeBytes)
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{5580000000, "5.2 GiB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestGuessQuantFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"Qwen3.5-9B-Q4_K_M.gguf", "Q4_K_M"},
		{"model-q5_k_s.gguf", "Q5_K_S"},
		{"llama-2-7b-Q8_0.gguf", "Q8_0"},
		{"model-f16.gguf", "F16"},
		{"unknown-model.gguf", "unknown"},
		{"gemma-4-31b-jang-crack-Q4_K_M.gguf", "Q4_K_M"},
	}
	for _, tt := range tests {
		got := guessQuantFromFilename(tt.filename)
		if got != tt.expected {
			t.Errorf("guessQuantFromFilename(%q) = %q, want %q", tt.filename, got, tt.expected)
		}
	}
}
