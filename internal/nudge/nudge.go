// Package nudge builds the one-sentence invitation printed after a recipe that is not
// in the bundled registry has just proved a restore.
//
// restored never opens a browser, never writes to the clipboard, and never sends
// anything anywhere. It prints four lines and stops - not a URL: see ADR-066 for why
// the prefilled GitHub link was removed.
package nudge

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Repo is the project this build invites contributions to.
const Repo = "https://github.com/spelingbee/restored"

// Input is everything the invitation needs.
type Input struct {
	Name string
	// YAML is the recipe as the user wrote it, with the run's overrides folded back
	// in, so what is submitted is what actually worked.
	YAML []byte
	// Path is the recipe.yaml on disk, named by the fallback text.
	Path string
	// Title is the application's name, for the "other people running X" sentence.
	Title string
	Width int
}

// Build renders the invitation.
func Build(in Input) string {
	width := in.Width
	if width <= 0 {
		width = 74
	}
	rule := strings.Repeat("─", width)

	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s\n", rule)
	fmt.Fprintf(&b, "  This recipe is not in the bundled registry, and it just proved a restore.\n")
	fmt.Fprintf(&b, "  Other people running %s would use it. ", in.Title)

	// No prefilled GitHub link. Two reasons, and the second one is the one that
	// settles it.
	//
	// It produced a branch containing recipe.yaml and nothing else, and `recipe
	// validate` cannot pass without compose.yaml - so the highest-volume acquisition
	// surface in the project sent people at a guaranteed red X (ADR-065).
	//
	// And it was unreadable. A recipe percent-encodes to a few thousand characters,
	// and a few thousand characters of %0A and %3A wrapped across twenty lines of
	// somebody's terminal is not a shortcut; it is the tool shouting. That was
	// invisible for three sessions because nothing had ever run this on a real TTY -
	// every test and every reviewer captured it through a pipe, where the nudge does
	// not fire at all. A screenshot from a recorded terminal is what found it.
	// See DECISIONS.md ADR-066.
	dir := strings.TrimSuffix(in.Path, "/recipe.yaml")
	fmt.Fprintf(&b, "Adding it is a fork and a\n  four-line pull request:\n\n")
	fmt.Fprintf(&b, "    1. fork  %s\n", Repo)
	fmt.Fprintf(&b, "    2. cp -r %s recipes/%s\n", dir, in.Name)
	fmt.Fprintf(&b, "    3. restored recipe test ./recipes/%s     # this is what CI runs\n", in.Name)
	fmt.Fprintf(&b, "    4. open a PR\n\n")
	fmt.Fprintf(&b, "  restored does not touch your clipboard. Your recipe is at %s.\n\n", dir)
	fmt.Fprintf(&b, "  (silence this with --no-nudge, or `nudge: false` in restored.yaml)\n")
	fmt.Fprintf(&b, "  %s\n", rule)
	return b.String()
}

// FoldOverrides writes the run's --set and --input values back into the recipe, so the
// recipe that is offered for submission is the one that actually worked.
//
// metadata.maintainers is left exactly as it is: restored does not guess the
// contributor's GitHub handle and does not read git config for it.
func FoldOverrides(raw []byte, vars map[string]string, inputPaths map[string]string) ([]byte, error) {
	if len(vars) == 0 && len(inputPaths) == 0 {
		return raw, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("re-reading the recipe: %w", err)
	}
	if len(doc.Content) == 0 {
		return raw, nil
	}
	root := doc.Content[0]

	if v := mappingValue(root, "vars"); v != nil {
		for k, val := range vars {
			setScalar(v, k, val)
		}
	}
	if inputs := mappingValue(root, "inputs"); inputs != nil {
		for name, p := range inputPaths {
			if in := mappingValue(inputs, name); in != nil {
				setScalar(in, "default_path", p)
			}
		}
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("rewriting the recipe: %w", err)
	}
	return out, nil
}

func mappingValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func setScalar(m *yaml.Node, key, value string) {
	if m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value != key {
			continue
		}
		m.Content[i+1].Kind = yaml.ScalarNode
		m.Content[i+1].Tag = "!!str"
		m.Content[i+1].Value = value
		m.Content[i+1].Style = 0
		return
	}
}
