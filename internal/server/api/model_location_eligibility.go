package api

import (
	"fmt"
	"strings"
)

var deployableVerificationStatuses = map[string]bool{
	"verified":          true,
	"warning":           true,
	"manually_accepted": true,
}

var deployableMatchStatuses = map[string]bool{
	"exact_match":     true,
	"probable_match":  true,
	"manual_attested": true,
}

func (h *AgentHandler) findDeployableModelLocation(modelArtifactID, nodeID string) (map[string]interface{}, []map[string]interface{}, string) {
	artifactName := modelArtifactID
	if artifact := h.getArtifactJSON(modelArtifactID); artifact != nil {
		artifactName = strVal(artifact, "display_name", "")
		if artifactName == "" {
			artifactName = strVal(artifact, "name", "")
		}
		if artifactName == "" {
			artifactName = modelArtifactID
		}
	}
	locations := h.listModelLocations(modelArtifactID)
	for _, loc := range locations {
		if strVal(loc, "node_id", "") != nodeID {
			continue
		}
		verificationStatus := strVal(loc, "verification_status", "")
		matchStatus := strVal(loc, "match_status", "")
		if deployableVerificationStatuses[verificationStatus] && deployableMatchStatuses[matchStatus] {
			return loc, locations, ""
		}
	}
	return nil, locations, modelLocationEligibilityReason(modelArtifactID, artifactName, nodeID, locations)
}

func modelLocationEligibilityReason(modelArtifactID, artifactName, nodeID string, locations []map[string]interface{}) string {
	visible := make([]string, 0, len(locations))
	for _, loc := range locations {
		visible = append(visible, fmt.Sprintf("id=%s node_id=%s verification_status=%s match_status=%s last_error=%s",
			strVal(loc, "id", ""),
			strVal(loc, "node_id", ""),
			strVal(loc, "verification_status", ""),
			strVal(loc, "match_status", ""),
			strVal(loc, "last_error", ""),
		))
	}
	if len(visible) == 0 {
		visible = append(visible, "<none>")
	}
	return fmt.Sprintf("model_location_missing: model_artifact_id=%s model=%s selected node_id=%s has no deployable location; deployable verification_status in [verified warning manually_accepted] and match_status in [exact_match probable_match manual_attested]; visible locations: %s",
		modelArtifactID, artifactName, nodeID, strings.Join(visible, "; "))
}
