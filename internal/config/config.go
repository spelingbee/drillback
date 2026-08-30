// Package config loads restored.yaml: sources, targets, defaults, and the precedence
// chain of SPEC.md section 2.9. It reads exactly one file and resolves nothing else:
// recipes are loaded by the caller, restic is never invoked from here, and the only
// I/O is reading the file it is given. See SPEC.md section 13.1 and DECISIONS.md
// ADR-067.
//
// Precedence, lowest to highest: recipe defaults, `defaults:`, the target block, then
// command-line flags. This package owns the middle two; the caller owns the outer two,
// because only the caller knows which flags a user actually typed.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is what restored.yaml is looked for under.
const FileName = "restored.yaml"

// Version is the one config schema version this build reads.
const Version = 1

// Config is one parsed restored.yaml.
type Config struct {
	Version  int               `yaml:"version"`
	Defaults Defaults          `yaml:"defaults"`
	Sources  map[string]Source `yaml:"sources"`
	Targets  map[string]Target `yaml:"targets"`

	// Path is where the file was read from. Relative host paths inside the file -
	// a target's recipe directory, a password_file, a dir source's path, a
	// workspace - resolve against its directory, not against the working directory:
	// the config is discovered by a search order and read from cron, so "relative
	// to wherever the process happened to start" would point somewhere different
	// on every invocation.
	Path string `yaml:"-"`

	// TargetOrder is the targets in file order, which is the order --all runs them
	// in (ADR-068). A YAML map carries its order in the document even though the
	// decoded Go map does not.
	TargetOrder []string `yaml:"-"`
}

// Defaults is the `defaults:` block: what a target uses when it says nothing.
type Defaults struct {
	Source         string   `yaml:"source"`
	Timeout        Duration `yaml:"timeout"`
	RestoreTimeout Duration `yaml:"restore_timeout"`
	ReadyTimeout   Duration `yaml:"ready_timeout"`
	CheckTimeout   Duration `yaml:"check_timeout"`
	Pull           string   `yaml:"pull"`
	Nudge          *bool    `yaml:"nudge"`
	Workspace      string   `yaml:"workspace"`
}

// Source is one entry of `sources:`.
type Source struct {
	Kind string `yaml:"kind"`

	// restic
	Repository      string            `yaml:"repository"`
	PasswordFile    string            `yaml:"password_file"`
	PasswordCommand []string          `yaml:"password_command"`
	Host            string            `yaml:"host"`
	Env             map[string]string `yaml:"env"`

	// dir
	Path string `yaml:"path"`
}

// Target is one entry of `targets:`.
type Target struct {
	Recipe  string            `yaml:"recipe"`
	Source  string            `yaml:"source"`
	Tags    []string          `yaml:"tags"`
	Inputs  map[string]string `yaml:"inputs"`
	Set     map[string]string `yaml:"set"`
	Enabled *bool             `yaml:"enabled"`

	Timeout        Duration `yaml:"timeout"`
	RestoreTimeout Duration `yaml:"restore_timeout"`
	ReadyTimeout   Duration `yaml:"ready_timeout"`
	CheckTimeout   Duration `yaml:"check_timeout"`
	Pull           string   `yaml:"pull"`
	Workspace      string   `yaml:"workspace"`
}

// SearchPath is SPEC.md section 2.9: --config overrides the search; otherwise
// ./restored.yaml, then $XDG_CONFIG_HOME/restored/restored.yaml (falling back to
// ~/.config when XDG_CONFIG_HOME is unset, which is what the variable means), then
// /etc/restored/restored.yaml. First match wins.
func SearchPath(explicit string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	paths := []string{FileName}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "restored", FileName))
	} else if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "restored", FileName))
	}
	return append(paths, filepath.Join("/etc", "restored", FileName))
}

// Discover returns the first config file that exists on the search path, or an error
// naming every path it looked at. --target and --all need a file to exist; a missing
// config is a loud answer, not an empty one.
func Discover(explicit string) (string, error) {
	searched := SearchPath(explicit)
	for _, path := range searched {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	if explicit != "" {
		return "", fmt.Errorf("--config %s: no such file", explicit)
	}
	return "", fmt.Errorf("no %s found (searched: %s)", FileName, strings.Join(searched, ", "))
}

// Load reads and validates one restored.yaml.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	cfg, err := parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	cfg.Path = path
	return cfg, nil
}

