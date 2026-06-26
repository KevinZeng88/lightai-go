package runplan

import (
	"fmt"
	"strings"
)

// LintSeverity represents the severity of a lint finding.
type LintSeverity string

const (
	LintSeverityError    LintSeverity = "error"
	LintSeverityWarning  LintSeverity = "warning"
	LintSeverityAdvisory LintSeverity = "advisory"
)

// LintCategory represents the category of a lint finding.
type LintCategory string

const (
	LintCategoryDuplicateArg       LintCategory = "duplicate_arg"
	LintCategoryEnvCLIConflict     LintCategory = "env_cli_conflict"
	LintCategoryPlatformOverridden LintCategory = "platform_overridden"
	LintCategoryHighRisk           LintCategory = "high_risk"
	LintCategoryUnsupported        LintCategory = "unsupported"
	LintCategoryDisabledApplied    LintCategory = "disabled_applied"
	LintCategoryMissingRequired    LintCategory = "missing_required"
	LintCategoryVendorIncompatible LintCategory = "vendor_incompatible"
)

// LintFinding is a single lint finding.
type LintFinding struct {
	ID         string       `json:"id"`
	Severity   LintSeverity `json:"severity"`
	Category   LintCategory `json:"category"`
	Message    string       `json:"message"`
	Suggestion string       `json:"suggestion"`
	FieldPath  string       `json:"field_path,omitempty"`
	Sources    []string     `json:"sources,omitempty"` // platform, user_extra_args, user_env, backend_default
}

// LintResult holds the full lint output.
type LintResult struct {
	Status   string        `json:"status"` // ok, warning, error
	Findings []LintFinding `json:"findings"`
}

// LintInput holds all data needed for linting.
type LintInput struct {
	// Pre-dedup raw args (Layers 1-4, before deduplicateArgs)
	PreDedupArgs []string
	// Final resolved args (after deduplicateArgs + applyServiceArgs)
	FinalArgs []string
	// Environment variables (from all layers merged)
	Env map[string]string
	// Platform-owned parameters that should not be overridden
	PlatformOwnedParams []LogicalParamSpec
	// Backend name for context
	BackendName string
	// Docker spec for high-risk checks
	DockerSpec *DockerSpecInfo
	// Env source tracking: which env vars came from which layer
	// Key: env var name, Value: source (platform, backend_default, user_env, node_override)
	EnvSources map[string]string
	// RequiredParamKeys: parameter keys that are required (must have a value in final args)
	RequiredParamKeys []string
	// DisabledParamKeys: parameter keys that were disabled — should NOT appear in final args
	DisabledParamKeys []string
	// DisabledParamSources: where each disabled param was disabled (for error messages)
	DisabledParamSources map[string]string
	// ActiveParamKeys: all parameter keys that were active (enabled) during resolution
	ActiveParamKeys []string
	// Vendor: hardware vendor for compatibility checks (nvidia, amd, metax, cpu)
	Vendor string
	// BackendArgsSchema: known CLI flag names for the backend (to detect unsupported params)
	BackendArgsSchema []string
}

// LogicalParamSpec defines a logical parameter and its conflict policy.
type LogicalParamSpec struct {
	Name     string   `json:"name"`
	CLIFlags []string `json:"cli_flags"`
	EnvVars  []string `json:"env_vars"`
	Owner    string   `json:"owner"`    // platform, user, backend_default
	Conflict string   `json:"conflict"` // reject, warn, platform_wins, user_wins
}

// DefaultLogicalParamSpecs returns the default parameter specs for all backends.
func DefaultLogicalParamSpecs() []LogicalParamSpec {
	return []LogicalParamSpec{
		// llama.cpp
		{
			Name: "host", CLIFlags: []string{"--host"}, EnvVars: []string{"LLAMA_ARG_HOST"},
			Owner: "platform", Conflict: "reject",
		},
		{
			Name: "port", CLIFlags: []string{"--port"}, EnvVars: []string{"LLAMA_ARG_PORT"},
			Owner: "platform", Conflict: "reject",
		},
		// vLLM
		{
			Name: "model_path", CLIFlags: []string{"--model"}, EnvVars: []string{},
			Owner: "platform", Conflict: "reject",
		},
		{
			Name: "served_model_name", CLIFlags: []string{"--served-model-name"}, EnvVars: []string{},
			Owner: "platform", Conflict: "reject",
		},
		// SGLang
		{
			Name: "sglang_model_path", CLIFlags: []string{"--model-path"}, EnvVars: []string{},
			Owner: "platform", Conflict: "reject",
		},
	}
}

