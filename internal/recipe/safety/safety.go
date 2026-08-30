// Package safety enforces the isolation rules on a recipe's compose.yaml.
//
// The rules are hard failures, never warnings, and they are enforced by a schema
// rather than by discipline. See SPEC.md sections 3.5 and 9.
package safety

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	restored "github.com/spelingbee/restored"
	"github.com/spelingbee/restored/internal/recipe"
)

var composeSchema = mustCompile("schema/compose-safety.schema.json")

func mustCompile(path string) *jsonschema.Schema {
	b, err := restored.Schemas.ReadFile(path)
	if err != nil {
		panic(err)
	}
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(b)))
	if err != nil {
		panic(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(path, doc); err != nil {
		panic(err)
	}
	s, err := c.Compile(path)
	if err != nil {
		panic(err)
	}
	return s
}

// Compose is the small typed view of a compose file that the Go-only rules need.
type Compose struct {
	Services map[string]struct {
		Image   string   `yaml:"image"`
		Volumes []any    `yaml:"volumes"`
		Profile []string `yaml:"profiles"`
	} `yaml:"services"`
	Networks map[string]any `yaml:"networks"`
	Volumes  map[string]any `yaml:"volumes"`
}

// ServiceNames returns the compose services, sorted, so error messages are stable.
func (c *Compose) ServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for n := range c.Services {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Parse reads a compose file, refusing YAML tags before anything else looks at it.
func Parse(raw []byte) (*Compose, error) {
	if err := recipe.RejectYAMLTags(raw); err != nil {
		return nil, fmt.Errorf("compose.yaml: %w", err)
	}
	var c Compose
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("compose.yaml: parsing YAML: %w", err)
	}
	if len(c.Services) == 0 {
		return nil, errors.New("compose.yaml: no services")
	}
	return &c, nil
}

// ValidateSchema checks compose.yaml against schema/compose-safety.schema.json.
//
// It runs on the file as written, with ${RESTORED_*} placeholders still in it: the
// schema's volume rule can only recognise a bind mount restored controls while the
// placeholder is intact. Containment of the resolved paths is checked separately by
// CheckResolvedMounts, after interpolation. See DECISIONS.md ADR-039.
func ValidateSchema(raw []byte) error {
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("compose.yaml: parsing YAML: %w", err)
	}
	// The schema expresses the forbidden keys as one `not`, which is correct and
	// tells a contributor nothing about which key they used. The same rules are
	// checked here first, so the message names the service and the key.
	if err := checkForbiddenKeys(doc); err != nil {
		return err
	}
	// Before the schema, because JSON Schema expresses the tag rule as a `not` and
	// reports it as `services.app.image: 'not' failed`, which tells a contributor
	// nothing at all. The most common way to hit it is copying a compose file that
	// says :latest. See docs/review/maintainer.md MNT-10.
	if err := checkImages(doc); err != nil {
		return err
	}
	norm, err := recipe.Normalise(doc)
	if err != nil {
		return fmt.Errorf("compose.yaml: %w", err)
	}
	if err := composeSchema.Validate(norm); err != nil {
		return fmt.Errorf("compose.yaml: %w", recipe.FlattenSchemaError(err))
	}
	return nil
}

var placeholderRe = regexp.MustCompile(`\$(?:\{([A-Za-z_][A-Za-z0-9_]*)\}|([A-Za-z_][A-Za-z0-9_]*))`)

