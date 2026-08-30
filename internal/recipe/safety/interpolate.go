package safety

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Interpolate expands ${NAME} and $NAME in a compose file from env, and turns the $$
// escape into a literal $. An unknown name is an error, never an empty string.
//
// drillback interpolates compose.yaml itself, writes the result into the workspace, and
// runs docker compose against that file. Doing the substitution here is what lets the
// resolved bind sources be checked for containment before a container ever exists.
func Interpolate(raw []byte, env map[string]string) ([]byte, error) {
	s := string(raw)
	var b strings.Builder
	b.Grow(len(s))
	var missing []string

	for i := 0; i < len(s); i++ {
		if s[i] != '$' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '$' {
			// A literal dollar. Re-escape it so compose does not interpolate it
			// again when it reads the file we write.
			b.WriteString("$$")
			i++
			continue
		}
		name, next, ok := readName(s, i+1)
		if !ok {
			b.WriteByte('$')
			continue
		}
		value, found := env[name]
		if !found {
			missing = append(missing, name)
			value = ""
		}
		// A line break in a substituted value is how a scalar becomes a new key in
		// the document that docker compose is about to run. CheckInterpolationShape
		// catches the consequence; this catches the cause, and names the variable.
		if !singleLine(value) {
			return nil, fmt.Errorf("compose.yaml: the value of ${%s} contains a line break, "+
				"which would add lines to the compose file rather than fill in a value. "+
				"Recipe variables are single-line values; use an input for anything larger", name)
		}
		b.WriteString(escapeDollars(value))
		i = next - 1
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("compose.yaml refers to undefined %s", quoteAll(dedupe(missing)))
	}
	return []byte(b.String()), nil
}

func readName(s string, i int) (name string, next int, ok bool) {
	if i >= len(s) {
		return "", i, false
	}
	braced := s[i] == '{'
	if braced {
		i++
	}
	start := i
	for i < len(s) && (isNameByte(s[i]) || (i > start && s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == start {
		return "", i, false
	}
	name = s[start:i]
	if braced {
		if i >= len(s) || s[i] != '}' {
			return "", i, false
		}
		i++
	}
	return name, i, true
}

func isNameByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// escapeDollars keeps a substituted value literal for the compose parser that reads
// the file afterwards.
func escapeDollars(v string) string { return strings.ReplaceAll(v, "$", "$$") }

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// CheckResolvedMounts is the containment half of the volume rule: after
// interpolation, every bind source must be inside the run workspace. The schema
// rejects a bind that is not a ${DRILLBACK_*} placeholder; this rejects a placeholder
// that resolved somewhere it should not.
func CheckResolvedMounts(resolved []byte, workspace string) error {
	var doc struct {
		Services map[string]struct {
			Volumes []any `yaml:"volumes"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(resolved, &doc); err != nil {
		return fmt.Errorf("compose.yaml: parsing the interpolated file: %w", err)
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	for _, svc := range sortedKeys(doc.Services) {
		for _, v := range doc.Services[svc].Volumes {
			s, ok := v.(string)
			if !ok {
				continue // long syntax; the schema already restricts it to volume/tmpfs
			}
			source, _, ok := cutMount(s)
			if !ok || !isHostPath(source) {
				continue
			}
			abs, err := filepath.Abs(filepath.FromSlash(source))
			if err != nil {
				return err
			}
			if !within(root, abs) {
				return fmt.Errorf("service %q mounts %s, which is outside the run workspace %s",
					svc, source, root)
			}
		}
	}
	return nil
}

// cutMount splits a compose short-syntax volume into its source and the rest, taking
// a Windows drive letter into account.
func cutMount(s string) (source, rest string, ok bool) {
	idx := strings.Index(s, ":")
	if idx == 1 && len(s) > 2 && (s[2] == '/' || s[2] == '\\') {
		// A drive letter, so the real separator is the next colon.
		next := strings.Index(s[2:], ":")
		if next < 0 {
			return "", "", false
		}
		idx = 2 + next
	}
	if idx <= 0 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

// isHostPath distinguishes a bind source from a named volume.
func isHostPath(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") {
		return true
	}
	return len(s) > 2 && s[1] == ':' && (s[2] == '/' || s[2] == '\\')
}

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
