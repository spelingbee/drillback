package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"

	restored "github.com/spelingbee/restored"
)

// Load reads a recipe from a path: a directory containing recipe.yaml, or the
// recipe.yaml file itself.
func Load(path string) (*Recipe, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading recipe %q: %w", path, err)
	}
	file := path
	if info.IsDir() {
		file = filepath.Join(path, "recipe.yaml")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading recipe %q: %w", file, err)
	}
	r, err := parse(raw)
	if err != nil {
		return nil, fmt.Errorf("recipe %q: %w", file, err)
	}
	r.File = file
	r.Dir = filepath.Dir(file)
	return r, nil
}

// LoadBundled reads one of the recipes compiled into the binary.
func LoadBundled(name string) (*Recipe, error) {
	raw, err := restored.Recipes.ReadFile("recipes/" + name + "/recipe.yaml")
	if err != nil {
		return nil, fmt.Errorf("no bundled recipe named %q (bundled: %s)",
			name, strings.Join(BundledNames(), ", "))
	}
	r, err := parse(raw)
	if err != nil {
		return nil, fmt.Errorf("bundled recipe %q: %w", name, err)
	}
	r.Bundled = true
	r.File = "bundled:recipes/" + name + "/recipe.yaml"
	r.Dir = "bundled:recipes/" + name
	return r, nil
}

// LoadAny resolves the --recipe argument: a bundled name, a directory, or a file.
func LoadAny(ref string) (*Recipe, error) {
	if !strings.ContainsAny(ref, `/\.`) {
		return LoadBundled(ref)
	}
	return Load(ref)
}

// BundledNames lists the recipes compiled into the binary.
func BundledNames() []string {
	entries, err := restored.Recipes.ReadDir("recipes")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// ReadFile returns a file from the recipe directory, from disk or from the bundle.
func (r *Recipe) ReadFile(name string) ([]byte, error) {
	if r.Bundled {
		b, err := restored.Recipes.ReadFile("recipes/" + r.Metadata.Name + "/" + name)
		if err != nil {
			return nil, fmt.Errorf("reading %s for bundled recipe %q: %w", name, r.Metadata.Name, err)
		}
		return b, nil
	}
	b, err := os.ReadFile(filepath.Join(r.Dir, name))
	if err != nil {
		return nil, fmt.Errorf("reading %s for recipe %q: %w", name, r.Metadata.Name, err)
	}
	return b, nil
}

func parse(raw []byte) (*Recipe, error) {
	if err := RejectYAMLTags(raw); err != nil {
		return nil, err
	}

	var generic any
	if err := yaml.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	if err := ValidateDocument(generic); err != nil {
		return nil, err
	}

	var r Recipe
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("decoding recipe: %w", err)
	}

	order, err := inputOrder(raw)
	if err != nil {
		return nil, err
	}
	r.InputOrder = order
	r.Raw = raw
	sum := sha256.Sum256(raw)
	r.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return &r, nil
}

// inputOrder recovers the order the inputs were written in, which Go maps do not keep
// and which the report and `recipe show` both display.
func inputOrder(raw []byte) ([]string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, errors.New("empty recipe")
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "inputs" {
			continue
		}
		m := root.Content[i+1]
		names := make([]string, 0, len(m.Content)/2)
		for j := 0; j+1 < len(m.Content); j += 2 {
			names = append(names, m.Content[j].Value)
		}
		return names, nil
	}
	return nil, errors.New("recipe has no inputs")
}

var recipeSchema = mustCompile("schema/recipe.schema.json")

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

// ValidateDocument validates an already-parsed recipe document against
// schema/recipe.schema.json. The document is the file as written, with templates
// unexpanded; see SPEC.md section 3.4.1.
func ValidateDocument(doc any) error {
	norm, err := Normalise(doc)
	if err != nil {
		return err
	}
	if err := recipeSchema.Validate(norm); err != nil {
		return fmt.Errorf("schema: %w", FlattenSchemaError(err))
	}
	return nil
}

// Normalise converts a YAML-decoded document into the plain JSON value model the
// schema validator expects: map[string]any, []any, string, float64, bool, nil.
func Normalise(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			n, err := Normalise(val)
			if err != nil {
				return nil, err
			}
			out[k] = n
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string mapping key %v", k)
			}
			n, err := Normalise(val)
			if err != nil {
				return nil, err
			}
			out[ks] = n
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			n, err := Normalise(val)
			if err != nil {
				return nil, err
			}
			out[i] = n
		}
		return out, nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case uint64:
		return float64(t), nil
	default:
		return v, nil
	}
}

// englishPrinter renders the validator's messages. The library needs a printer and
// panics on a nil one, and this project has exactly one language (ADR-012).
var englishPrinter = message.NewPrinter(language.English)

// FlattenSchemaError turns the validator's tree into the shortest set of concrete leaf
// messages. A contributor needs "inputs.db.load is required", not a tree.
func FlattenSchemaError(err error) error {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	var leaves []string
	seen := map[string]bool{}
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			where := "/"
			if len(e.InstanceLocation) > 0 {
				where = strings.Join(e.InstanceLocation, ".")
			}
			msg := fmt.Sprintf("%s: %s", where, e.ErrorKind.LocalizedString(englishPrinter))
			if !seen[msg] {
				seen[msg] = true
				leaves = append(leaves, msg)
			}
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	const max = 8
	if len(leaves) > max {
		extra := len(leaves) - max
		leaves = leaves[:max]
		leaves = append(leaves, fmt.Sprintf("(and %d more)", extra))
	}
	return errors.New(strings.Join(leaves, "; "))
}

// RejectYAMLTags refuses explicit YAML tags before a document is parsed, so neither a
// recipe nor a compose file can reach the tool through the YAML layer. See SPEC.md
// section 3.5, rule 1.
func RejectYAMLTags(raw []byte) error {
	s := string(raw)
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case inDouble:
			switch c {
			case '\\':
				i++
			case '"':
				inDouble = false
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '#':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case c == '!':
			// A tag can only appear where a value may start.
			if i == 0 || isYAMLValueStart(s[i-1]) {
				line := 1 + strings.Count(s[:i], "\n")
				return fmt.Errorf("line %d: YAML tags are not allowed", line)
			}
		}
	}
	return nil
}

func isYAMLValueStart(prev byte) bool {
	switch prev {
	case ' ', '\t', '\n', '\r', '[', '{', ',', '-':
		return true
	}
	return false
}
