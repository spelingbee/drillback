package check

import (
	"fmt"
	"strconv"
	"strings"
)

// Lookup evaluates the small JSONPath subset the expect vocabulary needs: a root $,
// dotted keys, bracketed keys, and array indices. There is deliberately no filter, no
// wildcard, and no recursive descent, because a recipe is data and not a program.
func Lookup(doc any, expr string) (any, error) {
	if expr == "" {
		return doc, nil
	}
	steps, err := parsePath(expr)
	if err != nil {
		return nil, err
	}
	cur := doc
	for i, s := range steps {
		switch {
		case s.index >= 0:
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("%s is not an array", pathSoFar(expr, steps, i))
			}
			if s.index >= len(arr) {
				return nil, fmt.Errorf("%s has %d elements, index %d is out of range",
					pathSoFar(expr, steps, i), len(arr), s.index)
			}
			cur = arr[s.index]
		default:
			obj, ok := cur.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s is not an object", pathSoFar(expr, steps, i))
			}
			v, ok := obj[s.key]
			if !ok {
				return nil, fmt.Errorf("%s has no key %q", pathSoFar(expr, steps, i), s.key)
			}
			cur = v
		}
	}
	return cur, nil
}

type pathStep struct {
	key   string
	index int
}

func pathSoFar(expr string, steps []pathStep, upto int) string {
	if upto == 0 {
		return "$"
	}
	var b strings.Builder
	b.WriteString("$")
	for _, s := range steps[:upto] {
		if s.index >= 0 {
			fmt.Fprintf(&b, "[%d]", s.index)
		} else {
			fmt.Fprintf(&b, ".%s", s.key)
		}
	}
	_ = expr
	return b.String()
}

func parsePath(expr string) ([]pathStep, error) {
	s := strings.TrimSpace(expr)
	s = strings.TrimPrefix(s, "$")
	var steps []pathStep
	for s != "" {
		switch s[0] {
		case '.':
			s = s[1:]
			end := strings.IndexAny(s, ".[")
			if end < 0 {
				end = len(s)
			}
			key := s[:end]
			if key == "" {
				return nil, fmt.Errorf("json_path %q: empty key", expr)
			}
			steps = append(steps, pathStep{key: key, index: -1})
			s = s[end:]
		case '[':
			end := strings.Index(s, "]")
			if end < 0 {
				return nil, fmt.Errorf("json_path %q: unclosed [", expr)
			}
			inner := s[1:end]
			s = s[end+1:]
			if len(inner) >= 2 && (inner[0] == '\'' || inner[0] == '"') {
				steps = append(steps, pathStep{key: inner[1 : len(inner)-1], index: -1})
				continue
			}
			n, err := strconv.Atoi(inner)
			if err != nil {
				return nil, fmt.Errorf("json_path %q: %q is neither an index nor a quoted key", expr, inner)
			}
			steps = append(steps, pathStep{index: n})
		default:
			return nil, fmt.Errorf("json_path %q: expected . or [ at %q", expr, s)
		}
	}
	return steps, nil
}

// describe renders a selected JSON value the way a report should show it.
func describe(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	case []any:
		return fmt.Sprintf("an array of %d", len(t))
	case map[string]any:
		return fmt.Sprintf("an object with %d keys", len(t))
	default:
		return fmt.Sprint(t)
	}
}

// lengthOf is what json_path_len_min measures: elements of an array, characters of a
// string, keys of an object.
func lengthOf(v any) (int, bool) {
	switch t := v.(type) {
	case []any:
		return len(t), true
	case string:
		return len(t), true
	case map[string]any:
		return len(t), true
	default:
		return 0, false
	}
}
