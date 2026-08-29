package recipe

import (
	"strings"
	"testing"
)

// minimalRecipe is the smallest document the schema accepts. Every negative case below
// is this document with exactly one thing wrong, so a failure names the constraint.
const minimalRecipe = `apiVersion: restored/v1
kind: Recipe
metadata:
  name: sample
  title: A sample application
  description: Verifies that a sample backup restores, for the schema tests.
inputs:
  data:
    kind: dir
    title: The data directory
    default_path: /srv/sample/data
    mount:
      env: RESTORED_INPUT_data
      into: app:/data
checks:
  - id: app-answers
    title: The application answers
    kind: http
    url: http://app:8080/
    expect:
      status: 200
`

func TestMinimalRecipeIsValid(t *testing.T) {
	r, err := parse([]byte(minimalRecipe))
	if err != nil {
		t.Fatalf("the minimal recipe must validate: %v", err)
	}
	if got := r.Metadata.Name; got != "sample" {
		t.Errorf("metadata.name = %q, want sample", got)
	}
	if got := r.InputOrder; len(got) != 1 || got[0] != "data" {
		t.Errorf("input order = %v, want [data]", got)
	}
	if !r.Inputs["data"].IsRequired() {
		t.Error("an input with no `required` key must be required")
	}
	if !strings.HasPrefix(r.Digest, "sha256:") {
		t.Errorf("digest = %q, want a sha256 digest", r.Digest)
	}
}

