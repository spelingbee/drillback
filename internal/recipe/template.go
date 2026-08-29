package recipe

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

// TemplateContext is the restricted context a recipe string is rendered against:
// .vars, .inputs.<name>.path, .inputs.<name>.kind and .run.id. Nothing else is
// reachable, and an unknown key is an error rather than an empty string.
type TemplateContext struct {
	Vars   map[string]any
	Inputs map[string]InputContext
	RunID  string
}

// InputContext is what a template may see about one input.
type InputContext struct {
	Path string
	Kind string
}

func (c TemplateContext) data() map[string]any {
	inputs := make(map[string]any, len(c.Inputs))
	for name, in := range c.Inputs {
		inputs[name] = map[string]any{"path": in.Path, "kind": in.Kind}
	}
	vars := make(map[string]any, len(c.Vars))
	for k, v := range c.Vars {
		vars[k] = v
	}
	return map[string]any{
		"vars":   vars,
		"inputs": inputs,
		"run":    map[string]any{"id": c.RunID},
	}
}

var funcs = template.FuncMap{
	"quote": func(v any) string { return strconv.Quote(toString(v)) },
	"default": func(def, v any) any {
		if isEmpty(v) {
			return def
		}
		return v
	},
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	default:
		return false
	}
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

// Render expands one recipe string.
func (c TemplateContext) Render(s string) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	t, err := template.New("recipe").Funcs(funcs).Option("missingkey=error").Parse(s)
	if err != nil {
		return "", fmt.Errorf("template %q: %w", s, err)
	}
	var b strings.Builder
	if err := t.Execute(&b, c.data()); err != nil {
		return "", fmt.Errorf("template %q: %w", s, err)
	}
	return b.String(), nil
}

// renderValue walks a value and expands every string it reaches. Rendering every
// string field uniformly is what keeps a new recipe field from silently missing out
// on templating.
func renderValue(v reflect.Value, c TemplateContext, skip map[string]bool) error {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return renderValue(v.Elem(), c, skip)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !f.IsExported() || skip[f.Name] {
				continue
			}
			if err := renderValue(v.Field(i), c, skip); err != nil {
				return fmt.Errorf("%s: %w", f.Name, err)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := renderValue(v.Index(i), c, skip); err != nil {
				return err
			}
		}
	case reflect.Map:
		// Map values are not addressable, so anything that is not reached through a
		// pointer is rendered into a copy and written back.
		for _, k := range v.MapKeys() {
			val := v.MapIndex(k)
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
			switch val.Kind() {
			case reflect.Invalid:
				continue
			case reflect.Pointer:
				if err := renderValue(val, c, skip); err != nil {
					return fmt.Errorf("%v: %w", k, err)
				}
			case reflect.String:
				out, err := c.Render(val.String())
				if err != nil {
					return fmt.Errorf("%v: %w", k, err)
				}
				v.SetMapIndex(k, reflect.ValueOf(out))
			default:
				cp := reflect.New(val.Type()).Elem()
				cp.Set(val)
				if err := renderValue(cp, c, skip); err != nil {
					return fmt.Errorf("%v: %w", k, err)
				}
				v.SetMapIndex(k, cp)
			}
		}
	case reflect.String:
		if !v.CanSet() {
			return nil
		}
		out, err := c.Render(v.String())
		if err != nil {
			return err
		}
		v.SetString(out)
	}
	return nil
}
