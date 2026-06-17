package runplan

import (
	"fmt"
	"regexp"
	"strings"
)

// varPattern matches {{VAR_NAME}} patterns (GPUStack style).
var varPattern = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// substituteVars replaces {{var}} placeholders with values from vars.
// Unknown variables return an error (not a warning, not silently preserved).
// ${VAR} syntax is NOT supported.
func substituteVars(template string, vars map[string]string) (string, error) {
	var unknown []string
	result := varPattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name between {{ and }}
		name := match[2 : len(match)-2]
		if val, ok := vars[name]; ok {
			return val
		}
		unknown = append(unknown, name)
		return match // keep original if unknown (error reported below)
	})

	if len(unknown) > 0 {
		return result, fmt.Errorf("undefined variable(s): %s", strings.Join(unknown, ", "))
	}

	return result, nil
}
