package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// silencer reads the one key the nudge needs. It is deliberately not Load: the nudge
// is a courtesy printed after a PASS, and no failure to parse a configuration file
// should turn into an error a user has to deal with while they are doing something
// else. Load's strictness is for --target and --all, where the file is the input.
type silencer struct {
	Defaults struct {
		Nudge *bool `yaml:"nudge"`
	} `yaml:"defaults"`
}

// NudgeSilenced reports whether a configuration file turns the contribution
// invitation off: `defaults.nudge: false` is the config equivalent of --no-nudge.
// A missing file, an unreadable one, and a malformed one all mean "not silenced".
//
// This took over internal/nudge's one-key reader when internal/config landed
// (ADR-045, ADR-067); the search order and the semantics are unchanged.
func NudgeSilenced(explicit string) bool {
	for _, path := range SearchPath(explicit) {
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
