package runplan

import "strings"

// ClassifyEntrypointShape classifies a Docker image ENTRYPOINT into a shape category
// used for profile matching. The classification is based on structural heuristics:
//
//	"empty"          — no ENTRYPOINT set
//	"server_binary"  — single or multi-token binary path (e.g., ["vllm","serve"], ["/app/llama-server"])
//	"wrapper_script" — shell script path (e.g., ["/opt/nvidia/nvidia_entrypoint.sh"])
//	"python_launcher"— starts with python/python3
//	"unknown"        — fallback
func ClassifyEntrypointShape(entrypoint []string) string {
	if len(entrypoint) == 0 {
		return "empty"
	}

	first := entrypoint[0]

	// Python launcher detection.
	if first == "python" || first == "python3" {
		return "python_launcher"
	}

	// Wrapper script detection: shell scripts, entrypoint wrappers.
	if strings.HasSuffix(first, ".sh") ||
		strings.Contains(first, "entrypoint") ||
		strings.Contains(first, "nvidia") {
		return "wrapper_script"
	}

	// Server binary: known binary paths or multi-token commands.
	if strings.HasPrefix(first, "/") ||
		strings.Contains(first, "llama") ||
		strings.Contains(first, "vllm") ||
		strings.Contains(first, "sglang") {
		return "server_binary"
	}

	// Binary name without path (e.g., "vllm", "ollama").
	if !strings.Contains(first, "/") && !strings.Contains(first, ".") {
		return "server_binary"
	}

	return "unknown"
}

// DetectProcessStart evaluates candidate profiles against image inspect evidence
// and returns the best-match detection result.
//
// Parameters:
//   - backendFamily: derived from inference_backends.name (e.g., "vllm", "sglang")
//   - imageRef: the Docker image reference (for evidence only, not matching)
//   - imageEntrypoint: Config.Entrypoint from docker image inspect
//   - imageCmd: Config.Cmd from docker image inspect
func DetectProcessStart(backendFamily, imageRef string, imageEntrypoint, imageCmd []string) *ProcessStartDetection {
	detection := &ProcessStartDetection{
		Source: "backend_profile+image_inspect",
		Evidence: &DetectionEvidence{
			BackendFamily:   backendFamily,
			ImageRef:        imageRef,
			ImageEntrypoint: imageEntrypoint,
			ImageCmd:        imageCmd,
		},
	}

	// Get profiles for this backend family.
	allProfiles := DefaultProcessStartProfiles()
	profiles, ok := allProfiles[backendFamily]
	if !ok || len(profiles) == 0 {
		detection.Status = "no_profiles"
		detection.Confidence = "low"
		detection.Warnings = append(detection.Warnings,
			"no process start profiles defined for backend_family="+backendFamily)
		return detection
	}

	// Classify the image entrypoint shape.
	entrypointShape := ClassifyEntrypointShape(imageEntrypoint)
	detection.Evidence.MatchedSignals = append(detection.Evidence.MatchedSignals,
		"backend_family", "entrypoint_shape:"+entrypointShape)

	// Score each candidate profile.
	var candidates []DetectionCandidate
	for _, p := range profiles {
		c := scoreProfile(p, entrypointShape, imageEntrypoint)
		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		detection.Status = "no_profiles"
		detection.Confidence = "low"
		return detection
	}

	// Select the highest-scoring candidate.
	best := candidates[0]
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Score > best.Score {
			best = candidates[i]
		}
	}

	// Find the matching profile.
	var selectedProfile *ProcessStartProfile
	for i := range profiles {
		if profiles[i].ID == best.ProfileID {
			selectedProfile = &profiles[i]
			break
		}
	}

	detection.Status = "candidate_found"
	detection.SelectedProfile = best.ProfileID
	detection.Confidence = best.Confidence
	detection.Warnings = append(detection.Warnings, best.Warnings...)
	detection.CandidateProfile = &best
	detection.AllCandidates = candidates

	if selectedProfile != nil {
		detection.EntrypointMode = selectedProfile.EntrypointMode
		detection.Entrypoint = selectedProfile.Entrypoint
		detection.CommandPrefix = selectedProfile.CommandPrefix
		if len(selectedProfile.Warnings) > 0 {
			detection.Warnings = append(detection.Warnings, selectedProfile.Warnings...)
		}
	}

	return detection
}

// scoreProfile evaluates a single profile against image evidence.
func scoreProfile(p ProcessStartProfile, entrypointShape string, imageEntrypoint []string) DetectionCandidate {
	c := DetectionCandidate{
		ProfileID: p.ID,
		Reasons:   []string{"backend_family=" + p.BackendFamily},
	}

	// Base score from profile priority.
	score := p.Priority

	// Entrypoint shape compatibility.
	hints := p.DetectionHints
	if hints != nil {
		shapeOK := false
		for _, k := range hints.EntrypointKinds {
			if k == entrypointShape {
				shapeOK = true
				break
			}
		}
		if shapeOK {
			score += 20
			c.Reasons = append(c.Reasons, "entrypoint_shape="+entrypointShape+" matches profile hints")
		} else if entrypointShape == "unknown" {
			score -= 10
			c.Reasons = append(c.Reasons, "entrypoint_shape=unknown reduces confidence")
			c.Warnings = append(c.Warnings, "image entrypoint shape is unknown; detection may be inaccurate")
		} else {
			score -= 20
			c.Reasons = append(c.Reasons, "entrypoint_shape="+entrypointShape+" not in profile hints")
		}

		// AvoidIfEntrypointAlreadyStartsBackend check.
		if hints.AvoidIfEntrypointAlreadyStartsBackend && entrypointShape == "server_binary" {
			// Check if the entrypoint already contains backend launcher tokens.
			for _, token := range imageEntrypoint {
				lower := strings.ToLower(token)
				if strings.Contains(lower, "sglang") || strings.Contains(lower, "vllm") ||
					strings.Contains(lower, "llama") || strings.Contains(lower, "server") {
					score -= 30
					c.Warnings = append(c.Warnings,
						"image entrypoint may already start the backend; command_prefix could cause double-launch")
					break
				}
			}
		}
	}

	// Confidence from score.
	switch {
	case score >= 100:
		c.Confidence = "high"
	case score >= 70:
		c.Confidence = "medium"
	default:
		c.Confidence = "low"
	}

	c.Score = score
	return c
}

// DeriveBackendFamily extracts the backend family name from the backend_id convention.
// Target catalog uses "backend.<name>" (e.g., "backend.vllm").
// Legacy seed uses "backend-<name>" (e.g., "backend-vllm").
// Returns the original id if neither pattern matches.
func DeriveBackendFamily(backendID string) string {
	if strings.HasPrefix(backendID, "backend.") {
		return strings.TrimPrefix(backendID, "backend.")
	}
	if strings.HasPrefix(backendID, "backend-") {
		return strings.TrimPrefix(backendID, "backend-")
	}
	return backendID
}
