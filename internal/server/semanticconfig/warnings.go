package semanticconfig

import "fmt"

func EvaluateWarnings(reg *Registry, snapshot Snapshot) map[string][]Warning {
	out := map[string][]Warning{}
	contextLen := intFromAny(snapshot.Items["model_runtime.context_length"].Value, 0)
	maxLen := intFromAny(snapshot.Items["model_runtime.max_model_len"].Value, 0)
	if contextLen > 0 && maxLen > contextLen {
		out["model_runtime.max_model_len"] = append(out["model_runtime.max_model_len"], Warning{
			Code:        "value_risk",
			SemanticKey: "model_runtime.max_model_len",
			Message:     fmt.Sprintf("max_model_len %d is above model context length %d", maxLen, contextLen),
		})
	}
	if privileged, ok := snapshot.Items["docker.privileged"].Value.(bool); ok && privileged {
		out["docker.privileged"] = append(out["docker.privileged"], Warning{
			Code:        "security_risk",
			SemanticKey: "docker.privileged",
			Message:     "privileged container weakens isolation",
		})
	}
	return out
}
