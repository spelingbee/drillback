// Package nudge builds the one-sentence invitation printed after a recipe that is not
// in the bundled registry has just proved a restore.
//
// restored never opens a browser, never writes to the clipboard, and never sends
// anything anywhere. It prints a URL and stops.
package nudge

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

// Repo is the project this build invites contributions to.
const Repo = "https://github.com/spelingbee/restored"

// MaxURL is the practical ceiling for a prefilled link, checked after encoding.
// Above it the invitation becomes a four-line instruction instead. See SPEC.md 8.3.
const MaxURL = 6000

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

	link := fmt.Sprintf("%s/new/main?filename=%s&value=%s",
		Repo,
		url.QueryEscape("recipes/"+in.Name+"/recipe.yaml"),
		url.QueryEscape(string(in.YAML)))

	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s\n", rule)
	fmt.Fprintf(&b, "  This recipe is not in the bundled registry, and it just proved a restore.\n")
	fmt.Fprintf(&b, "  Other people running %s would use it. ", in.Title)

	if len(link) <= MaxURL {
		fmt.Fprintf(&b, "Adding it is one click:\n\n")
		fmt.Fprintf(&b, "    %s\n\n", link)
	} else {
		dir := strings.TrimSuffix(in.Path, "/recipe.yaml")
		fmt.Fprintf(&b, "It is too large for a\n")
		fmt.Fprintf(&b, "  prefilled link (%.1f KB encoded), so:\n\n", float64(len(link))/1024)
		fmt.Fprintf(&b, "    1. fork  %s\n", Repo)
		fmt.Fprintf(&b, "    2. cp -r %s recipes/%s\n", dir, in.Name)
		fmt.Fprintf(&b, "    3. restored recipe test ./recipes/%s     # this is what CI runs\n", in.Name)
		fmt.Fprintf(&b, "    4. open a PR\n\n")
		fmt.Fprintf(&b, "  restored does not touch your clipboard. The file is at %s.\n\n", in.Path)
	}
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
