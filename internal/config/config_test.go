package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// loadString parses an inline document as if it lived at path.
func loadString(t *testing.T, path, body string) (*Config, error) {
	t.Helper()
	cfg, err := parse([]byte(body))
	if cfg != nil {
		cfg.Path = path
	}
	return cfg, err
}

func mustLoadSpec(t *testing.T) *Config {
	t.Helper()
	cfg, err := Load(filepath.Join("testdata", "spec.yaml"))
	if err != nil {
		t.Fatalf("the SPEC.md 2.9 example must load: %v", err)
	}
	return cfg
}

func TestSpecExampleLoads(t *testing.T) {
	cfg := mustLoadSpec(t)

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if got, want := len(cfg.Sources), 3; got != want {
		t.Errorf("sources = %d, want %d", got, want)
	}
	if got, want := len(cfg.Targets), 4; got != want {
		t.Errorf("targets = %d, want %d", got, want)
	}
	wantOrder := []string{"gitea", "vaultwarden", "uptime-kuma", "paperless"}
	if !reflect.DeepEqual(cfg.TargetOrder, wantOrder) {
		t.Errorf("TargetOrder = %v, want %v (file order, ADR-068)", cfg.TargetOrder, wantOrder)
	}
	if got := cfg.Defaults.Timeout.D(); got != 15*time.Minute {
		t.Errorf("defaults.timeout = %v, want 15m", got)
	}
	if cfg.Defaults.Nudge == nil || !*cfg.Defaults.Nudge {
		t.Error("defaults.nudge should be true")
	}
	if got := cfg.Targets["uptime-kuma"].CheckTimeout.D(); got != 90*time.Second {
		t.Errorf("uptime-kuma check_timeout = %v, want 90s", got)
	}
	if e := cfg.Targets["paperless"].Enabled; e == nil || *e {
		t.Error("paperless should be enabled: false")
	}
}

func TestEnabledTargetsSkipsDisabledAndKeepsFileOrder(t *testing.T) {
	cfg := mustLoadSpec(t)
	want := []string{"gitea", "vaultwarden", "uptime-kuma"}
	if got := cfg.EnabledTargets(); !reflect.DeepEqual(got, want) {
		t.Errorf("EnabledTargets() = %v, want %v", got, want)
	}
}

func TestJobMergesDefaultsAndTarget(t *testing.T) {
	cfg := mustLoadSpec(t)

	j, err := cfg.Job("gitea")
	if err != nil {
		t.Fatal(err)
	}
	if j.SourceKind != "restic" || j.From != "sftp:backup@nas.lan:/srv/restic" {
		t.Errorf("gitea source = %s %s", j.SourceKind, j.From)
	}
	if j.Host != "hypervisor" {
		t.Errorf("host = %q, want the source's default filter", j.Host)
	}
	if !reflect.DeepEqual(j.Tags, []string{"gitea"}) {
		t.Errorf("tags = %v", j.Tags)
	}
	if j.Timeout != 15*time.Minute || j.CheckTimeout != 60*time.Second {
		t.Errorf("timeouts = %v/%v, want the defaults", j.Timeout, j.CheckTimeout)
	}
	if j.Inputs["db"] != "/srv/gitea/dumps/gitea.sql" {
		t.Errorf("inputs = %v", j.Inputs)
	}
	if want := "RESTIC_PASSWORD_FILE=/etc/drillback/nas.pass"; !contains(j.Env, want) {
		t.Errorf("env %v missing %q", j.Env, want)
	}

	// The refusal below must hold even on a machine that exports real AWS
	// credentials: t.Setenv registers the restore, Unsetenv makes it truly unset.
	for _, name := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}

	// The target block beats the defaults.
	_, err = cfg.Job("uptime-kuma")
	if err == nil {
		t.Fatal("offsite references ${AWS_*}; with the variables unset, Job must refuse loudly")
	}
	if !strings.Contains(err.Error(), "AWS_ACCESS_KEY_ID") {
		t.Errorf("the error should name the unset variable: %v", err)
	}
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	j, err = cfg.Job("uptime-kuma")
	if err != nil {
		t.Fatal(err)
	}
	if j.CheckTimeout != 90*time.Second {
		t.Errorf("check_timeout = %v, want the target's 90s over the defaults' 60s", j.CheckTimeout)
	}
	if j.Timeout != 15*time.Minute {
		t.Errorf("timeout = %v, want the defaults' 15m", j.Timeout)
	}
	if want := "AWS_ACCESS_KEY_ID=AKIATEST"; !contains(j.Env, want) {
		t.Errorf("env %v missing %q (${NAME} resolves from the process environment)", j.Env, want)
	}
	if want := "RESTIC_PASSWORD_COMMAND=pass show restic/offsite"; !contains(j.Env, want) {
		t.Errorf("env %v missing %q", j.Env, want)
	}
}

