package collector

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// GGUFMetadata holds extracted metadata from a GGUF file header.
type GGUFMetadata struct {
	Architecture    string `json:"architecture"`
	ContextLength   int64  `json:"context_length"`
	EmbeddingLength int64  `json:"embedding_length"`
	BlockCount      int64  `json:"block_count"`
	VocabSize       int64  `json:"vocab_size"`
	HeadCount       int64  `json:"head_count"`
	HeadCountKV     int64  `json:"head_count_kv"`
	Quantization    string `json:"quantization"`
	FileSizeBytes   int64  `json:"file_size_bytes"`
	Warnings        []string `json:"warnings,omitempty"`
}

const (
	ggufMagic   = 0x46554747 // "GGUF" in little-endian
	ggufTypeUint8    = 0
	ggufTypeInt8     = 1
	ggufTypeUint16   = 2
	ggufTypeInt16    = 3
	ggufTypeUint32   = 4
	ggufTypeInt32    = 5
	ggufTypeFloat32  = 6
	ggufTypeBool     = 7
	ggufTypeString   = 8
	ggufTypeArray    = 9
	ggufTypeUint64   = 10
	ggufTypeInt64    = 11
	ggufTypeFloat64  = 12
)

// readGGUFMeta reads GGUF metadata from a file. It reads only the header portion.
func readGGUFMeta(path string) (*GGUFMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	fileSize := fi.Size()

	meta := &GGUFMetadata{
		FileSizeBytes: fileSize,
	}

	// Read magic
	var magic uint32
	if err := binary.Read(f, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if magic != ggufMagic {
		return nil, fmt.Errorf("not a GGUF file: magic=%08x", magic)
	}

	// Read version
	var version uint32
	if err := binary.Read(f, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	_ = version // v2 or v3, both have same header structure

	// Read tensor count and metadata KV count
	var tensorCount, kvCount uint64
	if err := binary.Read(f, binary.LittleEndian, &tensorCount); err != nil {
		return nil, fmt.Errorf("read tensor_count: %w", err)
	}
	if err := binary.Read(f, binary.LittleEndian, &kvCount); err != nil {
		return nil, fmt.Errorf("read kv_count: %w", err)
	}

	// Read metadata KV pairs into a simple map
	kv := make(map[string]interface{})
	for i := uint64(0); i < kvCount; i++ {
		key, val, err := readGGUFKV(f)
		if err != nil {
			return nil, fmt.Errorf("read kv[%d]: %w", i, err)
		}
		kv[key] = val
	}

	// Extract known fields
	meta.Architecture = ggufString(kv, "general.architecture")
	if meta.Architecture == "" {
		meta.Architecture = "unknown"
		meta.Warnings = append(meta.Warnings, "architecture not found in GGUF metadata")
	}

	archPrefix := meta.Architecture
	meta.ContextLength = ggufInt(kv, archPrefix+".context_length")
	if meta.ContextLength == 0 {
		meta.Warnings = append(meta.Warnings, "context_length not found in GGUF metadata")
	}
	meta.EmbeddingLength = ggufInt(kv, archPrefix+".embedding_length")
	meta.BlockCount = ggufInt(kv, archPrefix+".block_count")
	meta.VocabSize = ggufInt(kv, archPrefix+".vocab_size")
	meta.HeadCount = ggufInt(kv, archPrefix+".attention.head_count")
	meta.HeadCountKV = ggufInt(kv, archPrefix+".attention.head_count_kv")
	if meta.HeadCountKV == 0 {
		meta.HeadCountKV = ggufInt(kv, archPrefix+".attention.head_count")
	}

	// Quantization: analyze tensor types
	meta.Quantization = detectGGUFQuantization(f, tensorCount)

	return meta, nil
}

// readGGUFKV reads one metadata key-value pair from the GGUF file.
func readGGUFKV(r io.Reader) (string, interface{}, error) {
	key, err := readGGUFString(r)
	if err != nil {
		return "", nil, fmt.Errorf("read key: %w", err)
	}

	var valType uint32
	if err := binary.Read(r, binary.LittleEndian, &valType); err != nil {
		return "", nil, fmt.Errorf("read value type: %w", err)
	}

	val, err := readGGUFValue(r, valType)
	if err != nil {
		return "", nil, fmt.Errorf("read value: %w", err)
	}
	return key, val, nil
}

// readGGUFValue reads a value of the given GGUF type.
func readGGUFValue(r io.Reader, valType uint32) (interface{}, error) {
	switch valType {
	case ggufTypeUint8:
		var v uint8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeInt8:
		var v int8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeUint16:
		var v uint16
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeInt16:
		var v int16
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeUint32:
		var v uint32
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeInt32:
		var v int32
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeFloat32:
		var v float32
		err := binary.Read(r, binary.LittleEndian, &v)
		return float64(v), err
	case ggufTypeBool:
		var v uint8
		err := binary.Read(r, binary.LittleEndian, &v)
		return v != 0, err
	case ggufTypeString:
		return readGGUFString(r)
	case ggufTypeUint64:
		var v uint64
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeInt64:
		var v int64
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeFloat64:
		var v float64
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case ggufTypeArray:
		return readGGUFArray(r)
	default:
		return nil, fmt.Errorf("unknown GGUF type: %d", valType)
	}
}

// readGGUFString reads a GGUF string (uint64 length + bytes).
func readGGUFString(r io.Reader) (string, error) {
	var length uint64
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	if length > 10*1024*1024 { // safety limit: 10MB
		return "", fmt.Errorf("string too long: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// readGGUFArray reads a GGUF array (type + count + values).
func readGGUFArray(r io.Reader) ([]interface{}, error) {
	var elemType uint32
	if err := binary.Read(r, binary.LittleEndian, &elemType); err != nil {
		return nil, err
	}
	var count uint64
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	if count > 1000000 { // safety limit
		return nil, fmt.Errorf("array too long: %d", count)
	}
	result := make([]interface{}, count)
	for i := uint64(0); i < count; i++ {
		val, err := readGGUFValue(r, elemType)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

// ggufString extracts a string value from the KV map.
func ggufString(kv map[string]interface{}, key string) string {
	v, ok := kv[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// ggufInt extracts an integer value from the KV map.
func ggufInt(kv map[string]interface{}, key string) int64 {
	v, ok := kv[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case uint8:
		return int64(n)
	case int8:
		return int64(n)
	case uint16:
		return int64(n)
	case int16:
		return int64(n)
	case uint32:
		return int64(n)
	case int32:
		return int64(n)
	case uint64:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// tensorTypeName returns a human-readable name for a GGUF tensor type number.
func tensorTypeName(t uint32) string {
	// GGML quantized type codes (common subset)
	types := map[uint32]string{
		0:  "f32",
		1:  "f16",
		2:  "q4_0",
		3:  "q4_1",
		6:  "q5_0",
		7:  "q5_1",
		8:  "q8_0",
		9:  "q8_1",
		10: "q2_k",
		11: "q3_k",
		12: "q4_k",
		13: "q5_k",
		14: "q6_k",
		15: "q8_k",
		16: "iq2_xxs",
		17: "iq2_xs",
		18: "iq3_xxs",
		19: "iq1_s",
		20: "iq4_nl",
		21: "iq3_s",
		22: "iq2_s",
		23: "iq4_xs",
		24: "i8",
		25: "i16",
		26: "i32",
		27: "i64",
		28: "f64",
		29: "iq1_m",
		30: "bf16",
	}
	if name, ok := types[t]; ok {
		return name
	}
	return fmt.Sprintf("type_%d", t)
}

// quantNameFromGGML maps GGML tensor type names to human-readable quantization names.
func quantNameFromGGML(t string) string {
	canonical := map[string]string{
		"f32": "F32", "f16": "F16", "bf16": "BF16",
		"q4_0": "Q4_0", "q4_1": "Q4_1",
		"q5_0": "Q5_0", "q5_1": "Q5_1",
		"q8_0": "Q8_0", "q8_1": "Q8_1",
		"q2_k": "Q2_K", "q3_k": "Q3_K",
		"q4_k": "Q4_K_M", "q5_k": "Q5_K_M",
		"q6_k": "Q6_K", "q8_k": "Q8_K",
		"iq1_s": "IQ1_S", "iq1_m": "IQ1_M",
		"iq2_xxs": "IQ2_XXS", "iq2_xs": "IQ2_XS", "iq2_s": "IQ2_S",
		"iq3_xxs": "IQ3_XXS", "iq3_s": "IQ3_S",
		"iq4_nl": "IQ4_NL", "iq4_xs": "IQ4_XS",
		"i8": "I8", "i16": "I16", "i32": "I32", "i64": "I64",
	}
	if name, ok := canonical[t]; ok {
		return name
	}
	return strings.ToUpper(t)
}

// detectGGUFQuantization analyzes tensor types to determine quantization.
// It counts the occurrences of each tensor type and returns the dominant quantized type.
func detectGGUFQuantization(r io.ReadSeeker, tensorCount uint64) string {
	if tensorCount == 0 {
		return guessQuantFromFilename("")
	}

	typeCount := make(map[string]int)
	var dominantType string
	dominantCount := 0

	for i := uint64(0); i < tensorCount; i++ {
		// Read tensor name
		name, err := readGGUFString(r)
		if err != nil {
			break
		}
		_ = name

		// Read n_dimensions
		var nDims uint32
		if err := binary.Read(r, binary.LittleEndian, &nDims); err != nil {
			break
		}

		// Skip dimensions
		dimBuf := make([]byte, nDims*8)
		if _, err := io.ReadFull(r, dimBuf); err != nil {
			break
		}

		// Read tensor type
		var ttype uint32
		if err := binary.Read(r, binary.LittleEndian, &ttype); err != nil {
			break
		}

		// Skip offset
		var offset uint64
		if err := binary.Read(r, binary.LittleEndian, &offset); err != nil {
			break
		}
		_ = offset

		tname := tensorTypeName(ttype)
		typeCount[tname]++
		if typeCount[tname] > dominantCount {
			dominantCount = typeCount[tname]
			dominantType = tname
		}
	}

	if dominantType == "" {
		return "unknown"
	}

	// Special handling: if the dominant type is f16/f32 and there's a q* type with
	// many tensors, the q* type might be the actual quantization (f16/f32 could be
	// output/embedding tensors). Use the most common quantized type if available.
	if dominantType == "f16" || dominantType == "f32" {
		for t, c := range typeCount {
			if strings.HasPrefix(t, "q") || strings.HasPrefix(t, "iq") {
				if c > typeCount[dominantType]/4 {
					return quantNameFromGGML(t)
				}
			}
		}
		return dominantType
	}

	return quantNameFromGGML(dominantType)
}

// guessQuantFromFilename tries to detect quantization from GGUF filename.
// Recognizes patterns like Q4_K_M, Q5_K_S, Q8_0, F16, etc.
func guessQuantFromFilename(filename string) string {
	lower := strings.ToLower(filename)

	// Ordered by specificity (longer patterns first)
	patterns := []string{
		"iq1_s", "iq1_m", "iq2_xxs", "iq2_xs", "iq2_s",
		"iq3_xxs", "iq3_s", "iq4_nl", "iq4_xs",
		"q2_k", "q3_k_s", "q3_k_m", "q3_k_l",
		"q4_k_s", "q4_k_m", "q4_0", "q4_1",
		"q5_k_s", "q5_k_m", "q5_0", "q5_1",
		"q6_k", "q8_k", "q8_0", "q8_1",
		"f16", "f32", "bf16",
	}

	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return strings.ToUpper(p)
		}
	}
	return "unknown"
}
