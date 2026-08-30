package nudge

import (
	"net/url"
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

// The link is the whole mechanism: it has to open GitHub's file-creation editor with
// the recipe already in it. If the encoding is wrong the contributor lands on an empty
// editor, which is worse than not having been asked.
func TestBuildMakesAUsablePrefilledLink(t *testing.T) {
	out := Build(Input{
		Name:  "paperless",
		YAML:  []byte(smallRecipe),
		Path:  "/home/you/recipes/paperless/recipe.yaml",
		Title: "Paperless-ngx",
	})
	link := extractLink(t, out)

	u, err := url.Parse(link)
	if err != nil {
		t.Fatalf("the invitation printed something that is not a URL: %v", err)
	}
	if u.Scheme != "https" || u.Host != "github.com" {
		t.Errorf("link points at %s://%s, want https://github.com", u.Scheme, u.Host)
	}
	if u.Path != "/spelingbee/restored/new/main" {
		t.Errorf("link path = %q, want the /new/<branch> route", u.Path)
	}
	q := u.Query()
	if got := q.Get("filename"); got != "recipes/paperless/recipe.yaml" {
		t.Errorf("filename = %q, want recipes/paperless/recipe.yaml", got)
	}
	if got := q.Get("value"); got != smallRecipe {
		t.Errorf("value did not survive the round trip:\n got %q\nwant %q", got, smallRecipe)
	}

	// A recipe carries characters that must not reach GitHub raw: a colon-space in
	// every mapping, a newline on every line, and a slash in every path.
	if strings.Contains(link, "\n") || strings.Contains(link, " ") {
		t.Error("the link contains a raw newline or space")
	}
	if !strings.Contains(out, "Other people running Paperless-ngx would use it") {
		t.Error("the invitation does not say why other people would want this recipe")
	}
	if !strings.Contains(out, "--no-nudge") {
		t.Error("the invitation is not silenceable in the same breath")
	}
}

// Above 6,000 characters the link stops working in a browser, so the invitation
// becomes four lines of instruction instead. Printing a link that does not open is
// worse than printing no link.
func TestBuildFallsBackWhenTheLinkIsTooLong(t *testing.T) {
	big := []byte(smallRecipe + strings.Repeat("# padding padding padding\n", 400))
	out := Build(Input{
		Name:  "immich",
		YAML:  big,
		Path:  "/home/you/recipes/immich/recipe.yaml",
		Title: "Immich",
	})
	if strings.Contains(out, "github.com/spelingbee/restored/new/main") {
		t.Fatal("a link over the length limit was printed anyway")
	}
	for _, want := range []string{
		"cp -r /home/you/recipes/immich recipes/immich",
		"restored recipe test ./recipes/immich",
		"/home/you/recipes/immich",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the fallback does not contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "clipboard?") {
		t.Error("the fallback asks a question it then answers; say it once")
	}
}

func TestBuildStaysUnderTheLimitItAdvertises(t *testing.T) {
	// One byte under the threshold must still be a link, and the threshold is
	// checked after encoding, because percent-encoding YAML costs roughly 1.9x.
	body := []byte(strings.Repeat("a", 2900))
	out := Build(Input{Name: "x", YAML: body, Title: "X", Path: "/p/recipe.yaml"})
	link := extractLink(t, out)
	if len(link) > MaxURL {
		t.Fatalf("printed a link of %d characters, over the %d limit", len(link), MaxURL)
	}
}

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

func extractLink(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if s := strings.TrimSpace(line); strings.HasPrefix(s, "https://") {
			return s
		}
	}
	t.Fatalf("no link in the invitation:\n%s", out)
	return ""
}

// MNT-04. The prefilled link creates a branch holding recipe.yaml and nothing else,
// and the first thing CI does to it is `recipe validate`, which cannot pass without
// compose.yaml. So even when the link fits, the four-step path is the offer and the
// link is presented as the shortcut it is - never as "one click". See ADR-065.
func TestBuildNeverPromisesOneClick(t *testing.T) {
	out := Build(Input{
		Name:  "paperless-ngx",
		YAML:  []byte(smallRecipe),
		Path:  "/home/you/recipes/paperless-ngx/recipe.yaml",
		Title: "Paperless-ngx",
	})

	// The link is still offered: it is a real shortcut for the first file.
	if !strings.Contains(out, "github.com/spelingbee/restored/new/main") {
		t.Fatal("a recipe well under the length limit was not offered the prefilled link")
	}
	if strings.Contains(out, "one click") {
		t.Error(`the nudge still says "one click" for a path that needs two files`)
	}
	// And the path that actually produces a mergeable pull request is there too.
	for _, want := range []string{
		"1. fork",
		"cp -r /home/you/recipes/paperless-ngx recipes/paperless-ngx",
		"restored recipe test ./recipes/paperless-ngx",
		"compose.yaml",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the nudge does not contain %q:\n%s", want, out)
		}
	}
}
