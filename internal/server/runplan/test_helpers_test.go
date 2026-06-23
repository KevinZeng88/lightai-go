package runplan

import "strings"

// makeNbrSnapshotFromInput creates an NBR snapshot from ResolveInput's BV/BR data.
// At NBR creation time, the server freezes BV default_args + BR args_override into NBR snapshot.
// This helper simulates that frozen state for tests.
func makeNbrSnapshotFromInput(in ResolveInput) *NBRSnapshotInfo {
	// Combine BV default_args + BR args_override (as frozen at NBR creation)
	var allArgs []string
	allArgs = append(allArgs, in.BackendVersion.DefaultArgs...)
	allArgs = append(allArgs, in.BackendVersion.DefaultBackendParams...)
	allArgs = append(allArgs, in.BackendRuntime.ArgsOverride...)

	// Combine Backend + BV + BR env (as frozen at NBR creation)
	allEnv := make(map[string]string)
	for k, v := range in.Backend.DefaultEnv {
		allEnv[k] = v
	}
	for k, v := range in.BackendVersion.Env {
		allEnv[k] = v
	}
	for k, v := range in.BackendRuntime.DefaultEnv {
		allEnv[k] = v
	}

	// Entrypoint: BR override > BV default
	entrypoint := in.BackendVersion.DefaultEntrypoint
	if len(in.BackendRuntime.EntrypointOverride) > 0 {
		entrypoint = in.BackendRuntime.EntrypointOverride
	}

	snapshot := &NBRSnapshotInfo{
		ArgsOverride:       allArgs,
		DefaultEnv:         allEnv,
		EntrypointOverride: entrypoint,
		Docker:             in.BackendRuntime.Docker,
		ModelMount:         in.BackendRuntime.ModelMount,
		ParameterSchema:    in.BackendVersion.ParameterDefs,
		ParameterValues:    []ParameterValue{},
	}
	if in.BackendRuntime.HealthCheckOverride != nil {
		snapshot.HealthCheckOverride = in.BackendRuntime.HealthCheckOverride
	}
	return snapshot
}

// ensureNbrSnapshot adds NBRConfigSnapshot to input if not already set.
func ensureNbrSnapshot(in ResolveInput) ResolveInput {
	if in.NBRConfigSnapshot == nil {
		in.NBRConfigSnapshot = makeNbrSnapshotFromInput(in)
	}
	return in
}

// containsArg checks if args contain a specific flag or value.
func containsArg(args []string, target string) bool {
	for _, a := range args {
		if strings.Contains(a, target) {
			return true
		}
	}
	return false
}