// Placeholders lists the distinct ${NAME} and $NAME references in a compose file,
// ignoring the $$ escape.
func Placeholders(raw []byte) []string {
	s := stripEscapes(string(raw))
	seen := map[string]bool{}
	var out []string
	for _, m := range placeholderRe.FindAllStringSubmatch(s, -1) {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// stripEscapes removes $$ pairs so they cannot look like a reference.
func stripEscapes(s string) string { return strings.ReplaceAll(s, "$$", "") }

// KnownNames is the set of placeholders restored itself defines for a resolution.
func KnownNames(r *recipe.Resolved) map[string]bool {
	known := map[string]bool{
		"RESTORED_RUN_ID":      true,
		"RESTORED_TEST_ASSETS": true,
		"RESTORED_EXPORT":      true,
	}
	for k := range r.Vars {
		known["RESTORED_VAR_"+k] = true
	}
	for _, in := range r.Inputs {
		if in.Mount != nil {
			known[in.Mount.Env] = true
		}
	}
	return known
}

// CheckPlaceholders is Go-only rule 2: every ${...} in compose.yaml must be one
// restored defines. An unset variable silently expanding to the empty string is how a
// volume mount becomes "/".
func CheckPlaceholders(raw []byte, r *recipe.Resolved) error {
	known := KnownNames(r)
	var unknown []string
	for _, name := range Placeholders(raw) {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	return fmt.Errorf("compose.yaml refers to %s, which restored does not define; "+
		"only ${RESTORED_VAR_<var>}, ${RESTORED_INPUT_<input>}, ${RESTORED_TEST_ASSETS}, "+
		"${RESTORED_EXPORT} and ${RESTORED_RUN_ID} are available",
		quoteAll(unknown))
}

// CheckServiceReferences is Go-only rule 3: every service named by a mount, a probe,
// a check, or a harness step must exist in compose.yaml. A typo in a service name is
// caught here, not sixty seconds into a run.
func CheckServiceReferences(c *Compose, r *recipe.Recipe) error {
	var problems []string
	note := func(where, service string) {
		if service == "" {
			return
		}
		if _, ok := c.Services[service]; !ok {
			problems = append(problems, fmt.Sprintf("%s refers to service %q", where, service))
		}
	}
	for _, name := range r.InputOrder {
		in := r.Inputs[name]
		if in.Mount != nil {
			svc, _, ok := strings.Cut(in.Mount.Into, ":")
			if !ok {
				return fmt.Errorf("input %q: mount.into %q is not service:path", name, in.Mount.Into)
			}
			note(fmt.Sprintf("input %q mount.into", name), svc)
		}
		if in.Load != nil {
			note(fmt.Sprintf("input %q load.service", name), in.Load.Service)
		}
	}
	for _, p := range r.Ready {
		note(fmt.Sprintf("ready probe %q", p.Name), p.Service)
	}
	for _, ch := range r.Checks {
		note(fmt.Sprintf("check %q", ch.ID), ch.Service)
	}
	if r.Test != nil {
		for _, s := range append(append([]*recipe.Step{}, r.Test.Seed...), r.Test.Export...) {
			note(fmt.Sprintf("test step %q", s.Name), s.Service)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s, but compose.yaml defines only %s",
		strings.Join(problems, "; "), strings.Join(c.ServiceNames(), ", "))
}

// Validate runs every rule against a recipe's compose.yaml: the schema, and the three
// rules that JSON Schema cannot express.
func Validate(raw []byte, r *recipe.Resolved) error {
	c, err := Parse(raw)
	if err != nil {
		return err
	}
	if err := ValidateSchema(raw); err != nil {
		return err
	}
	if err := CheckPlaceholders(raw, r); err != nil {
		return err
	}
	if err := CheckServiceReferences(c, r.Recipe); err != nil {
		return err
	}
	// Render the file the way a run would, and throw the result away. The schema
	// sees the compose file with its ${RESTORED_*} placeholders intact (ADR-039), so
	// a variable whose value adds YAML to the document passes every check above and
	// only fails minutes later, inside a run. Doing the substitution here means
	// `recipe validate` - which is what CI gates a contributed recipe on, and what a
	// maintainer reads before merging - refuses it. See DECISIONS.md ADR-056.
	if _, err := Render(raw, r.ComposeEnv()); err != nil {
		return err
	}
	return nil
}

func quoteAll(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = fmt.Sprintf("${%s}", n)
	}
	return strings.Join(q, ", ")
}

// Warnings are the --strict findings: not isolation problems, but the things a
// maintainer would otherwise have to ask for in review.
func Warnings(r *recipe.Recipe, composeRaw []byte) []string {
	var w []string
	if len(r.Metadata.Maintainers) == 0 {
		w = append(w, "metadata.maintainers is empty: nobody is named as the contact for this recipe")
	}
	if len(r.Checks) < 2 {
		w = append(w, fmt.Sprintf("the recipe has %d check(s): one check cannot distinguish "+
			"an application that started from a restore that worked", len(r.Checks)))
	}
	if c, err := Parse(composeRaw); err == nil {
		for _, name := range c.ServiceNames() {
			img := c.Services[name].Image
			if img == "" {
				continue
			}
			if !strings.Contains(img, ":") && !strings.Contains(img, "@") {
				w = append(w, fmt.Sprintf("service %q uses image %q with no tag", name, img))
			}
		}
	}
	return w
}

// forbiddenService lists the compose keys that break the run's isolation. Each one is
// also a `not` branch in schema/compose-safety.schema.json; the two must agree, and
// the schema is the contract that CI's independent validator checks.
// forbiddenTopLevel is the same idea as forbiddenService, for the document root.
// `configs` and `secrets` both take a `file:` that the daemon reads from the host,
// which is a host-file read dressed as a compose feature; the schema's
// `additionalProperties: false` at the root rejects them too, and this exists so the
// message names the key.
var forbiddenTopLevel = map[string]string{
	"include": "a recipe is one self-contained file",
	"configs": "a `configs` entry reads a file from the host; a recipe's data comes from the backup",
	"secrets": "a `secrets` entry reads a file from the host; a recipe's data comes from the backup",
}

var forbiddenService = map[string]string{
	"ports":               "restored never publishes a port; checks run from a helper container on the run's internal network",
	"privileged":          "a privileged container is not isolated from the host",
	"network_mode":        "the run gets its own internal network, and nothing else",
	"pid":                 "sharing the host PID namespace is not isolation",
	"ipc":                 "sharing the host IPC namespace is not isolation",
	"userns_mode":         "the user namespace is the host's to decide, not the recipe's",
	"devices":             "a restore drill needs no host device",
	"device_cgroup_rules": "a restore drill needs no host device",
	"cgroup_parent":       "the cgroup is the host's to decide, not the recipe's",
	"build":               "a recipe references published images; it does not build one",
	"extends":             "a recipe is one self-contained file",
	"external_links":      "a run is connected to nothing outside itself",
}

func checkForbiddenKeys(doc any) error {
	root, ok := doc.(map[string]any)
	if !ok {
		return errors.New("compose.yaml: the document is not a mapping")
	}
	for _, key := range sortedNames(root) {
		if why, bad := forbiddenTopLevel[key]; bad {
			return fmt.Errorf("compose.yaml: `%s` is not allowed at the top level: %s", key, why)
		}
	}
	services, ok := root["services"].(map[string]any)
	if !ok {
		return nil
	}
	for _, name := range sortedNames(services) {
		body, ok := services[name].(map[string]any)
		if !ok {
			continue
		}
		for _, key := range sortedNames(body) {
			if why, bad := forbiddenService[key]; bad {
				return fmt.Errorf("compose.yaml: service %q uses `%s`, which restored does not allow: %s",
					name, key, why)
			}
		}
	}
	return nil
}

func sortedNames(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// checkImages reports the two image mistakes the schema catches but cannot explain.
func checkImages(doc any) error {
	root, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	services, ok := root["services"].(map[string]any)
	if !ok {
		return nil
	}
	for _, name := range sortedNames(services) {
		body, ok := services[name].(map[string]any)
		if !ok {
			continue
		}
		image, ok := body["image"].(string)
		if !ok || image == "" {
			continue
		}
		switch {
		case strings.HasSuffix(image, ":latest"):
			return fmt.Errorf("compose.yaml: service %q uses image %q. "+
				"`latest` moves, so a recipe pinned to it stops testing the same thing "+
				"from one week to the next, and a failure cannot be told apart from an "+
				"upstream change. Pin the version you actually run", name, image)
		case !strings.Contains(image, ":") && !strings.Contains(image, "@"):
			return fmt.Errorf("compose.yaml: service %q uses image %q with no tag, "+
				"which means `latest`. Pin the version you actually run", name, image)
		}
	}
	return nil
}