// LintRunPlan performs lint on a RunPlan.
// It runs three stages: pre-normalization, param rules, and final lint.
func LintRunPlan(in LintInput) LintResult {
	var findings []LintFinding

	// Stage 1: Pre-normalization lint (on raw args before dedup)
	preFindings := lintPreNormalization(in.PreDedupArgs, in.PlatformOwnedParams, in.Env, in.EnvSources)
	findings = append(findings, preFindings...)

	// Stage 2: Parameter rules (disabled applied, missing required, vendor incompatible, unsupported)
	paramFindings := lintParamRules(in)
	findings = append(findings, paramFindings...)

	// Stage 3: Final lint (on resolved args + env)
	finalFindings := lintFinal(in.FinalArgs, in.Env, in.PlatformOwnedParams, in.EnvSources, in.DockerSpec)
	findings = append(findings, finalFindings...)

	// Determine overall status
	status := "ok"
	for _, f := range findings {
		switch f.Severity {
		case LintSeverityError:
			status = "error"
		case LintSeverityWarning:
			if status != "error" {
				status = "warning"
			}
		case LintSeverityAdvisory:
			if status == "ok" {
				status = "warning"
			}
		}
	}

	return LintResult{Status: status, Findings: findings}
}

// lintPreNormalization checks for duplicate flags and user overrides of platform-owned params.
func lintPreNormalization(args []string, specs []LogicalParamSpec, env map[string]string, envSources map[string]string) []LintFinding {
	var findings []LintFinding

	// Detect duplicate CLI flags
	flagCounts := make(map[string][]int) // flag -> indices
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagCounts[arg] = append(flagCounts[arg], i)
		}
		if strings.HasPrefix(arg, "-") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i += 2
		} else {
			i++
		}
	}
	for flag, indices := range flagCounts {
		if len(indices) > 1 {
			findings = append(findings, LintFinding{
				ID:         "arg.duplicate",
				Severity:   LintSeverityError,
				Category:   LintCategoryDuplicateArg,
				Message:    fmt.Sprintf("CLI flag %q appears %d times", flag, len(indices)),
				Suggestion: fmt.Sprintf("Remove duplicate %q flags. The last occurrence will be used after deduplication.", flag),
				FieldPath:  "args",
				Sources:    []string{"user_extra_args"},
			})
		}
	}

	// Detect user overrides of platform-owned params
	for _, spec := range specs {
		if spec.Owner != "platform" || spec.Conflict != "reject" {
			continue
		}
		for _, cliFlag := range spec.CLIFlags {
			count := 0
			for _, arg := range args {
				if arg == cliFlag {
					count++
				}
			}
			if count > 1 {
				findings = append(findings, LintFinding{
					ID:         "arg.platform_overridden",
					Severity:   LintSeverityError,
					Category:   LintCategoryPlatformOverridden,
					Message:    fmt.Sprintf("Platform-owned parameter %q appears multiple times; user args may override platform defaults", cliFlag),
					Suggestion: fmt.Sprintf("Platform-owned parameter %q should not be duplicated. If you need to override it, use deployment parameters.", cliFlag),
					FieldPath:  "args",
					Sources:    []string{"platform", "user_extra_args"},
				})
			}
		}
	}

	return findings
}

