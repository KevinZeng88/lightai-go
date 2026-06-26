package api

import (
	"lightai-go/internal/server/semanticconfig"
)

func semanticDeploymentSnapshot(configSet map[string]interface{}, service map[string]interface{}) semanticconfig.Snapshot {
	reg := semanticconfig.DefaultRegistry()
	snapshot, err := semanticconfig.NormalizeConfigSet(reg, configSet)
	if err != nil {
		return semanticconfig.Snapshot{Items: map[string]semanticconfig.SnapshotItem{}}
	}
	var fields []semanticconfig.PatchField
	if hostPort := intFromAny(service["host_port"], 0); hostPort > 0 {
		fields = append(fields, semanticconfig.PatchField{Key: "deployment.host_port", Value: hostPort})
	}
	if containerPort := intFromAny(service["container_port"], 0); containerPort > 0 {
		fields = append(fields, semanticconfig.PatchField{Key: "service.container_port", Value: containerPort})
	}
	if servedName := strVal(service, "served_model_name", ""); servedName != "" {
		fields = append(fields, semanticconfig.PatchField{Key: "deployment.served_model_name", Value: servedName})
	}
	if len(fields) == 0 {
		return snapshot
	}
	out, err := semanticconfig.ApplyPatch(reg, snapshot, fields)
	if err != nil {
		return snapshot
	}
	return out
}
