package runplan

// ProcessStartProfile defines a candidate startup skeleton for a backend family.
// Profiles are matched by backend_family and image Entrypoint/Cmd characteristics,
// NOT by official image repository name. This ensures vendor/private/custom images work.
type ProcessStartProfile struct {
	ID             string   `json:"id"`
	BackendFamily  string   `json:"backend_family"`
	EntrypointMode string   `json:"entrypoint_mode"` // "image_default" | "custom"
	Entrypoint     []string `json:"entrypoint,omitempty"`
	CommandPrefix  []string `json:"command_prefix,omitempty"`
	Priority       int      `json:"priority"`
	Description    string   `json:"description,omitempty"`
	// DetectionHints guides profile scoring based on image characteristics.
	DetectionHints *ProcessStartDetectionHints `json:"detection_hints,omitempty"`
	Warnings       []string                    `json:"warnings,omitempty"`
}

// ProcessStartDetectionHints guides profile-to-image matching.
type ProcessStartDetectionHints struct {
	// EntrypointKinds that are compatible with this profile.
	EntrypointKinds []string `json:"entrypoint_kinds,omitempty"`
	// AvoidIfEntrypointAlreadyStartsBackend: if true, deprioritize this profile
	// when the image ENTRYPOINT already appears to start the backend server.
	AvoidIfEntrypointAlreadyStartsBackend bool `json:"avoid_if_entrypoint_already_starts_backend,omitempty"`
}

// ProcessStartConfig is the user-accepted authoritative Layer 3 configuration.
// It is stored in NBR.config_snapshot_json.process_start_config and
// frozen into Deployment.config_snapshot_json.
type ProcessStartConfig struct {
	EntrypointMode string   `json:"entrypoint_mode"`          // "image_default" | "custom"
	Entrypoint     []string `json:"entrypoint,omitempty"`     // only when mode=custom
	CommandPrefix  []string `json:"command_prefix,omitempty"` // prepended to Cmd
	ShellMode      bool     `json:"shell_mode,omitempty"`     // v1 default false
	ProfileID      string   `json:"profile_id,omitempty"`     // profile that was accepted
	Source         string   `json:"source,omitempty"`         // "user_accepted_detection" | "user_override"
	Confidence     string   `json:"confidence,omitempty"`     // "high" | "medium" | "low"
	Warnings       []string `json:"warnings,omitempty"`
}

// ProcessStartDetection is the system-generated suggestion for how to start
// a container process for a given backend_family + image combination.
type ProcessStartDetection struct {
	Status           string               `json:"status"` // "candidate_found" | "no_profiles" | "image_not_inspected"
	SelectedProfile  string               `json:"selected_profile_id,omitempty"`
	EntrypointMode   string               `json:"entrypoint_mode,omitempty"`
	Entrypoint       []string             `json:"entrypoint,omitempty"`
	CommandPrefix    []string             `json:"command_prefix,omitempty"`
	ShellMode        bool                 `json:"shell_mode,omitempty"`
	Confidence       string               `json:"confidence,omitempty"` // "high" | "medium" | "low"
	Source           string               `json:"source,omitempty"`     // "backend_profile+image_inspect"
	CandidateProfile *DetectionCandidate  `json:"candidate_profile,omitempty"`
	AllCandidates    []DetectionCandidate `json:"candidate_profiles,omitempty"`
	Evidence         *DetectionEvidence   `json:"evidence,omitempty"`
	Warnings         []string             `json:"warnings,omitempty"`
}

// DetectionCandidate is a scored profile candidate.
type DetectionCandidate struct {
	ProfileID  string   `json:"profile_id"`
	Score      int      `json:"score"`
	Confidence string   `json:"confidence"`
	Reasons    []string `json:"reasons,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// DetectionEvidence records the inputs and signals used for detection.
type DetectionEvidence struct {
	BackendFamily   string   `json:"backend_family,omitempty"`
	ImageRef        string   `json:"image_ref,omitempty"`
	ImageEntrypoint []string `json:"image_entrypoint,omitempty"`
	ImageCmd        []string `json:"image_cmd,omitempty"`
	MatchedSignals  []string `json:"matched_signals,omitempty"`
}

// DefaultProcessStartProfiles returns the v1 default profiles keyed by backend_family.
// v1 storage: Go constants. Future: YAML catalog or DB column.
func DefaultProcessStartProfiles() map[string][]ProcessStartProfile {
	return map[string][]ProcessStartProfile{
		"vllm": {
			{
				ID:             "vllm.image_default",
				BackendFamily:  "vllm",
				EntrypointMode: "image_default",
				CommandPrefix:  nil,
				Priority:       100,
				Description:    "Preserve image ENTRYPOINT, pass model args as CMD.",
				DetectionHints: &ProcessStartDetectionHints{
					EntrypointKinds: []string{"server_binary", "empty", "unknown"},
				},
			},
		},
		"sglang": {
			{
				ID:             "sglang.python_module_launcher",
				BackendFamily:  "sglang",
				EntrypointMode: "image_default",
				CommandPrefix:  []string{"python3", "-m", "sglang.launch_server"},
				Priority:       100,
				Description:    "Run SGLang through python module launcher as Docker CMD, preserving image ENTRYPOINT.",
				DetectionHints: &ProcessStartDetectionHints{
					EntrypointKinds:                       []string{"wrapper_script", "empty", "unknown"},
					AvoidIfEntrypointAlreadyStartsBackend: true,
				},
			},
			{
				ID:             "sglang.custom_entrypoint",
				BackendFamily:  "sglang",
				EntrypointMode: "custom",
				Entrypoint:     []string{"python3", "-m", "sglang.launch_server"},
				Priority:       40,
				Description:    "Set ENTRYPOINT explicitly to python3 module launcher.",
				Warnings:       []string{"May bypass image ENTRYPOINT wrapper."},
				DetectionHints: &ProcessStartDetectionHints{
					EntrypointKinds: []string{"server_binary", "empty", "unknown"},
				},
			},
		},
		"llamacpp": {
			{
				ID:             "llamacpp.image_default",
				BackendFamily:  "llamacpp",
				EntrypointMode: "image_default",
				CommandPrefix:  nil,
				Priority:       100,
				Description:    "Preserve image ENTRYPOINT, pass model args as CMD.",
				DetectionHints: &ProcessStartDetectionHints{
					EntrypointKinds: []string{"server_binary", "empty", "unknown"},
				},
			},
		},
	}
}
