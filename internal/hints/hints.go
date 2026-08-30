// Package hints matches a catalog of error patterns against what a failing run
// produced, and offers at most one likely cause.
//
// A hint is presentation only. It can never change the verdict or the exit code.
package hints

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	drillback "github.com/spelingbee/drillback"
)

// Rule is one entry in the catalog.
type Rule struct {
	ID       string   `yaml:"id"`
	When     *When    `yaml:"when,omitempty"`
	Match    string   `yaml:"match"`
	Title    string   `yaml:"title"`
	Text     string   `yaml:"text"`
	Commands []string `yaml:"commands,omitempty"`

	re *regexp.Regexp
}

// When scopes a rule so a Postgres rule cannot fire on a SQLite failure.
type When struct {
	Driver string `yaml:"driver,omitempty"`
}

// Catalog is an ordered set of rules. Order is the whole mechanism: rules are written
// most specific first, and the first match wins.
type Catalog struct {
	Version int    `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

// Subject is one piece of evidence a rule may match against.
type Subject struct {
	// Where is the JSON pointer-ish location the report records, for example
	// "checks[1].observed.error" or "logs.db".
	Where string
	Text  string
	// Driver scopes the subject, so a `when: {driver: postgres}` rule only sees
	// Postgres evidence.
	Driver string
}

// Builtin is the catalog compiled into the binary.
func Builtin() (*Catalog, error) {
	raw, err := drillback.Hints.ReadFile("docs/hints.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading the built-in hint catalog: %w", err)
	}
	return Parse(raw)
}

// Load reads an additional catalog from disk. Its rules are matched before the
// built-in ones, so a user or a distribution can add rules without a rebuild.
func Load(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading hints from %q: %w", path, err)
	}
	return Parse(raw)
}

// Parse decodes a catalog and compiles every pattern, so a broken regular expression
// is a load-time error rather than a silent non-match at the worst moment.
func Parse(raw []byte) (*Catalog, error) {
	var c Catalog
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parsing the hint catalog: %w", err)
	}
	seen := map[string]bool{}
	for i := range c.Rules {
		r := &c.Rules[i]
		switch {
		case r.ID == "":
			return nil, fmt.Errorf("hint rule %d has no id", i)
		case r.Match == "":
			return nil, fmt.Errorf("hint rule %q has no match", r.ID)
		case r.Title == "":
			return nil, fmt.Errorf("hint rule %q has no title", r.ID)
		case r.Text == "":
			return nil, fmt.Errorf("hint rule %q has no text", r.ID)
		case seen[r.ID]:
			return nil, fmt.Errorf("hint rule %q is defined twice", r.ID)
		}
		seen[r.ID] = true
		re, err := regexp.Compile(r.Match)
		if err != nil {
			return nil, fmt.Errorf("hint rule %q: match does not compile: %w", r.ID, err)
		}
		r.re = re
	}
	return &c, nil
}

// Concat puts an extra catalog's rules in front of this one's.
func Concat(first, second *Catalog) *Catalog {
	out := &Catalog{Version: first.Version}
	out.Rules = append(out.Rules, first.Rules...)
	out.Rules = append(out.Rules, second.Rules...)
	return out
}

// Match returns the first rule that matches any subject, in subject order and then in
// rule order, together with where it matched. At most one hint is ever shown: a list
// of five possible causes is a list of five things to ignore.
func (c *Catalog) Match(subjects []Subject) (*Rule, string, bool) {
	for _, s := range subjects {
		if s.Text == "" {
			continue
		}
		for i := range c.Rules {
			r := &c.Rules[i]
			if r.When != nil && r.When.Driver != "" && r.When.Driver != s.Driver {
				continue
			}
			if r.re.MatchString(s.Text) {
				return r, s.Where, true
			}
		}
	}
	return nil, "", false
}

// CommandContext is the restricted context a rule's commands are rendered against.
// They are printed verbatim and never executed.
type CommandContext struct {
	// Inputs maps an input name to the path in the user's backup, not in the
	// workspace: the commands tell the user how to inspect their own file.
	Inputs map[string]string
	// SnapshotID is the snapshot the run used, for the restic commands.
	SnapshotID string
}

// RenderCommands expands a rule's commands. A command that will not render is dropped
// rather than shown broken.
func (r *Rule) RenderCommands(ctx CommandContext) []string {
	if len(r.Commands) == 0 {
		return nil
	}
	inputs := make(map[string]any, len(ctx.Inputs))
	for name, p := range ctx.Inputs {
		inputs[name] = map[string]any{"path": p}
	}
	// `restic ls  | head -50` helps nobody. When the run failed before it selected a
	// snapshot - which is when these hints fire most - `latest` is what the user
	// would have typed anyway.
	snapshotID := ctx.SnapshotID
	if snapshotID == "" {
		snapshotID = "latest"
	}
	data := map[string]any{
		"input":    inputs,
		"snapshot": map[string]any{"id": snapshotID},
	}
	var out []string
	for _, cmd := range r.Commands {
		t, err := template.New("hint").Option("missingkey=error").Parse(cmd)
		if err != nil {
			continue
		}
		var b strings.Builder
		if err := t.Execute(&b, data); err != nil {
			continue
		}
		out = append(out, b.String())
	}
	return out
}
