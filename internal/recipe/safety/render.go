package safety

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Render is the only supported way to turn a recipe's compose.yaml into the document
// that is written to the workspace and handed to `docker compose`.
//
// Interpolate on its own is not safe to run and then trust. The safety schema is
// checked against the file *as written*, with the ${DRILLBACK_*} placeholders intact
// (ADR-039), so anything a substituted value adds to the document arrives after the
// only structural check has already passed. A `vars` value of
//
//	"8080\n    privileged: true"
//
// used in an unquoted scalar position adds a key to a service, and the recipe still
// validates. See DECISIONS.md ADR-056.
//
// Render closes that by asserting the invariant the substitution is supposed to have:
// interpolation replaces scalar values and changes nothing else. Not "it does not add
// `privileged`" - it adds no key anywhere, removes none, and turns no scalar into a
// mapping or a list. A deny-list of keys would have to grow with the compose
// specification; this does not.
func Render(raw []byte, env map[string]string) ([]byte, error) {
	rendered, err := Interpolate(raw, env)
	if err != nil {
		return nil, err
	}
	if err := CheckInterpolationShape(raw, rendered); err != nil {
		return nil, err
	}
	// Belt and braces, and it produces the better message: if a value ever does
	// smuggle a key past the shape check, name the key rather than the path.
	var doc any
	if err := yaml.Unmarshal(rendered, &doc); err != nil {
		return nil, fmt.Errorf("compose.yaml: parsing the interpolated file: %w", err)
	}
	if err := checkForbiddenKeys(doc); err != nil {
		return nil, err
	}
	return rendered, nil
}

// CheckInterpolationShape reports whether the interpolated document has the same
// shape as the one it was rendered from: the same mapping keys at every path, the
// same sequence lengths, and a scalar wherever there was a scalar.
//
// Scalar *values* are free to differ - that is the entire point of interpolation.
// Everything else differing means a substituted value was parsed as YAML structure,
// which is the injection.
func CheckInterpolationShape(before, after []byte) error {
	var b, a any
	if err := yaml.Unmarshal(before, &b); err != nil {
		return fmt.Errorf("compose.yaml: parsing YAML: %w", err)
	}
	if err := yaml.Unmarshal(after, &a); err != nil {
		return fmt.Errorf("compose.yaml: parsing the interpolated file: %w", err)
	}
	return sameShape(b, a, "")
}

func sameShape(before, after any, path string) error {
	switch bv := before.(type) {
	case map[string]any:
		av, ok := after.(map[string]any)
		if !ok {
			return shapeErr(path, "a mapping", kindOf(after))
		}
		if added := missing(av, bv); len(added) > 0 {
			return fmt.Errorf("compose.yaml: interpolating a value added %s at %s. "+
				"A recipe variable may only replace a value; it may not add keys to the "+
				"compose file. Quote the value, or remove the line break from it",
				quoteAll(added), where(path))
		}
		if removed := missing(bv, av); len(removed) > 0 {
			return fmt.Errorf("compose.yaml: interpolating a value removed %s at %s. "+
				"A recipe variable may only replace a value", quoteAll(removed), where(path))
		}
		for _, k := range sortedNames(bv) {
			if err := sameShape(bv[k], av[k], join(path, k)); err != nil {
				return err
			}
		}
		return nil
	case []any:
		av, ok := after.([]any)
		if !ok {
			return shapeErr(path, "a list", kindOf(after))
		}
		if len(av) != len(bv) {
			return fmt.Errorf("compose.yaml: interpolating a value changed the list at %s "+
				"from %d item(s) to %d. A recipe variable may only replace a value",
				where(path), len(bv), len(av))
		}
		for i := range bv {
			if err := sameShape(bv[i], av[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	default:
		// A scalar, or nil. Either way the value may change; the kind may not.
		switch after.(type) {
		case map[string]any:
			return shapeErr(path, "a value", "a mapping")
		case []any:
			return shapeErr(path, "a value", "a list")
		}
		return nil
	}
}

func shapeErr(path, want, got string) error {
	return fmt.Errorf("compose.yaml: interpolating a value turned %s into %s at %s. "+
		"A recipe variable may only replace a value; quote it if it contains YAML syntax",
		want, got, where(path))
}

func kindOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return "a mapping"
	case []any:
		return "a list"
	case nil:
		return "nothing"
	default:
		return "a value"
	}
}

func where(path string) string {
	if path == "" {
		return "the top level of the document"
	}
	return path
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// missing returns the keys of `have` that `want` does not have.
func missing(have, want map[string]any) []string {
	var out []string
	for k := range have {
		if _, ok := want[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// singleLine reports whether a substituted value can change the document's shape by
// being pasted into it. It is the cheap half of the defence; CheckInterpolationShape
// is the half that is actually complete.
func singleLine(v string) bool {
	return !strings.ContainsAny(v, "\n\r")
}