func parse(raw []byte) (*Config, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	// Strict: an unknown key is a typo, and a typo in a config file that silently
	// does nothing is how `enabld: false` runs a target the user turned off.
	dec.KnownFields(true)
	cfg := &Config{}
	if err := dec.Decode(cfg); err != nil {
		return nil, err
	}
	cfg.TargetOrder = targetOrder(raw)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// targetOrder reads the keys of the `targets:` mapping in document order. The strict
// decode above has already established the document parses and the keys are known.
func targetOrder(raw []byte) []string {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "targets" {
			continue
		}
		targets := root.Content[i+1]
		if targets.Kind != yaml.MappingNode {
			return nil
		}
		var order []string
		for j := 0; j+1 < len(targets.Content); j += 2 {
			order = append(order, targets.Content[j].Value)
		}
		return order
	}
	return nil
}

func (c *Config) validate() error {
	switch {
	case c.Version == 0:
		return fmt.Errorf("missing `version: 1` (this build reads config version %d)", Version)
	case c.Version != Version:
		return fmt.Errorf("config version %d: this build reads version %d", c.Version, Version)
	}
	for name, s := range c.Sources {
		switch s.Kind {
		case "restic":
			if s.Repository == "" {
				return fmt.Errorf("source %q: kind restic needs a repository", name)
			}
			if s.Path != "" {
				return fmt.Errorf("source %q: `path` belongs to kind dir, not restic", name)
			}
		case "dir":
			if s.Path == "" {
				return fmt.Errorf("source %q: kind dir needs a path", name)
			}
			for field, set := range map[string]bool{
				"repository":       s.Repository != "",
				"password_file":    s.PasswordFile != "",
				"password_command": len(s.PasswordCommand) > 0,
				"host":             s.Host != "",
				"env":              len(s.Env) > 0,
			} {
				if set {
					return fmt.Errorf("source %q: `%s` belongs to kind restic, not dir", name, field)
				}
			}
		case "":
			return fmt.Errorf("source %q: missing kind (restic or dir)", name)
		default:
			return fmt.Errorf("source %q: unknown kind %q (restored reads restic or dir)", name, s.Kind)
		}
	}
	if d := c.Defaults.Source; d != "" {
		if _, ok := c.Sources[d]; !ok {
			return fmt.Errorf("defaults.source %q: no such source (have: %s)", d, strings.Join(sourceNames(c.Sources), ", "))
		}
	}
	if err := validPull("defaults.pull", c.Defaults.Pull); err != nil {
		return err
	}
	for name, t := range c.Targets {
		if t.Recipe == "" {
			return fmt.Errorf("target %q: missing recipe", name)
		}
		src := t.Source
		if src == "" {
			src = c.Defaults.Source
		}
		if src == "" {
			return fmt.Errorf("target %q: no source, and defaults.source is not set", name)
		}
		if _, ok := c.Sources[src]; !ok {
			return fmt.Errorf("target %q: no such source %q (have: %s)", name, src, strings.Join(sourceNames(c.Sources), ", "))
		}
		if err := validPull(fmt.Sprintf("target %q: pull", name), t.Pull); err != nil {
			return err
		}
	}
	return nil
}

func validPull(where, v string) error {
	switch v {
	case "", "always", "missing", "never":
		return nil
	default:
		return fmt.Errorf("%s %q: expected always, missing or never", where, v)
	}
}

func sourceNames(m map[string]Source) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sortStrings(names)
	return names
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// envRef matches ${NAME} in a sources.env value.
var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// resolveEnv expands ${NAME} from the process environment. The values are read from
// the environment of the restored process, never out of the file (SPEC.md 2.9); an
// unset variable is a loud error, because the alternative is restic failing later
// with a message about credentials that never mentions the config file.
func resolveEnv(sourceName string, env map[string]string) ([]string, error) {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sortStrings(keys)
	out := make([]string, 0, len(env))
	for _, k := range keys {
		v := env[k]
		var missing string
		expanded := envRef.ReplaceAllStringFunc(v, func(ref string) string {
			name := envRef.FindStringSubmatch(ref)[1]
			val, ok := os.LookupEnv(name)
			if !ok && missing == "" {
				missing = name
			}
			return val
		})
		if missing != "" {
			return nil, fmt.Errorf("source %q: env %s references ${%s}, which is not set in the environment", sourceName, k, missing)
		}
		out = append(out, k+"="+expanded)
	}
	return out, nil
}
