package recipe

import (
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// Origin says where an input's path came from, so a report can show whether the user
// or the recipe chose it.
type Origin string

// The three ways an input path is decided.
const (
	OriginRecipeDefault Origin = "recipe_default"
	OriginFlag          Origin = "flag"
	OriginWithin        Origin = "within"
)

// ResolvedInput is one input with its paths decided: where it lives in the backup,
// and where it will live in the run workspace.
type ResolvedInput struct {
	Name       string
	Kind       string
	Title      string
	BackupPath string
	LocalPath  string
	Origin     Origin
	Within     string
	Mount      *Mount
	Load       *LoadSpec
	Required   bool
}

// Resolved is a recipe with variables overridden, inputs located, and every template
// expanded. Everything downstream of RESOLVE works with this, never with the raw
// recipe.
type Resolved struct {
	Recipe    *Recipe
	Vars      map[string]any
	Inputs    []*ResolvedInput
	byName    map[string]*ResolvedInput
	RunID     string
	InputsDir string

	testAssetsDir string
	exportDir     string
}

// Options are the user's overrides for one resolution.
type Options struct {
	// InputPaths overrides an input's path inside the backup, from --input name=path.
	InputPaths map[string]string
	// Vars overrides a recipe variable, from --set key=value.
	Vars map[string]string
	// InputsDir is the workspace directory that will hold the materialised inputs.
	InputsDir string
	// TestAssetsDir and ExportDir back ${RESTORED_TEST_ASSETS} and ${RESTORED_EXPORT}.
	// They are always defined, even for a plain `check` where no harness service
	// starts, because docker compose interpolates the whole file regardless of which
	// profiles are active.
	TestAssetsDir string
	ExportDir     string
	// RunID identifies the run, and appears in the compose project name.
	RunID string
}

// Resolve applies the user's overrides, locates every input, and expands every
// template in the recipe. The returned Resolved owns its own copy of the recipe, so
// the caller's parsed recipe is never mutated.
func Resolve(r *Recipe, opts Options) (*Resolved, error) {
	cp, err := parse(r.Raw)
	if err != nil {
		return nil, err
	}
	cp.Dir, cp.File, cp.Bundled, cp.Digest = r.Dir, r.File, r.Bundled, r.Digest

	vars := make(map[string]any, len(cp.Vars)+len(opts.Vars))
	for k, v := range cp.Vars {
		vars[k] = v
	}
	for k, v := range opts.Vars {
		if _, ok := vars[k]; !ok {
			return nil, fmt.Errorf("--set %s: recipe %q has no variable %q", k, cp.Metadata.Name, k)
		}
		vars[k] = v
	}

	for name := range opts.InputPaths {
		if _, ok := cp.Inputs[name]; !ok {
			return nil, fmt.Errorf("--input %s: recipe %q has no input %q (has: %s)",
				name, cp.Metadata.Name, name, strings.Join(cp.InputOrder, ", "))
		}
	}

	res := &Resolved{
		Recipe:        cp,
		Vars:          vars,
		byName:        map[string]*ResolvedInput{},
		RunID:         opts.RunID,
		InputsDir:     opts.InputsDir,
		testAssetsDir: opts.TestAssetsDir,
		exportDir:     opts.ExportDir,
	}

	// First pass: everything that stands on its own.
	for _, name := range cp.InputOrder {
		in := cp.Inputs[name]
		if in.Within != "" {
			continue
		}
		ri, err := resolveOne(name, in, opts)
		if err != nil {
			return nil, err
		}
		ri.LocalPath = filepath.Join(opts.InputsDir, name)
		res.add(ri)
	}
	// Second pass: inputs that live inside another input are located relative to it,
	// so the same bytes are never restored twice.
	for _, name := range cp.InputOrder {
		in := cp.Inputs[name]
		if in.Within == "" {
			continue
		}
		parent, ok := res.byName[in.Within]
		if !ok {
			return nil, fmt.Errorf("input %q: within: %q is not an input of this recipe", name, in.Within)
		}
		if parent.Within != "" {
			return nil, fmt.Errorf("input %q: within: %q is itself nested; only one level is allowed", name, in.Within)
		}
		ri, err := resolveOne(name, in, opts)
		if err != nil {
			return nil, err
		}
		rel, err := relativeTo(parent.BackupPath, ri.BackupPath)
		if err != nil {
			return nil, fmt.Errorf("input %q: %w", name, err)
		}
		ri.LocalPath = filepath.Join(parent.LocalPath, filepath.FromSlash(rel))
		if ri.Origin == OriginRecipeDefault {
			ri.Origin = OriginWithin
		}
		res.add(ri)
	}

	if err := res.renderRecipe(); err != nil {
		return nil, err
	}
	// The rendered recipe carries the authoritative load block for each input.
	for _, ri := range res.Inputs {
		ri.Load = cp.Inputs[ri.Name].Load
		ri.Mount = cp.Inputs[ri.Name].Mount
	}
	if err := res.checkCollisions(); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *Resolved) add(ri *ResolvedInput) {
	r.Inputs = append(r.Inputs, ri)
	r.byName[ri.Name] = ri
}

// Input returns one resolved input by name.
func (r *Resolved) Input(name string) (*ResolvedInput, bool) {
	ri, ok := r.byName[name]
	return ri, ok
}

func resolveOne(name string, in *Input, opts Options) (*ResolvedInput, error) {
	p := in.DefaultPath
	origin := OriginRecipeDefault
	if override, ok := opts.InputPaths[name]; ok {
		p = override
		origin = OriginFlag
	}
	if err := validBackupPath(p); err != nil {
		return nil, fmt.Errorf("input %q: %w", name, err)
	}
	return &ResolvedInput{
		Name:       name,
		Kind:       in.Kind,
		Title:      in.Title,
		BackupPath: path.Clean(p),
		Origin:     origin,
		Within:     in.Within,
		Required:   in.IsRequired(),
	}, nil
}

func validBackupPath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("path %q is not absolute; paths inside a backup are absolute POSIX paths", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return fmt.Errorf("path %q contains %q", p, "..")
		}
	}
	return nil
}

