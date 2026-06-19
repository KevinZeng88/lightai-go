package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanCandidate represents one model candidate found during scanning.
type ScanCandidate struct {
	Path             string                 `json:"path"`
	PathType         string                 `json:"path_type"` // "file" or "directory"
	Format           string                 `json:"format"`
	DetectedMetadata map[string]interface{} `json:"detected_metadata"`
	Warnings         []string               `json:"warnings"`
	AutoSelected     bool                   `json:"auto_selected"`
	SelectionReason  string                 `json:"selection_reason"`
	SizeBytes        int64                  `json:"size_bytes"`
	SizeLabel        string                 `json:"size_label"`
}

// ScanResult is the structured result of a model path scan.
type ScanResult struct {
	ScanRoot   string          `json:"scan_root"`
	Candidates []ScanCandidate `json:"candidates"`
	Warnings   []string        `json:"warnings"`
	Error      string          `json:"error,omitempty"`
}

// ScanModelPath scans a directory or file for model candidates.
// For directories: detects HF config, GGUF files, or mixed.
// For files: detects GGUF by extension.
func ScanModelPath(root, relPath string) map[string]interface{} {
	absPath := filepath.Join(root, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("cannot access path: %v", err),
		}
	}

	if info.IsDir() {
		return scanDirectory(absPath, root, relPath)
	}

	// Single file scan
	return scanSingleFile(absPath, root, relPath, info)
}

