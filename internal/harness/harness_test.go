package harness

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/spelingbee/drillback/internal/recipe"
)

func TestBudgetsAreSharesOfTheTotal(t *testing.T) {
	b := budgets(20 * time.Minute)
	if b.stageA != 5*time.Minute || b.seed != 5*time.Minute ||
		b.export != 3*time.Minute || b.check != 7*time.Minute {
		t.Fatalf("default budgets do not match SPEC.md section 7.4: %+v", b)
	}
	// Lowering the total has to shorten every phase, not starve the last one, or a
	// CI job with a tighter budget would spend it all on the seed and time out in
	// the check that matters.
	half := budgets(10 * time.Minute)
	if half.check != 3*time.Minute+30*time.Second {
		t.Fatalf("check budget did not scale: got %s", half.check)
	}
	if got := budgets(0); got != budgets(DefaultTimeout) {
		t.Fatalf("a zero timeout must mean the default, got %+v", got)
	}
}

func TestWriteEmptyShapes(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "data")
	if err := writeEmpty("dir", dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("dir input: want a directory, got %v %v", info, err)
	}

	db := filepath.Join(root, "nested", "app.db")
	if err := writeEmpty("sqlite", db); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		t.Fatalf("sqlite input: want a zero-length file, got %d bytes", len(body))
	}

	dump := filepath.Join(root, "db.sql")
	if err := writeEmpty("postgres-dump", dump); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != emptyDump {
		t.Fatalf("postgres-dump input: want %q, got %q", emptyDump, body)
	}
}

// stageBRecipe covers the three cases emptyInputs has to tell apart.
const stageBRecipe = `apiVersion: drillback/v1
kind: Recipe
metadata:
  name: shapes
  title: Shapes
  description: A recipe that exercises every input shape.
vars: {}
inputs:
  data:
    kind: dir
    title: Data
    default_path: /srv/shapes/data
    mount:
      env: DRILLBACK_INPUT_data
      into: app:/data
  db:
    kind: sqlite
    title: Database
    default_path: /srv/shapes/data/app.db
    within: data
  dump:
    kind: postgres-dump
    title: Dump
    default_path: /srv/shapes/dump.sql
    load:
      service: db
      database: shapes
      user: shapes
checks:
  - id: one
    title: The first check
    kind: http
    url: http://app:8080/
    expect:
      status: 200
  - id: two
    title: The second check
    kind: http
    url: http://app:8080/health
    expect:
      status: 200
`

func resolveShapes(t *testing.T, inputsDir string) *recipe.Resolved {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "recipe.yaml"), []byte(stageBRecipe), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, err := recipe.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := recipe.Resolve(rec, recipe.Options{InputsDir: inputsDir, RunID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestEmptyInputsLeavesTheApplicationSomethingToCreate(t *testing.T) {
	inputs := filepath.Join(t.TempDir(), "inputs")
	res := resolveShapes(t, inputs)
	if err := emptyInputs(res); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(inputs, "data")); err != nil || !info.IsDir() {
		t.Fatalf("stage B needs the mounted dir input to exist: %v", err)
	}
	// A zero-length SQLite file is a fine empty restore and a hopeless empty start:
	// the application crash-loops instead of running its migrations. See ADR-053.
	if _, err := os.Stat(filepath.Join(inputs, "data", "app.db")); !os.IsNotExist(err) {
		t.Fatalf("stage B must not pre-create a nested database file: %v", err)
	}
	// Nothing mounts the dump; an export step produces it.
	if _, err := os.Stat(filepath.Join(inputs, "dump")); !os.IsNotExist(err) {
		t.Fatalf("stage B must not pre-create an unmounted dump: %v", err)
	}
}

func TestEmptyTreeWritesEveryInputAtItsBackupPath(t *testing.T) {
	root := t.TempDir()
	res := resolveShapes(t, filepath.Join(root, "inputs"))
	tree := filepath.Join(root, "tree")
	if err := emptyTree(tree, res); err != nil {
		t.Fatal(err)
	}
	// Stage A restores, so every input has to be there in its empty shape.
	for _, want := range []string{
		filepath.Join(tree, "srv", "shapes", "data"),
		filepath.Join(tree, "srv", "shapes", "data", "app.db"),
		filepath.Join(tree, "srv", "shapes", "dump.sql"),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("stage A tree is missing %s: %v", want, err)
		}
	}
}

func TestWithExportMountHandlesBothEnvironmentSyntaxes(t *testing.T) {
	in := []byte(`services:
  mapping:
    image: alpine:3.20
    environment:
      EXISTING: "1"
  sequence:
    image: alpine:3.20
    environment:
      - EXISTING=1
    volumes:
      - /host/data:/data
  bare:
    image: alpine:3.20
`)
	out, err := withExportMount(in, "/work/export")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Services map[string]struct {
			Environment any   `yaml:"environment"`
			Volumes     []any `yaml:"volumes"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Services) != 3 {
		t.Fatalf("want 3 services, got %d", len(doc.Services))
	}
	for name, svc := range doc.Services {
		found := false
		for _, v := range svc.Volumes {
			if v == "/work/export:"+exportMount {
				found = true
			}
		}
		if !found {
			t.Errorf("service %q did not get the export mount: %v", name, svc.Volumes)
		}
		switch env := svc.Environment.(type) {
		case map[string]any:
			if env["DRILLBACK_EXPORT"] != exportMount {
				t.Errorf("service %q: mapping environment did not get DRILLBACK_EXPORT: %v", name, env)
			}
		case []any:
			if len(env) == 0 || env[len(env)-1] != "DRILLBACK_EXPORT="+exportMount {
				t.Errorf("service %q: list environment did not get DRILLBACK_EXPORT: %v", name, env)
			}
		default:
			t.Errorf("service %q: unexpected environment type %T", name, svc.Environment)
		}
	}
	// The recipe's own mount must survive untouched.
	if got := doc.Services["sequence"].Volumes; len(got) != 2 || got[0] != "/host/data:/data" {
		t.Errorf("the recipe's own volume was disturbed: %v", got)
	}
}

func TestDurationReadsAsATime(t *testing.T) {
	for _, tc := range []struct {
		ms   int64
		want string
	}{
		{412, "412ms"},
		{5_400, "5.4s"},
		{93_000, "1m33s"},
		{3_723_000, "62m03s"},
	} {
		if got := duration(tc.ms); got != tc.want {
			t.Errorf("duration(%d) = %q, want %q", tc.ms, got, tc.want)
		}
	}
}
