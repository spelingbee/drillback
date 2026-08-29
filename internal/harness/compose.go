package harness

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// withExportMount gives every service the harness's export directory, at /export and
// under the environment variable $RESTORED_EXPORT.
//
// The name appears twice with two meanings, which is worth stating once: inside
// compose.yaml, ${RESTORED_EXPORT} is interpolated by restored and is a *host* path,
// so a recipe can mount it somewhere of its own choosing. Inside a container,
// $RESTORED_EXPORT is an ordinary environment variable and is /export. An export step
// writes to the second one.
//
// This runs only in the harness. During `restored check` no service sees /export at
// all, so a recipe cannot come to depend on it for anything but its own test.
func withExportMount(raw []byte, hostExport string) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("compose.yaml: parsing the interpolated file: %w", err)
	}
	services, ok := doc["services"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("compose.yaml: no services")
	}
	bind := dockerPath(hostExport) + ":" + exportMount
	for _, name := range sortedKeys(services) {
		body, ok := services[name].(map[string]any)
		if !ok || body == nil {
			body = map[string]any{}
		}
		body["volumes"] = appendVolume(body["volumes"], bind)
		body["environment"] = setEnv(body["environment"], "RESTORED_EXPORT", exportMount)
		services[name] = body
	}
	doc["services"] = services
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("compose.yaml: writing the harness file: %w", err)
	}
	return out, nil
}

func appendVolume(existing any, bind string) any {
	switch t := existing.(type) {
	case []any:
		return append(t, bind)
	default:
		return []any{bind}
	}
}

// setEnv writes one variable into whichever of the two environment syntaxes the
// service already uses.
func setEnv(existing any, key, value string) any {
	switch t := existing.(type) {
	case map[string]any:
		t[key] = value
		return t
	case []any:
		return append(t, key+"="+value)
	default:
		return map[string]any{key: value}
	}
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
