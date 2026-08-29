package recipe

import (
	"path/filepath"
	"strings"
	"testing"
)

const twoInputRecipe = `apiVersion: restored/v1
kind: Recipe
metadata:
  name: sample
  title: A sample application
  description: Verifies that a sample backup restores, for the resolution tests.
vars:
  port: 8080
inputs:
  data:
    kind: dir
    title: The data directory
    default_path: /srv/sample/data
    mount:
      env: RESTORED_INPUT_data
      into: app:/data
  db:
    kind: sqlite
    title: The database
    default_path: /srv/sample/data/app.db
    within: data
    load:
      integrity_check: true
checks:
  - id: rows
    title: The database has rows
    kind: sql
    driver: sqlite
    file: "{{ .inputs.db.path }}"
    query: "SELECT count(*) FROM thing;"
    expect:
      scalar_int_min: 1
  - id: answers
    title: The application answers
    kind: http
    url: http://app:{{ .vars.port }}/
    expect:
      status: 200
`

func resolve(t *testing.T, doc string, opts Options) *Resolved {
	t.Helper()
	r, err := parse([]byte(doc))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if opts.InputsDir == "" {
		opts.InputsDir = filepath.Join("/ws", "inputs")
	}
	if opts.RunID == "" {
		opts.RunID = "testrun"
	}
	res, err := Resolve(r, opts)
	if err != nil {
		t.Fatalf("resolving: %v", err)
	}
	return res
}

func TestResolveUsesRecipeDefaults(t *testing.T) {
	res := resolve(t, twoInputRecipe, Options{})

	data, ok := res.Input("data")
	if !ok {
		t.Fatal("no input named data")
	}
	if data.BackupPath != "/srv/sample/data" {
		t.Errorf("data backup path = %q", data.BackupPath)
	}
	if data.Origin != OriginRecipeDefault {
		t.Errorf("data origin = %q, want %q", data.Origin, OriginRecipeDefault)
	}
	if want := filepath.Join("/ws", "inputs", "data"); data.LocalPath != want {
		t.Errorf("data local path = %q, want %q", data.LocalPath, want)
	}

	// An input declared `within` another is located inside the parent's restored
	// tree, so the same bytes are never restored twice.
	db, _ := res.Input("db")
	if want := filepath.Join("/ws", "inputs", "data", "app.db"); db.LocalPath != want {
		t.Errorf("db local path = %q, want %q", db.LocalPath, want)
	}
	if db.Origin != OriginWithin {
		t.Errorf("db origin = %q, want %q", db.Origin, OriginWithin)
	}
}

func TestResolveAppliesOverrides(t *testing.T) {
	res := resolve(t, twoInputRecipe, Options{
		InputPaths: map[string]string{"data": "/mnt/backup/sample"},
		Vars:       map[string]string{"port": "9999"},
	})

	data, _ := res.Input("data")
	if data.BackupPath != "/mnt/backup/sample" {
		t.Errorf("data backup path = %q", data.BackupPath)
	}
	if data.Origin != OriginFlag {
		t.Errorf("data origin = %q, want %q", data.Origin, OriginFlag)
	}
	// The nested input follows its parent, because `within` is a relationship and not
	// a second copy of the path.
	db, _ := res.Input("db")
	if want := filepath.Join("/mnt", "backup", "sample", "app.db"); !strings.HasSuffix(db.LocalPath, "app.db") {
		t.Errorf("db local path = %q, want it to end in app.db (%q)", db.LocalPath, want)
	}

	// --set reaches the templates.
	for _, c := range res.Recipe.Checks {
		if c.ID == "answers" && c.URL != "http://app:9999/" {
			t.Errorf("check url = %q, want the overridden port", c.URL)
		}
	}
}

func TestResolveExpandsTemplatesEverywhere(t *testing.T) {
	res := resolve(t, twoInputRecipe, Options{})
	for _, c := range res.Recipe.Checks {
		if strings.Contains(c.File, "{{") || strings.Contains(c.URL, "{{") {
			t.Errorf("check %q still holds an unexpanded template: file=%q url=%q", c.ID, c.File, c.URL)
		}
	}
	db, _ := res.Input("db")
	for _, c := range res.Recipe.Checks {
		if c.ID == "rows" && c.File != db.LocalPath {
			t.Errorf("sqlite check file = %q, want the resolved input path %q", c.File, db.LocalPath)
		}
	}
}

func TestResolveRejects(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		want string
	}{
		{"an unknown input name", Options{InputPaths: map[string]string{"nope": "/x"}}, "has no input"},
		{"an unknown variable", Options{Vars: map[string]string{"nope": "1"}}, "has no variable"},
		{"a relative override", Options{InputPaths: map[string]string{"data": "relative/path"}}, "not absolute"},
		{"an override escaping upwards", Options{InputPaths: map[string]string{"data": "/srv/../etc/shadow"}}, ".."},
		{"a nested input moved out of its parent", Options{InputPaths: map[string]string{"db": "/elsewhere/app.db"}}, "is not inside"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := parse([]byte(twoInputRecipe))
			if err != nil {
				t.Fatal(err)
			}
			opts := tc.opts
			opts.InputsDir = "/ws/inputs"
			opts.RunID = "testrun"
			if _, err := Resolve(r, opts); err == nil {
				t.Fatal("this resolution must fail")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestResolveRejectsCollidingInputs(t *testing.T) {
	doc := strings.Replace(twoInputRecipe,
		`    default_path: /srv/sample/data/app.db
    within: data`,
		`    default_path: /srv/sample/data`, 1)
	r, err := parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Resolve(r, Options{InputsDir: "/ws/inputs", RunID: "testrun"})
	if err == nil || !strings.Contains(err.Error(), "both resolve to") {
		t.Fatalf("two inputs on the same path must be rejected, got %v", err)
	}
}

func TestComposeEnv(t *testing.T) {
	res := resolve(t, twoInputRecipe, Options{
		TestAssetsDir: filepath.Join("/ws", "test-assets"),
		ExportDir:     filepath.Join("/ws", "export"),
	})
	env := res.ComposeEnv()

	want := map[string]string{
		"RESTORED_VAR_port":    "8080",
		"RESTORED_RUN_ID":      "testrun",
		"RESTORED_INPUT_data":  "/ws/inputs/data",
		"RESTORED_TEST_ASSETS": "/ws/test-assets",
		"RESTORED_EXPORT":      "/ws/export",
	}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("env[%s] = %q, want %q", k, env[k], v)
		}
	}
	// A sqlite input has no mount, so it contributes no placeholder: a compose file
	// that refers to one is a recipe bug and validation says so.
	if _, ok := env["RESTORED_INPUT_db"]; ok {
		t.Error("an input without a mount must not define a placeholder")
	}
}