// scanDirectory scans a directory for model candidates.
func scanDirectory(absPath, root, relPath string) map[string]interface{} {
	result := &ScanResult{
		ScanRoot: absPath,
	}

	hasHF := false
	hasGGUF := false

	// ── HF Directory Detection ──
	configPath := filepath.Join(absPath, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var config map[string]interface{}
		if json.Unmarshal(data, &config) == nil {
			hasHF = true
			candidate := ScanCandidate{
				Path:            absPath,
				PathType:        "directory",
				Format:          "huggingface",
				DetectedMetadata: make(map[string]interface{}),
				Warnings:        []string{},
				AutoSelected:    false,
				SelectionReason: "",
			}

			// Extract HF metadata
			extractHFMetadata(absPath, config, &candidate)

			result.Candidates = append(result.Candidates, candidate)
		}
	}

	// ── GGUF File Detection ──
	if matches, _ := filepath.Glob(filepath.Join(absPath, "*.gguf")); len(matches) > 0 {
		hasGGUF = true
		for _, ggufPath := range matches {
			candidate := ScanCandidate{
				Path:            ggufPath,
				PathType:        "file",
				Format:          "gguf",
				DetectedMetadata: make(map[string]interface{}),
				Warnings:        []string{},
				AutoSelected:    false,
				SelectionReason: "",
			}
			if fi, err := os.Stat(ggufPath); err == nil {
				candidate.SizeBytes = fi.Size()
				candidate.SizeLabel = FormatBytes(fi.Size())
				candidate.DetectedMetadata["file_size_bytes"] = float64(fi.Size())
			}
			candidate.DetectedMetadata["format"] = "gguf"
			extractGGUFMetadata(ggufPath, &candidate)
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	_ = hasHF
	_ = hasGGUF

	// ── Auto-selection logic ──
	totalCandidates := len(result.Candidates)
	if totalCandidates == 1 {
		result.Candidates[0].AutoSelected = true
		result.Candidates[0].SelectionReason = fmt.Sprintf("single %s candidate in directory", result.Candidates[0].Format)
	} else if totalCandidates > 1 {
		// Check if all are same type
		allGGUF := true
		for _, c := range result.Candidates {
			if c.Format != "gguf" {
				allGGUF = false
				break
			}
		}
		if allGGUF {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("found %d GGUF files, user must select one", totalCandidates))
		} else {
			result.Warnings = append(result.Warnings,
				"mixed model types found (HF + GGUF), user must select one")
		}
	}

	if totalCandidates == 0 {
		result.Error = "no recognizable model files found in directory"
	}

	// ── Convert to map for API compatibility ──
	return toMap(result)
}

// scanSingleFile scans a single file for model metadata.
func scanSingleFile(absPath, root, relPath string, info os.FileInfo) map[string]interface{} {
	result := &ScanResult{
		ScanRoot: absPath,
	}

	candidate := ScanCandidate{
		Path:            absPath,
		PathType:        "file",
		DetectedMetadata: make(map[string]interface{}),
		Warnings:        []string{},
		AutoSelected:    true,
		SelectionReason: "single file selected",
	}

	candidate.SizeBytes = info.Size()
	candidate.SizeLabel = FormatBytes(info.Size())
	candidate.DetectedMetadata["file_size_bytes"] = float64(info.Size())

	if strings.HasSuffix(strings.ToLower(relPath), ".gguf") {
		candidate.Format = "gguf"
		candidate.DetectedMetadata["format"] = "gguf"
		extractGGUFMetadata(absPath, &candidate)
	} else {
		candidate.Format = "custom"
		candidate.DetectedMetadata["format"] = "custom"
	}

	result.Candidates = append(result.Candidates, candidate)

	return toMap(result)
}

// extractGGUFMetadata reads GGUF header metadata and populates the candidate.
func extractGGUFMetadata(path string, candidate *ScanCandidate) {
	meta, err := readGGUFMeta(path)
	if err != nil {
		candidate.Warnings = append(candidate.Warnings,
			fmt.Sprintf("GGUF metadata read failed: %v", err))
		// Fall back to filename-based quantization guess
		if candidate.DetectedMetadata["quantization"] == nil {
			candidate.DetectedMetadata["quantization"] = guessQuantFromFilename(path)
		}
		return
	}

	candidate.DetectedMetadata["architecture"] = meta.Architecture
	if meta.ContextLength > 0 {
		candidate.DetectedMetadata["context_length"] = float64(meta.ContextLength)
	}
	if meta.EmbeddingLength > 0 {
		candidate.DetectedMetadata["embedding_length"] = float64(meta.EmbeddingLength)
	}
	if meta.BlockCount > 0 {
		candidate.DetectedMetadata["block_count"] = float64(meta.BlockCount)
	}
	if meta.VocabSize > 0 {
		candidate.DetectedMetadata["vocab_size"] = float64(meta.VocabSize)
	}
	if meta.HeadCount > 0 {
		candidate.DetectedMetadata["head_count"] = float64(meta.HeadCount)
	}
	if meta.HeadCountKV > 0 {
		candidate.DetectedMetadata["head_count_kv"] = float64(meta.HeadCountKV)
	}
	candidate.DetectedMetadata["file_size_bytes"] = float64(meta.FileSizeBytes)

	// Quantization
	q := meta.Quantization
	if q == "" || q == "unknown" {
		q = guessQuantFromFilename(path)
	}
	if q != "" && q != "unknown" {
		candidate.DetectedMetadata["quantization"] = q
	}

	candidate.Warnings = append(candidate.Warnings, meta.Warnings...)
}

// extractHFMetadata reads HuggingFace config files and populates metadata.
func extractHFMetadata(absPath string, config map[string]interface{}, candidate *ScanCandidate) {
	candidate.DetectedMetadata["format"] = "huggingface"

	// config.json fields
	if arch, ok := config["architectures"]; ok {
		candidate.DetectedMetadata["architectures"] = arch
		if archList, ok := arch.([]interface{}); ok && len(archList) > 0 {
			candidate.DetectedMetadata["architecture"] = archList[0]
		}
	}
	if mt, ok := config["model_type"]; ok {
		candidate.DetectedMetadata["model_type"] = mt
	}
	if dt, ok := config["torch_dtype"]; ok {
		candidate.DetectedMetadata["torch_dtype"] = dt
	}
	if mpe, ok := config["max_position_embeddings"]; ok {
		candidate.DetectedMetadata["max_position_embeddings"] = mpe
		switch v := mpe.(type) {
		case float64:
			candidate.DetectedMetadata["context_length"] = v
		}
	}
	if hs, ok := config["hidden_size"]; ok {
		candidate.DetectedMetadata["hidden_size"] = hs
	}
	if nhl, ok := config["num_hidden_layers"]; ok {
		candidate.DetectedMetadata["num_hidden_layers"] = nhl
	}
	if nah, ok := config["num_attention_heads"]; ok {
		candidate.DetectedMetadata["num_attention_heads"] = nah
	}
	if nkvh, ok := config["num_key_value_heads"]; ok {
		candidate.DetectedMetadata["num_key_value_heads"] = nkvh
	}
	if vs, ok := config["vocab_size"]; ok {
		candidate.DetectedMetadata["vocab_size"] = vs
	}
	if rs, ok := config["rope_scaling"]; ok {
		candidate.DetectedMetadata["rope_scaling"] = rs
	}
	if qc, ok := config["quantization_config"]; ok {
		candidate.DetectedMetadata["quantization_config"] = qc
	}

	// Safetensors: compute total file size
	safePattern := filepath.Join(absPath, "*.safetensors")
	if matches, _ := filepath.Glob(safePattern); len(matches) > 0 {
		var totalSize int64
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil {
				totalSize += fi.Size()
			}
		}
		candidate.DetectedMetadata["safetensors_count"] = float64(len(matches))
		candidate.SizeBytes = totalSize
		candidate.SizeLabel = FormatBytes(totalSize)
		candidate.DetectedMetadata["file_size_bytes"] = float64(totalSize)
	} else {
		// Compute total directory size
		var totalSize int64
		filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
		candidate.SizeBytes = totalSize
		candidate.SizeLabel = FormatBytes(totalSize)
		candidate.DetectedMetadata["file_size_bytes"] = float64(totalSize)
	}

	// Check for tokenizer
	if _, err := os.Stat(filepath.Join(absPath, "tokenizer_config.json")); err == nil {
		candidate.DetectedMetadata["has_tokenizer"] = true
	}

	// Check for generation_config.json
	if data, err := os.ReadFile(filepath.Join(absPath, "generation_config.json")); err == nil {
		var genConfig map[string]interface{}
		if json.Unmarshal(data, &genConfig) == nil {
			if ml, ok := genConfig["max_length"]; ok {
				candidate.DetectedMetadata["generation_max_length"] = ml
			}
		}
	}

	// Mark missing fields with warnings
	if candidate.DetectedMetadata["max_position_embeddings"] == nil {
		candidate.Warnings = append(candidate.Warnings, "max_position_embeddings not found in config.json")
	}
	if candidate.DetectedMetadata["architectures"] == nil {
		candidate.Warnings = append(candidate.Warnings, "architectures not found in config.json")
	}
}

// FormatBytes converts a byte count to a human-readable string.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// toMap converts a ScanResult to a map[string]interface{} for API compatibility.
func toMap(sr *ScanResult) map[string]interface{} {
	if sr.Error != "" {
		return map[string]interface{}{
			"scan_root":  sr.ScanRoot,
			"error":      sr.Error,
			"candidates": []interface{}{},
			"warnings":   sr.Warnings,
		}
	}
	return map[string]interface{}{
		"scan_root":  sr.ScanRoot,
		"candidates": sr.Candidates,
		"warnings":   sr.Warnings,
	}
}
