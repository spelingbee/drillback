package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Job is one target resolved against its source and the defaults: everything the
// caller needs to run it, with the config's half of the precedence chain already
// applied. A zero value - a 0 duration, an empty pull - means the config said
// nothing, and the caller falls back to the flag default.
type Job struct {
	Target    string
	RecipeRef string

	SourceKind string
	From       string
	Host       string
	Tags       []string

	Inputs map[string]string
	Set    map[string]string

	// Env is extra KEY=VALUE entries for the source's child processes: the restic
	// password settings and the source's env block, ${NAME} already expanded from
	// the process environment. Never logged.
	Env []string

	Timeout        time.Duration
	RestoreTimeout time.Duration
	ReadyTimeout   time.Duration
	CheckTimeout   time.Duration
	Pull           string
	Workspace      string
}

// TargetNames lists every target in file order.
func (c *Config) TargetNames() []string { return c.TargetOrder }

// EnabledTargets lists the targets --all runs, in file order (ADR-068). A target with
// `enabled: false` is skipped here and still runnable with --target.
func (c *Config) EnabledTargets() []string {
	var out []string
	for _, name := range c.TargetOrder {
		t := c.Targets[name]
		if t.Enabled == nil || *t.Enabled {
			out = append(out, name)
		}
	}
	return out
}

// Job resolves one named target.
func (c *Config) Job(name string) (*Job, error) {
	t, ok := c.Targets[name]
	if !ok {
		if len(c.TargetOrder) == 0 {
			return nil, fmt.Errorf("%s: no target %q (the file defines no targets)", c.Path, name)
		}
		return nil, fmt.Errorf("%s: no target %q (targets: %s)", c.Path, name, strings.Join(c.TargetOrder, ", "))
	}

	sourceName := t.Source
	if sourceName == "" {
		sourceName = c.Defaults.Source
	}
	src := c.Sources[sourceName] // validated at load time

	j := &Job{
		Target:    name,
		RecipeRef: c.recipeRef(t.Recipe),
		Host:      src.Host,
		Tags:      append([]string(nil), t.Tags...),
		Inputs:    copyMap(t.Inputs),
		Set:       copyMap(t.Set),

		Timeout:        firstDuration(t.Timeout, c.Defaults.Timeout),
		RestoreTimeout: firstDuration(t.RestoreTimeout, c.Defaults.RestoreTimeout),
		ReadyTimeout:   firstDuration(t.ReadyTimeout, c.Defaults.ReadyTimeout),
		CheckTimeout:   firstDuration(t.CheckTimeout, c.Defaults.CheckTimeout),
		Pull:           firstString(t.Pull, c.Defaults.Pull),
		Workspace:      c.hostPath(firstString(t.Workspace, c.Defaults.Workspace)),
	}

	switch src.Kind {
	case "restic":
		j.SourceKind = "restic"
		// The repository is left exactly as written: it is a restic backend
		// reference (sftp:, s3:, or a local path), not necessarily a filesystem
		// path, and rewriting it here would corrupt every remote form.
		j.From = src.Repository
		if src.PasswordFile != "" {
			j.Env = append(j.Env, "RESTIC_PASSWORD_FILE="+c.hostPath(src.PasswordFile))
		}
		if len(src.PasswordCommand) > 0 {
			j.Env = append(j.Env, "RESTIC_PASSWORD_COMMAND="+shellJoin(src.PasswordCommand))
		}
		env, err := resolveEnv(sourceName, src.Env)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.Path, err)
		}
		j.Env = append(j.Env, env...)
	case "dir":
		j.SourceKind = "dir"
		j.From = c.hostPath(src.Path)
	}
	return j, nil
}

// recipeRef resolves a target's recipe reference. A bundled name - no separator, no
// dot, the same rule recipe.LoadAny applies - passes through untouched; a relative
// path resolves against the config file's directory like every other host path.
func (c *Config) recipeRef(ref string) string {
	if !strings.ContainsAny(ref, `/\.`) {
		return ref
	}
	return c.hostPath(ref)
}

// hostPath resolves a relative host-filesystem path against the config file's
// directory. The file is discovered by a search order and read from cron, so
// "relative to wherever the process started" points somewhere different on every
// invocation; relative to the file is the reading its author can predict.
func (c *Config) hostPath(p string) string {
	// Rooted counts as absolute here even where filepath.IsAbs says otherwise: on
	// Windows "/etc/restored/nas.pass" has no drive letter, but the person who
	// wrote it meant a fixed place, and silently gluing the config directory in
	// front of it would be worse than leaving it as written.
	if p == "" || filepath.IsAbs(p) || p[0] == '/' || p[0] == '\\' {
		return p
	}
	return filepath.Join(filepath.Dir(c.Path), p)
}

func firstDuration(vals ...Duration) time.Duration {
	for _, v := range vals {
		if v != 0 {
			return v.D()
		}
	}
	return 0
}

func firstString(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func copyMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// shellJoin renders an argv as the single string restic's RESTIC_PASSWORD_COMMAND
// wants, quoting with single quotes the way shell-words splitters read them.
func shellJoin(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		if a != "" && !strings.ContainsAny(a, " \t'\"\\$`") {
			parts[i] = a
			continue
		}
		parts[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(parts, " ")
}
