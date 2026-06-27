package runplan

import (
	"fmt"

	"lightai-go/internal/server/semanticconfig"
)

func ApplySemanticSnapshot(in ResolveInput, snapshot semanticconfig.Snapshot, backendName string) ResolveInput {
	if in.Deployment == nil {
		in.Deployment = &DeploymentInfo{}
	}
	if in.Deployment.Parameters == nil {
		in.Deployment.Parameters = map[string]interface{}{}
	}
	if item, ok := snapshot.Items["deployment.host_port"]; ok {
		in.Deployment.Service.HostPort = intFromSemantic(item.Value)
	}
	if item, ok := snapshot.Items["service.container_port"]; ok {
		port := intFromSemantic(item.Value)
		in.Deployment.Service.ContainerPort = port
		in.Deployment.Service.AppPort = port
	}
	if item, ok := snapshot.Items["service.listen_host"]; ok {
		if host := fmt.Sprintf("%v", item.Value); host != "" && host != "<nil>" {
			in.Deployment.Service.ListenHost = host
		}
	}
	if item, ok := snapshot.Items["deployment.served_model_name"]; ok {
		value := fmt.Sprintf("%v", item.Value)
		flag := adapterFlag(backendName, "deployment.served_model_name")
		if flag != "" {
			in.Deployment.Parameters["served_model_name"] = value
			in.Deployment.ParameterValues = upsertParameterValue(in.Deployment.ParameterValues, ParameterValue{
				Key:         "deployment.served_model_name",
				Type:        "string",
				Target:      "arg",
				CliName:     flag,
				RenderStyle: "flag_space_value",
				Enabled:     item.Enabled,
				Value:       value,
				Source:      "semantic_snapshot",
				CopiedFrom:  item.CopiedFrom,
			})
		}
	}
	if item, ok := snapshot.Items["model_runtime.max_model_len"]; ok {
		value := item.Value
		in.Deployment.Parameters["max_model_len"] = value
		in.Deployment.ParameterValues = upsertParameterValue(in.Deployment.ParameterValues, ParameterValue{
			Key:         "model_runtime.max_model_len",
			Type:        "integer",
			Target:      "arg",
			CliName:     adapterFlag(backendName, "model_runtime.max_model_len"),
			RenderStyle: "flag_space_value",
			Enabled:     item.Enabled,
			Value:       value,
			Source:      "semantic_snapshot",
			CopiedFrom:  item.CopiedFrom,
		})
	}
	if item, ok := snapshot.Items["model_runtime.gpu_memory_utilization"]; ok {
		value := item.Value
		in.Deployment.Parameters["gpu_memory_utilization"] = value
		in.Deployment.ParameterValues = upsertParameterValue(in.Deployment.ParameterValues, ParameterValue{
			Key:         "model_runtime.gpu_memory_utilization",
			Type:        "number",
			Target:      "arg",
			CliName:     adapterFlag(backendName, "model_runtime.gpu_memory_utilization"),
			RenderStyle: "flag_space_value",
			Enabled:     item.Enabled,
			Value:       value,
			Source:      "semantic_snapshot",
			CopiedFrom:  item.CopiedFrom,
		})
	}
	return in
}

func adapterFlag(backendName, semanticKey string) string {
	switch backendName {
	case "sglang":
		switch semanticKey {
		case "model_runtime.max_model_len":
			return "--context-length"
		case "model_runtime.gpu_memory_utilization":
			return "--mem-fraction-static"
		case "deployment.served_model_name":
			return "--served-model-name"
		}
	case "llamacpp":
		switch semanticKey {
		case "model_runtime.max_model_len":
			return "--ctx-size"
		}
	default:
		switch semanticKey {
		case "model_runtime.max_model_len":
			return "--max-model-len"
		case "model_runtime.gpu_memory_utilization":
			return "--gpu-memory-utilization"
		case "deployment.served_model_name":
			return "--served-model-name"
		}
	}
	return ""
}

func upsertParameterValue(values []ParameterValue, next ParameterValue) []ParameterValue {
	if next.CliName == "" {
		return values
	}
	for i, existing := range values {
		if existing.Key == next.Key || existing.CliName == next.CliName {
			values[i] = next
			return values
		}
	}
	return append(values, next)
}

func intFromSemantic(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}
