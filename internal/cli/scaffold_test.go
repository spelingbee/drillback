package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spelingbee/drillback/internal/recipe"
	"github.com/spelingbee/drillback/internal/recipe/safety"
)

// A scaffolded recipe has to validate as it comes out. A contributor whose first
// command fails learns that the format is fiddly, which is the opposite of what this
// project needs.
func TestScaffoldedRecipesValidate(t *testing.T) {
	cases := []struct {
		name string
		db   string
		dirs []string
	}{
		{"miniflux", "none", []string{"data"}},
		{"paperless", "postgres-dump", []string{"media", "thumbs"}},
		{"vaultwarden", "sqlite", []string{"data"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files, err := scaffold(tc.name, tc.db, "", tc.dirs)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), tc.name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, body := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			r, err := recipe.Load(dir)
			if err != nil {
				t.Fatalf("the scaffolded recipe does not validate: %v\n%s", err, files["recipe.yaml"])
			}
			composeRaw, err := r.ReadFile("compose.yaml")
			if err != nil {
				t.Fatal(err)
			}
			res, err := recipe.Resolve(r, recipe.Options{InputsDir: "/ws/inputs", RunID: "scaffold"})
			if err != nil {
				t.Fatalf("the scaffolded recipe does not resolve: %v", err)
			}
			if err := safety.Validate(composeRaw, res); err != nil {
				t.Fatalf("the scaffolded compose file breaks a safety rule: %v\n%s", err, composeRaw)
			}

			// The scaffold must not pretend to be finished: it says, in the file, that
			// its checks prove nothing yet.
			if !strings.Contains(files["recipe.yaml"], "data-sensitive") {
				t.Error("the scaffolded recipe does not tell the author to write a data-sensitive check")
			}
		})
	}
}

func TestScaffoldRejectsABadName(t *testing.T) {
	for _, name := range []string{"", "ab", "Not_A_Name", "with space", strings.Repeat("x", 60)} {
		if _, err := scaffold(name, "none", "", nil); err == nil {
			t.Errorf("scaffold(%q) must be refused", name)
		}
	}
}

func TestTitleCase(t *testing.T) {
	cases := map[string]string{
		"gitea":        "Gitea",
		"uptime-kuma":  "Uptime Kuma",
		"paperless-ng": "Paperless Ng",
	}
	for in, want := range cases {
		if got := title(in); got != want {
			t.Errorf("title(%q) = %q, want %q", in, got, want)
		}
	}
}
