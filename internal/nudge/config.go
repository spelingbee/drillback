package nudge

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigName is the file `restored.yaml` is looked for under.
const ConfigName = "restored.yaml"

// silencer is the one key this package reads out of restored.yaml. Everything else in
// that file belongs to internal/config, which does not exist yet (DECISIONS.md
// ADR-045), and this deliberately reads nothing else: `defaults.nudge: false` is the
// config equivalent of --no-nudge, and a user who has written it should not have to
// wait for the rest of the configuration system to be believed.
//
// When internal/config lands it takes this over, and the search order below is the
// one SPEC.md section 2.9 specifies, so the behaviour will not change under anyone.
type silencer struct {
	Defaults struct {
		Nudge *bool `yaml:"nudge"`
	} `yaml:"defaults"`
}

// Silenced reports whether a configuration file turns the invitation off.
//
// A missing file, an unreadable one, and a malformed one all mean "not silenced": the
// nudge is a courtesy, and no failure to parse a configuration file should turn into
// an error a user has to deal with while they are doing something else.
func Silenced(explicit string) bool {
	for _, path := range configSearchPath(explicit) {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s silencer
		if err := yaml.Unmarshal(raw, &s); err != nil {
			return false
		}
		return s.Defaults.Nudge != nil && !*s.Defaults.Nudge
	}
	return false
}

// configSearchPath is SPEC.md section 2.9: --config overrides the search; otherwise
// ./restored.yaml, then $XDG_CONFIG_HOME/restored/restored.yaml, then
// /etc/restored/restored.yaml. First match wins.
func configSearchPath(explicit string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	paths := []string{ConfigName}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "restored", ConfigName))
	} else if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "restored", ConfigName))
	}
	return append(paths, filepath.Join("/etc", "restored", ConfigName))
}