// lintFinal checks the final resolved args and env for conflicts.
func lintFinal(args []string, env map[string]string, specs []LogicalParamSpec, envSources map[string]string, dockerSpec *DockerSpecInfo) []LintFinding {
	var findings []LintFinding

	// Build a set of CLI flags present in final args
	cliFlags := make(map[string]string) // flag -> value
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			cliFlags[arg] = args[i+1]
			i += 2
		} else {
			i++
		}
	}

	// Check env/CLI conflicts for known logical params
	for _, spec := range specs {
		// Check if any env var for this param is set
		for _, envVar := range spec.EnvVars {
			envVal, hasEnv := env[envVar]
			if !hasEnv {
				continue
			}

			// Check if any CLI flag for this param is also set
			for _, cliFlag := range spec.CLIFlags {
				if _, hasCLI := cliFlags[cliFlag]; !hasCLI {
					continue
				}

				// Both env and CLI are set — conflict!
				envSource := "unknown"
				if envSources != nil {
					if src, ok := envSources[envVar]; ok {
						envSource = src
					}
				}

				severity := LintSeverityWarning
				suggestion := fmt.Sprintf("Remove either the %s env var or the %s CLI flag to avoid ambiguity.", envVar, cliFlag)

				// Image-provided env is warning (not blocking)
				// User-provided env conflicting with platform CLI is error
				if envSource == "user_env" && spec.Owner == "platform" {
					severity = LintSeverityError
					suggestion = fmt.Sprintf("You set %s=%q which conflicts with platform-owned %s. Remove the env override.", envVar, envVal, cliFlag)
				} else if envSource == "platform" || envSource == "backend_default" {
					severity = LintSeverityWarning
					suggestion = fmt.Sprintf("The %s env var is set by the %s and conflicts with %s CLI. This is typically harmless as the CLI value takes precedence.", envVar, envSource, cliFlag)
				}

				findings = append(findings, LintFinding{
					ID:         "arg.env_cli_conflict",
					Severity:   severity,
					Category:   LintCategoryEnvCLIConflict,
					Message:    fmt.Sprintf("Environment variable %s=%q conflicts with CLI flag %s", envVar, envVal, cliFlag),
					Suggestion: suggestion,
					FieldPath:  "env+args",
					Sources:    []string{envSource, "platform"},
				})
			}
		}
	}

	// High-risk Docker flags
	if dockerSpec != nil {
		if dockerSpec.Privileged {
			findings = append(findings, LintFinding{
				ID:         "security.privileged_enabled",
				Severity:   LintSeverityWarning,
				Category:   LintCategoryHighRisk,
				Message:    "Container runs in privileged mode",
				Suggestion: "Privileged mode grants full host access. Use specific device mappings instead if possible.",
				FieldPath:  "docker.privileged",
				Sources:    []string{"platform"},
			})
		}
		if dockerSpec.IPCMode == "host" {
			findings = append(findings, LintFinding{
				ID:         "security.ipc_host",
				Severity:   LintSeverityWarning,
				Category:   LintCategoryHighRisk,
				Message:    "Container uses IPC mode host",
				Suggestion: "IPC host mode allows shared memory with the host. This may be required for some GPU workloads but reduces isolation.",
				FieldPath:  "docker.ipc_mode",
				Sources:    []string{"platform"},
			})
		}
		for _, opt := range dockerSpec.SecurityOptions {
			if opt == "seccomp=unconfined" || opt == "apparmor=unconfined" {
				findings = append(findings, LintFinding{
					ID:         "security.unconfined",
					Severity:   LintSeverityWarning,
					Category:   LintCategoryHighRisk,
					Message:    fmt.Sprintf("Container uses security option %q", opt),
					Suggestion: "Unconfined security profiles reduce container isolation.",
					FieldPath:  "docker.security_options",
					Sources:    []string{"platform"},
				})
			}
		}
	}

	return findings
}

