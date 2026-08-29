package runner

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/spelingbee/restored/internal/compose"
)

// labelCompose stamps com.restored.run on every service, network and volume the run
// creates, so orphans from a crashed process are always findable:
//
//	docker ps -aq --filter label=com.restored.run
//
// It runs on the interpolated file restored writes into the workspace, never on the
// recipe's own compose.yaml.
func labelCompose(raw []byte, runID string) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("compose.yaml: parsing the interpolated file: %w", err)
	}
	for _, section := range []string{"services", "networks", "volumes"} {
		entries, ok := doc[section].(map[string]any)
		if !ok {
			continue
		}
		for name, v := range entries {
			body, ok := v.(map[string]any)
			if !ok || body == nil {
				body = map[string]any{}
			}
			body["labels"] = addLabel(body["labels"], runID)
			entries[name] = body
		}
		doc[section] = entries
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("compose.yaml: writing the interpolated file: %w", err)
	}
	return out, nil
}

// addLabel adds the run label to whichever of the two label syntaxes is already there.
func addLabel(existing any, runID string) any {
	label := compose.LabelRun + "=" + runID
	switch t := existing.(type) {
	case map[string]any:
		t[compose.LabelRun] = runID
		return t
	case []any:
		return append(t, label)
	default:
		return map[string]any{compose.LabelRun: runID}
	}
}
