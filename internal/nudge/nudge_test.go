package nudge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const smallRecipe = `apiVersion: restored/v1
kind: Recipe
metadata:
  name: paperless
  title: Paperless-ngx
vars:
  db_user: paperless
  app_port: 8000
inputs:
  data:
    kind: dir
    default_path: /srv/paperless/media
`

// The recipe is sent as the user wrote it, with the run's overrides folded back in, so
// what is submitted is what actually worked.
func TestFoldOverridesRewritesOnlyWhatTheRunChanged(t *testing.T) {
	out, err := FoldOverrides([]byte(smallRecipe),
		map[string]string{"db_user": "docs"},
		map[string]string{"data": "/mnt/backup/paperless/media"})
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Metadata map[string]string `yaml:"metadata"`
		Vars     map[string]any    `yaml:"vars"`
		Inputs   map[string]struct {
			Kind        string `yaml:"kind"`
			DefaultPath string `yaml:"default_path"`
		} `yaml:"inputs"`
	}
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("the folded recipe is not YAML any more: %v\n%s", err, out)
	}
	if doc.Vars["db_user"] != "docs" {
		t.Errorf("--set was not folded back: %v", doc.Vars["db_user"])
	}
	if doc.Vars["app_port"] != 8000 {
		t.Errorf("a variable the run did not set was disturbed: %v", doc.Vars["app_port"])
	}
	if got := doc.Inputs["data"].DefaultPath; got != "/mnt/backup/paperless/media" {
		t.Errorf("--input was not folded back: %q", got)
	}
	if doc.Metadata["name"] != "paperless" {
		t.Errorf("metadata was rewritten: %v", doc.Metadata)
	}
}

func TestFoldOverridesIsAnIdentityWithNoOverrides(t *testing.T) {
	out, err := FoldOverrides([]byte(smallRecipe), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != smallRecipe {
		t.Errorf("a run with no overrides must submit the file byte for byte:\n%s", out)
	}
}

func TestSilencedReadsOnlyTheOneKey(t *testing.T) {
	dir := t.TempDir()
	write := func(body string) string {
		p := filepath.Join(dir, ConfigName)
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
		if got := Silenced(write(tc.body)); got != tc.want {
			t.Errorf("%s: Silenced() = %v, want %v", tc.name, got, tc.want)
		}
	}
	if Silenced(filepath.Join(dir, "does-not-exist.yaml")) {
		t.Error("a missing configuration file must not silence the invitation")
	}
}

// ADR-066. The invitation prints no URL at all. It used to print a prefilled GitHub
// link, which produced a branch holding recipe.yaml and nothing else - a pull request
// `recipe validate` cannot pass - and which, on a real terminal, was a few thousand
// characters of percent-encoding wrapped across twenty lines. Both halves of that were
// invisible until something ran it on a TTY.
func TestBuildPrintsNoURL(t *testing.T) {
	out := Build(Input{
		Name:  "paperless-ngx",
		YAML:  []byte(smallRecipe),
		Path:  "/home/you/recipes/paperless-ngx/recipe.yaml",
		Title: "Paperless-ngx",
	})

	if strings.Contains(out, "new/main?") {
		t.Error("the invitation still prints a prefilled GitHub link")
	}
	if strings.Contains(out, "%3A") || strings.Contains(out, "%0A") {
		t.Error("the invitation contains percent-encoded text")
	}
	if strings.Contains(out, "one click") {
		t.Error(`the invitation still says "one click"`)
	}
	for _, want := range []string{
		"1. fork",
		"cp -r /home/you/recipes/paperless-ngx recipes/paperless-ngx",
		"restored recipe test ./recipes/paperless-ngx",
		"--no-nudge",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the invitation does not contain %q:\n%s", want, out)
		}
	}
}

// The whole thing has to fit on a screen. It is printed after a PASS, underneath a
// report somebody is reading.
func TestBuildIsShortEnoughToRead(t *testing.T) {
	big := []byte(smallRecipe + strings.Repeat("# padding padding padding\n", 400))
	out := Build(Input{
		Name:  "immich",
		YAML:  big,
		Path:  "/home/you/recipes/immich/recipe.yaml",
		Title: "Immich",
	})
	if lines := strings.Count(out, "\n"); lines > 16 {
		t.Errorf("the invitation is %d lines long; it is printed under a report somebody "+
			"is reading:\n%s", lines, out)
	}
}
