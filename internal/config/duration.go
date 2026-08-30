package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that reads YAML scalars like "15m" and "90s". A bare
// number is rejected the way time.ParseDuration rejects it: "90" could mean seconds
// to the person who wrote it and milliseconds to the tool, and a config file is the
// wrong place to guess.
type Duration time.Duration

// D returns the underlying time.Duration.
func (d Duration) D() time.Duration { return time.Duration(d) }

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("line %d: a duration is a string like \"15m\" or \"90s\"", node.Line)
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("line %d: %q is not a duration (write \"15m\", \"90s\", \"1h30m\")", node.Line, s)
	}
	if parsed < 0 {
		return fmt.Errorf("line %d: a negative duration (%q) budgets nothing", node.Line, s)
	}
	*d = Duration(parsed)
	return nil
}