func TestJobResolvesRelativeHostPathsAgainstTheConfigDir(t *testing.T) {
	dir := t.TempDir()
	body := `
version: 1
sources:
  export:
    kind: dir
    path: ./nightly
  nas:
    kind: restic
    repository: /srv/restic
    password_file: nas.pass
targets:
  bundled:
    recipe: gitea
    source: nas
  local:
    recipe: ./recipes-local/paperless
    source: export
    workspace: work
`
	cfg, err := loadString(t, filepath.Join(dir, FileName), body)
	if err != nil {
		t.Fatal(err)
	}

	j, err := cfg.Job("bundled")
	if err != nil {
		t.Fatal(err)
	}
	if j.RecipeRef != "gitea" {
		t.Errorf("a bundled name must pass through untouched, got %q", j.RecipeRef)
	}
	if want := "RESTIC_PASSWORD_FILE=" + filepath.Join(dir, "nas.pass"); !contains(j.Env, want) {
		t.Errorf("env %v: a relative password_file resolves against the config dir, want %q", j.Env, want)
	}

	j, err = cfg.Job("local")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "recipes-local", "paperless"); j.RecipeRef != want {
		t.Errorf("recipe = %q, want %q", j.RecipeRef, want)
	}
	if want := filepath.Join(dir, "nightly"); j.From != want {
		t.Errorf("dir source = %q, want %q", j.From, want)
	}
	if want := filepath.Join(dir, "work"); j.Workspace != want {
		t.Errorf("workspace = %q, want %q", j.Workspace, want)
	}
}

func TestJobUnknownTargetListsWhatExists(t *testing.T) {
	cfg := mustLoadSpec(t)
	_, err := cfg.Job("nosuch")
	if err == nil || !strings.Contains(err.Error(), "gitea, vaultwarden, uptime-kuma, paperless") {
		t.Errorf("the error should list the targets in file order: %v", err)
	}
}

func TestValidationRefusals(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string // a fragment the error must contain
	}{
		{"missing version", "targets: {}\n", "version: 1"},
		{"future version", "version: 2\n", "version 2"},
		{"unknown key", "version: 1\ntargts: {}\n", "targts"},
		{"unknown target key", "version: 1\nsources:\n  s: {kind: dir, path: /x}\ntargets:\n  t:\n    recipe: gitea\n    source: s\n    enabld: false\n", "enabld"},
		{"source without kind", "version: 1\nsources:\n  s: {repository: /x}\n", "missing kind"},
		{"unknown source kind", "version: 1\nsources:\n  s: {kind: tape}\n", "unknown kind"},
		{"restic without repository", "version: 1\nsources:\n  s: {kind: restic}\n", "needs a repository"},
		{"dir without path", "version: 1\nsources:\n  s: {kind: dir}\n", "needs a path"},
		{"dir with restic fields", "version: 1\nsources:\n  s: {kind: dir, path: /x, host: h}\n", "belongs to kind restic"},
		{"restic with dir fields", "version: 1\nsources:\n  s: {kind: restic, repository: /r, path: /x}\n", "belongs to kind dir"},
		{"target without recipe", "version: 1\nsources:\n  s: {kind: dir, path: /x}\ntargets:\n  t: {source: s}\n", "missing recipe"},
		{"target with unknown source", "version: 1\nsources:\n  s: {kind: dir, path: /x}\ntargets:\n  t: {recipe: gitea, source: nope}\n", `no such source "nope"`},
		{"target with no source anywhere", "version: 1\nsources:\n  s: {kind: dir, path: /x}\ntargets:\n  t: {recipe: gitea}\n", "defaults.source is not set"},
		{"defaults with unknown source", "version: 1\ndefaults: {source: nope}\n", `defaults.source "nope"`},
		{"bad pull", "version: 1\ndefaults: {pull: sometimes}\n", "always, missing or never"},
		{"bare number duration", "version: 1\ndefaults: {timeout: 90}\n", "not a duration"},
		{"negative duration", "version: 1\ndefaults: {timeout: -5m}\n", "budgets nothing"},
		{"zero duration", "version: 1\ndefaults: {check_timeout: 0s}\n", "budgets nothing"},
	}
	for _, tc := range cases {
		_, err := loadString(t, FileName, tc.body)
		if err == nil {
			t.Errorf("%s: accepted, must be refused", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q does not mention %q", tc.name, err, tc.want)
		}
	}
}