func relativeTo(parent, child string) (string, error) {
	parent = strings.TrimSuffix(path.Clean(parent), "/")
	child = path.Clean(child)
	if child == parent {
		return ".", nil
	}
	if !strings.HasPrefix(child, parent+"/") {
		return "", fmt.Errorf("path %q is not inside %q, but the recipe says it is", child, parent)
	}
	return strings.TrimPrefix(child, parent+"/"), nil
}

// checkCollisions rejects two independent inputs that resolve to the same path in the
// backup. Restoring the same bytes into two workspace locations is always a mistake,
// and it is the shape of a mis-typed --input.
func (r *Resolved) checkCollisions() error {
	seen := map[string]string{}
	names := make([]string, 0, len(r.Inputs))
	for _, in := range r.Inputs {
		if in.Within != "" {
			continue
		}
		names = append(names, in.Name)
	}
	sort.Strings(names)
	for _, name := range names {
		in := r.byName[name]
		if other, ok := seen[in.BackupPath]; ok {
			return fmt.Errorf("inputs %q and %q both resolve to %s", other, in.Name, in.BackupPath)
		}
		seen[in.BackupPath] = in.Name
	}
	return nil
}

// TemplateContext builds the restricted context this resolution renders against.
func (r *Resolved) TemplateContext() TemplateContext {
	inputs := make(map[string]InputContext, len(r.Inputs))
	for _, in := range r.Inputs {
		inputs[in.Name] = InputContext{Path: in.LocalPath, Kind: in.Kind}
	}
	return TemplateContext{Vars: r.Vars, Inputs: inputs, RunID: r.RunID}
}

// renderRecipe expands every template in the recipe copy. Metadata is prose and is
// deliberately left alone.
func (r *Resolved) renderRecipe() error {
	ctx := r.TemplateContext()
	skip := map[string]bool{"Metadata": true, "Raw": true, "Digest": true, "InputOrder": true}
	return renderValue(reflect.ValueOf(r.Recipe).Elem(), ctx, skip)
}

// ComposeEnv is the environment restored defines for compose.yaml interpolation.
func (r *Resolved) ComposeEnv() map[string]string {
	env := map[string]string{
		"RESTORED_RUN_ID":      r.RunID,
		"RESTORED_TEST_ASSETS": ComposePath(r.testAssetsDir),
		"RESTORED_EXPORT":      ComposePath(r.exportDir),
	}
	for k, v := range r.Vars {
		env["RESTORED_VAR_"+k] = toString(v)
	}
	for _, in := range r.Inputs {
		if in.Mount == nil {
			continue
		}
		env[in.Mount.Env] = ComposePath(in.LocalPath)
	}
	return env
}

// ComposePath renders a host path the way docker compose expects to see it. On
// Windows a backslash path is ambiguous with the volume separator, and compose
// accepts the forward-slash form of the same absolute path.
func ComposePath(p string) string {
	return filepath.ToSlash(p)
}