func TestSchemaRejects(t *testing.T) {
	cases := []struct {
		name string
		// edit turns the minimal recipe into the invalid one.
		old, new string
		want     string
	}{
		{"wrong apiVersion", "apiVersion: restored/v1", "apiVersion: restored/v2", "apiVersion"},
		{"wrong kind", "kind: Recipe", "kind: Cookbook", "kind"},
		{"name with an underscore", "name: sample", "name: not_a_name", "metadata.name"},
		{"name too short", "name: sample", "name: ab", "metadata.name"},
		{"description too short", "description: Verifies that a sample backup restores, for the schema tests.", "description: too short", "metadata.description"},
		{"maintainer without an at sign", "  description: Verifies", "  maintainers: [\"nobody\"]\n  description: Verifies", "maintainers"},
		{"unknown input kind", "    kind: dir", "    kind: tarball", "inputs.data.kind"},
		{"relative default_path", "default_path: /srv/sample/data", "default_path: srv/sample/data", "default_path"},
		{"default_path escaping upwards", "default_path: /srv/sample/data", "default_path: /srv/../etc", "default_path"},
		{"dir input without a mount", "    mount:\n      env: RESTORED_INPUT_data\n      into: app:/data", "", "mount"},
		{"mount env not namespaced", "env: RESTORED_INPUT_data", "env: DATA_DIR", "env"},
		{"mount into without a service", "into: app:/data", "into: /data", "into"},
		{"check id with capitals", "id: app-answers", "id: App-Answers", "checks.0.id"},
		{"unknown check kind", "    kind: http\n    url: http://app:8080/", "    kind: telepathy", "checks.0"},
		{"http check to loopback", "url: http://app:8080/", "url: http://127.0.0.1:8080/", "url"},
		{"http check to the host gateway", "url: http://app:8080/", "url: http://host.docker.internal:8080/", "url"},
		{"unknown expect key", "      status: 200", "      status_code: 200", "expect"},
		{"json_path_equals without json_path", "      status: 200", "      json_path_equals: \"x\"", "expect"},
		{"empty expect", "    expect:\n      status: 200", "    expect: {}", "expect"},
		{"no checks at all", "checks:\n  - id: app-answers", "checks: []\nunused:\n  - id: app-answers", "checks"},
		{"unknown top-level key", "kind: Recipe", "kind: Recipe\nsurprise: true", "/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := strings.Replace(minimalRecipe, tc.old, tc.new, 1)
			if doc == minimalRecipe {
				t.Fatalf("the edit did not apply; %q is not in the minimal recipe", tc.old)
			}
			_, err := parse([]byte(doc))
			if err == nil {
				t.Fatalf("this document must be rejected:\n%s", doc)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestSchemaAcceptsTheOptionalShapes(t *testing.T) {
	cases := []struct{ name, old, new string }{
		{"a postgres-dump input", "checks:", `  db:
    kind: postgres-dump
    title: The database dump
    default_path: /srv/sample/db.sql
    load:
      service: db
      database: sample
      user: sample
      timeout: 5m
checks:`},
		{"a sqlite input inside another", "checks:", `  db:
    kind: sqlite
    title: The database
    default_path: /srv/sample/data/app.db
    within: data
    load:
      integrity_check: true
checks:`},
		{"a templated port", "url: http://app:8080/", "url: http://app:{{ .vars.port }}/"},
		{"a ready probe", "checks:", `ready:
  - name: the app answers
    kind: http
    url: http://app:8080/
    expect_status: 200
    timeout: 90s
    interval: 2s
checks:`},
		{"an exec check", "    kind: http\n    url: http://app:8080/\n    expect:\n      status: 200",
			"    kind: exec\n    service: app\n    command: [\"sh\", \"-c\", \"test -s /data/x\"]\n    expect:\n      exit_code: 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := strings.Replace(minimalRecipe, tc.old, tc.new, 1)
			if _, err := parse([]byte(doc)); err != nil {
				t.Fatalf("this document must be accepted: %v\n%s", err, doc)
			}
		})
	}
}

func TestRejectYAMLTags(t *testing.T) {
	rejected := []string{
		"a: !!binary aGk=",
		"a:\n  - !!python/object:os.system\n",
		"a: [!!str 1]",
		"!!map\na: 1",
	}
	for _, doc := range rejected {
		if err := RejectYAMLTags([]byte(doc)); err == nil {
			t.Errorf("must be rejected: %q", doc)
		}
	}
	accepted := []string{
		minimalRecipe,
		"a: \"it works!!\"\n",
		"a: 'no!!'\n",
		"# a comment with !! in it\na: 1\n",
		"a: hello!world\n",
	}
	for _, doc := range accepted {
		if err := RejectYAMLTags([]byte(doc)); err != nil {
			t.Errorf("must be accepted: %q: %v", doc, err)
		}
	}
}

func TestBundledRecipesAreValid(t *testing.T) {
	names := BundledNames()
	if len(names) == 0 {
		t.Fatal("no recipes are bundled into the binary")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			r, err := LoadBundled(name)
			if err != nil {
				t.Fatalf("bundled recipe does not load: %v", err)
			}
			if r.Metadata.Name != name {
				t.Errorf("recipe in recipes/%s is named %q", name, r.Metadata.Name)
			}
			if len(r.Checks) < 2 {
				t.Errorf("a bundled recipe needs more than %d check", len(r.Checks))
			}
			if _, err := r.ReadFile("compose.yaml"); err != nil {
				t.Errorf("no compose.yaml: %v", err)
			}
		})
	}
}

func TestTemplateContext(t *testing.T) {
	ctx := TemplateContext{
		Vars:   map[string]any{"port": 3000, "name": "gitea", "blank": ""},
		Inputs: map[string]InputContext{"db": {Path: "/ws/inputs/db", Kind: "sqlite"}},
		RunID:  "abc123",
	}
	ok := []struct{ in, want string }{
		{"http://app:{{ .vars.port }}/", "http://app:3000/"},
		{"{{ .inputs.db.path }}", "/ws/inputs/db"},
		{"{{ .inputs.db.kind }}", "sqlite"},
		{"restored-{{ .run.id }}", "restored-abc123"},
		{"{{ printf \"%s-%d\" .vars.name .vars.port }}", "gitea-3000"},
		{"{{ .vars.blank | default \"fallback\" }}", "fallback"},
		{"{{ .vars.name | default \"fallback\" }}", "gitea"},
		{"{{ quote .vars.name }}", `"gitea"`},
		{"nothing to expand", "nothing to expand"},
	}
	for _, tc := range ok {
		got, err := ctx.Render(tc.in)
		if err != nil {
			t.Errorf("Render(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Render(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// An unknown field or function is an error, never an empty string: a template that
	// silently expands to nothing is how a volume mount becomes "/".
	bad := []string{
		"{{ .vars.nosuchvar }}",
		"{{ .inputs.nosuchinput.path }}",
		"{{ .secrets.password }}",
		"{{ exec \"rm\" }}",
		"{{ .vars.port ",
	}
	for _, in := range bad {
		if got, err := ctx.Render(in); err == nil {
			t.Errorf("Render(%q) = %q, want an error", in, got)
		}
	}
}