func TestDiscoverHonoursTheSearchOrder(t *testing.T) {
	dir := t.TempDir()
	xdg := filepath.Join(dir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Chdir(dir)

	if _, err := Discover(""); err == nil {
		t.Fatal("with no file anywhere, Discover must refuse")
	} else if !strings.Contains(err.Error(), FileName) {
		t.Errorf("the refusal should name what it searched for: %v", err)
	}

	xdgFile := filepath.Join(xdg, "drillback", FileName)
	if err := os.MkdirAll(filepath.Dir(xdgFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xdgFile, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := Discover(""); err != nil || got != xdgFile {
		t.Errorf("Discover = %q, %v; want the XDG file", got, err)
	}

	cwdFile := filepath.Join(dir, FileName)
	if err := os.WriteFile(cwdFile, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := Discover(""); err != nil || got != FileName {
		t.Errorf("Discover = %q, %v; want ./%s to win over XDG", got, err, FileName)
	}

	if _, err := Discover(filepath.Join(dir, "nope.yaml")); err == nil {
		t.Error("--config pointing at a missing file must refuse, not fall back to the search")
	}
}

// Ported from internal/nudge when internal/config took the one-key reader over
// (ADR-045): the behaviour must not change under anyone.
func TestNudgeSilencedReadsOnlyTheOneKey(t *testing.T) {
	dir := t.TempDir()
	write := func(body string) string {
		p := filepath.Join(dir, FileName)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"false silences", "version: 1\ndefaults:\n  nudge: false\n", true},
		{"true does not", "version: 1\ndefaults:\n  nudge: true\n", false},
		{"absent does not", "version: 1\ndefaults:\n  pull: missing\n", false},
		{"an unrelated file does not", "targets:\n  gitea:\n    recipe: gitea\n", false},
		{"a broken file does not", "defaults: [this is not a mapping\n", false},
	}
	for _, tc := range cases {
		if got := NudgeSilenced(write(tc.body)); got != tc.want {
			t.Errorf("%s: NudgeSilenced() = %v, want %v", tc.name, got, tc.want)
		}
	}
	if NudgeSilenced(filepath.Join(dir, "does-not-exist.yaml")) {
		t.Error("a missing configuration file must not silence the invitation")
	}
}

func TestShellJoinQuotesWhatNeedsIt(t *testing.T) {
	cases := []struct {
		argv []string
		want string
	}{
		{[]string{"pass", "show", "restic/offsite"}, "pass show restic/offsite"},
		{[]string{"cat", "/my secrets/pass"}, "cat '/my secrets/pass'"},
		{[]string{"echo", "it's"}, `echo 'it'\''s'`},
	}
	for _, tc := range cases {
		if got := shellJoin(tc.argv); got != tc.want {
			t.Errorf("shellJoin(%q) = %q, want %q", tc.argv, got, tc.want)
		}
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
