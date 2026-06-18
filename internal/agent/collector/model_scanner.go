package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanModelPath scans a directory or file for model metadata.
// Detects HuggingFace (config.json), safetensors, GGUF formats.
func ScanModelPath(root, relPath string) map[string]interface{} {
	absPath := filepath.Join(root, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("cannot access path: %v", err),
		}
	}

	result := map[string]interface{}{
		"absolute_path": absPath,
		"model_root":    root,
		"relative_path": relPath,
	}

	if info.IsDir() {
		result["path_type"] = "directory"
		// Check for HuggingFace config
		if data, err := os.ReadFile(filepath.Join(absPath, "config.json")); err == nil {
			var config map[string]interface{}
			if json.Unmarshal(data, &config) == nil {
				result["format"] = "huggingface"
				if arch, ok := config["architectures"]; ok {
					result["architecture"] = arch
				}
				if mt, ok := config["model_type"]; ok {
					result["model_type"] = mt
				}
				result["has_config"] = true
			}
		}
		// Check for safetensors
		safePattern := filepath.Join(absPath, "*.safetensors")
		if matches, _ := filepath.Glob(safePattern); len(matches) > 0 {
			result["has_safetensors"] = true
			result["safetensors_count"] = len(matches)
			var totalSize int64
			for _, m := range matches {
				if fi, err := os.Stat(m); err == nil {
					totalSize += fi.Size()
				}
			}
			result["estimated_size_bytes"] = totalSize
			result["size_label"] = FormatBytes(totalSize)
		}
		// Check for GGUF files
		ggufPattern := filepath.Join(absPath, "*.gguf")
		if matches, _ := filepath.Glob(ggufPattern); len(matches) > 0 {
			result["has_gguf"] = true
			result["format"] = "gguf"
			result["gguf_files"] = len(matches)
		}
		// Check for tokenizer
		if _, err := os.Stat(filepath.Join(absPath, "tokenizer_config.json")); err == nil {
			result["has_tokenizer"] = true
		}
		// Check for generation config
		if _, err := os.Stat(filepath.Join(absPath, "generation_config.json")); err == nil {
			result["has_generation_config"] = true
		}
	} else {
		result["path_type"] = "file"
		result["size_bytes"] = info.Size()
		result["size_label"] = FormatBytes(info.Size())
		if strings.HasSuffix(strings.ToLower(relPath), ".gguf") {
			result["format"] = "gguf"
		}
	}

	result["discovered_name"] = filepath.Base(relPath)
	result["status"] = "ok"
	return result
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