// lintParamRules checks parameter-level rules: disabled-applied, missing-required,
// vendor-incompatible, and unsupported parameters.
func lintParamRules(in LintInput) []LintFinding {
	var findings []LintFinding

	// Build set of CLI flags from final args for quick lookup.
	finalFlagSet := make(map[string]bool)
	i := 0
	for i < len(in.FinalArgs) {
		arg := in.FinalArgs[i]
		if strings.HasPrefix(arg, "-") {
			finalFlagSet[arg] = true
		}
		if strings.HasPrefix(arg, "-") && i+1 < len(in.FinalArgs) && !strings.HasPrefix(in.FinalArgs[i+1], "-") {
			i += 2
		} else {
			i++
		}
	}

	// Build set of known CLI flag names from schema.
	schemaFlagSet := make(map[string]bool)
	for _, f := range in.BackendArgsSchema {
		schemaFlagSet[f] = true
	}

	// 1. Check: disabled params that still appear in final args.
	for _, key := range in.DisabledParamKeys {
		for flag := range finalFlagSet {
			if paramKeyMatchesFlag(key, flag) && !schemaFlagSet[flag] {
				source := "unknown"
				if in.DisabledParamSources != nil {
					if s, ok := in.DisabledParamSources[key]; ok {
						source = s
					}
				}
				findings = append(findings, LintFinding{
					ID:         "param.disabled_applied",
					Severity:   LintSeverityWarning,
					Category:   LintCategoryDisabledApplied,
					Message:    fmt.Sprintf("Disabled parameter %q (flag %s) appears in final args", key, flag),
					Suggestion: fmt.Sprintf("Parameter %q was disabled at layer %q but its CLI flag %s was found in the resolved args. This may indicate a config merge bug.", key, source, flag),
					FieldPath:  "params",
					Sources:    []string{source},
				})
			}
		}
	}

	// 2. Check: required params that are missing from final args.
	for _, key := range in.RequiredParamKeys {
		found := false
		for flag := range finalFlagSet {
			if paramKeyMatchesFlag(key, flag) {
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, LintFinding{
				ID:         "param.missing_required",
				Severity:   LintSeverityError,
				Category:   LintCategoryMissingRequired,
				Message:    fmt.Sprintf("Required parameter %q is not present in final args", key),
				Suggestion: fmt.Sprintf("Enable and set a value for required parameter %q before starting the deployment.", key),
				FieldPath:  "params",
				Sources:    []string{"platform"},
			})
		}
	}

	// 3. Check: vendor-incompatible params (CUDA-specific params on non-NVIDIA).
	nvidiaOnlyFlags := map[string]bool{
		"--gpu-memory-utilization": true, "--tensor-parallel-size": true,
		"--pipeline-parallel-size": true, "--kv-cache-dtype": true,
	}
	if in.Vendor != "" && in.Vendor != "nvidia" && in.Vendor != "cpu" {
		for flag := range finalFlagSet {
			if nvidiaOnlyFlags[flag] {
				findings = append(findings, LintFinding{
					ID:         "param.vendor_incompatible",
					Severity:   LintSeverityError,
					Category:   LintCategoryVendorIncompatible,
					Message:    fmt.Sprintf("NVIDIA-specific flag %s used with vendor=%q", flag, in.Vendor),
					Suggestion: fmt.Sprintf("Flag %s requires an NVIDIA GPU. Remove it for %s deployments.", flag, in.Vendor),
					FieldPath:  "params",
					Sources:    []string{"user"},
				})
			}
		}
	}

	// 4. Check: unknown flags in final args that aren't in schema.
	// Only flag user-provided args, not platform-managed ones (--model, --host, --port, etc.).
	platformManaged := map[string]bool{
		"--model": true, "--host": true, "--port": true,
		"--model-path": true, "-m": true,
	}
	for flag := range finalFlagSet {
		if schemaFlagSet[flag] || platformManaged[flag] || len(schemaFlagSet) == 0 {
			continue
		}
		findings = append(findings, LintFinding{
			ID:         "param.unsupported",
			Severity:   LintSeverityWarning,
			Category:   LintCategoryUnsupported,
			Message:    fmt.Sprintf("CLI flag %s is not in the backend's known args schema for %s", flag, in.BackendName),
			Suggestion: fmt.Sprintf("Remove flag %s or add it to the backend catalog's default_args_schema.", flag),
			FieldPath:  "params",
			Sources:    []string{"user"},
		})
	}

	return findings
}

// paramKeyMatchesFlag checks whether a parameter key (e.g. "backend.arg.max_model_len")
// could produce the given CLI flag (e.g. "--max-model-len").
func paramKeyMatchesFlag(paramKey, flag string) bool {
	// Convert param key suffix to CLI flag: "max_model_len" -> "--max-model-len"
	suffix := paramKey
	if idx := strings.LastIndex(paramKey, "."); idx >= 0 {
		suffix = paramKey[idx+1:]
	}
	expected := "--" + strings.ReplaceAll(suffix, "_", "-")
	return flag == expected
}
