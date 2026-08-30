package safety

import (
	"strings"
	"testing"

	"github.com/spelingbee/drillback/internal/recipe"
)

// safeCompose is the smallest compose file that satisfies every isolation rule. Each
// negative case below is this file with exactly one dangerous construct added, so a
// failure names the rule rather than the fixture.
const safeCompose = `services:
  app:
    image: example/app:1.0.0
    volumes:
      - ${DRILLBACK_INPUT_data}:/data
    networks: [drillback]

networks:
  drillback:
    internal: true
`

const safeRecipe = `apiVersion: drillback/v1
kind: Recipe
metadata:
  name: sample
  title: A sample application
  description: Verifies that a sample backup restores, for the safety tests.
inputs:
  data:
    kind: dir
    title: The data directory
    default_path: /srv/sample/data
    mount:
      env: DRILLBACK_INPUT_data
      into: app:/data
checks:
  - id: app-answers
    title: The application answers
    kind: http
    url: http://app:8080/
    expect:
      status: 200
`

func resolved(t *testing.T, doc string) *recipe.Resolved {
	t.Helper()
	r, err := recipe.Load(writeRecipe(t, doc))
	if err != nil {
		t.Fatalf("loading the recipe: %v", err)
	}
	res, err := recipe.Resolve(r, recipe.Options{InputsDir: "/ws/inputs", RunID: "testrun"})
	if err != nil {
		t.Fatalf("resolving the recipe: %v", err)
	}
	return res
}

func writeRecipe(t *testing.T, doc string) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/recipe.yaml"
	if err := writeFile(path, doc); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSafeComposeIsAccepted(t *testing.T) {
	if err := Validate([]byte(safeCompose), resolved(t, safeRecipe)); err != nil {
		t.Fatalf("the safe compose file must be accepted: %v", err)
	}
}

func TestComposeSafetyRejects(t *testing.T) {
	cases := []struct{ name, insert, want string }{
		{"privileged", "    privileged: true\n", "privileged"},
		{"published ports", "    ports:\n      - \"8080:8080\"\n", "ports"},
		{"host networking", "    network_mode: host\n", "network_mode"},
		{"the host PID namespace", "    pid: host\n", "pid"},
		{"the host IPC namespace", "    ipc: host\n", "ipc"},
		{"a user namespace override", "    userns_mode: host\n", "userns_mode"},
		{"devices", "    devices:\n      - /dev/sda:/dev/sda\n", "devices"},
		{"a build context", "    build: .\n", "build"},
		{"extends", "    extends:\n      service: other\n", "extends"},
		{"external links", "    external_links:\n      - other\n", "external_links"},
		{"a cgroup parent", "    cgroup_parent: /custom\n", "cgroup_parent"},
		{"SYS_ADMIN", "    cap_add: [SYS_ADMIN]\n", "cap_add"},
		{"NET_ADMIN", "    cap_add: [NET_ADMIN]\n", "cap_add"},
		{"an unconfined seccomp profile", "    security_opt: [\"seccomp:unconfined\"]\n", "security_opt"},
		{"an unconfined apparmor profile", "    security_opt: [\"apparmor=unconfined\"]\n", "security_opt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := strings.Replace(safeCompose, "    networks: [drillback]\n",
				tc.insert+"    networks: [drillback]\n", 1)
			err := ValidateSchema([]byte(doc))
			if err == nil {
				t.Fatalf("this compose file must be rejected:\n%s", doc)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestComposeSafetyRejectsOtherShapes(t *testing.T) {
	cases := []struct{ name, doc, want string }{
		{
			"a host bind mount",
			strings.Replace(safeCompose, "${DRILLBACK_INPUT_data}:/data", "/etc:/data", 1),
			"volumes",
		},
		{
			"a long-syntax bind mount",
			strings.Replace(safeCompose,
				"    volumes:\n      - ${DRILLBACK_INPUT_data}:/data\n",
				"    volumes:\n      - type: bind\n        source: /etc\n        target: /data\n", 1),
			"volumes",
		},
		{
			"a network that is not internal",
			strings.Replace(safeCompose, "    internal: true", "    internal: false", 1),
			"internal",
		},
		{
			"an external network",
			strings.Replace(safeCompose, "    internal: true", "    internal: true\n    external: true", 1),
			"external",
		},
		{
			"an unpinned image",
			strings.Replace(safeCompose, "example/app:1.0.0", "example/app:latest", 1),
			"image",
		},
		{
			"an include directive",
			safeCompose + "include:\n  - other.yaml\n",
			"include",
		},
		{
			"a service with no network",
			strings.Replace(safeCompose, "    networks: [drillback]\n", "", 1),
			"networks",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSchema([]byte(tc.doc))
			if err == nil {
				t.Fatalf("this compose file must be rejected:\n%s", tc.doc)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// The three rules that JSON Schema cannot express.

func TestYAMLTagsAreRejected(t *testing.T) {
	doc := strings.Replace(safeCompose, "image: example/app:1.0.0", "image: !!str example/app:1.0.0", 1)
	if _, err := Parse([]byte(doc)); err == nil {
		t.Fatal("a YAML tag in compose.yaml must be rejected")
	}
}

func TestUndefinedPlaceholdersAreRejected(t *testing.T) {
	res := resolved(t, safeRecipe)
	doc := strings.Replace(safeCompose, "${DRILLBACK_INPUT_data}", "${HOME}", 1)
	err := CheckPlaceholders([]byte(doc), res)
	if err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Fatalf("an undefined placeholder must be rejected, got %v", err)
	}
	if err := CheckPlaceholders([]byte(safeCompose), res); err != nil {
		t.Fatalf("a defined placeholder must be accepted: %v", err)
	}
}

func TestUnknownServiceReferencesAreRejected(t *testing.T) {
	doc := strings.Replace(safeRecipe, "into: app:/data", "into: aap:/data", 1)
	r, err := recipe.Load(writeRecipe(t, doc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse([]byte(safeCompose))
	if err != nil {
		t.Fatal(err)
	}
	err = CheckServiceReferences(c, r)
	if err == nil || !strings.Contains(err.Error(), "aap") {
		t.Fatalf("a typo in a service name must be caught at validate time, got %v", err)
	}
}

func TestInterpolate(t *testing.T) {
	env := map[string]string{"DRILLBACK_INPUT_data": "/ws/inputs/data", "DRILLBACK_VAR_port": "8080"}
	cases := []struct{ in, want string }{
		{"a: ${DRILLBACK_INPUT_data}", "a: /ws/inputs/data"},
		{"a: $DRILLBACK_VAR_port", "a: 8080"},
		{"a: $$literal", "a: $$literal"},
		{"a: no placeholders", "a: no placeholders"},
	}
	for _, tc := range cases {
		got, err := Interpolate([]byte(tc.in), env)
		if err != nil {
			t.Errorf("Interpolate(%q): %v", tc.in, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("Interpolate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if _, err := Interpolate([]byte("a: ${NOPE}"), env); err == nil {
		t.Error("an undefined name must be an error, never an empty string")
	}
}

func TestCheckResolvedMounts(t *testing.T) {
	inside := "services:\n  app:\n    volumes:\n      - /ws/run/inputs/data:/data\n"
	outside := "services:\n  app:\n    volumes:\n      - /etc:/data\n"
	named := "services:\n  app:\n    volumes:\n      - db-data:/var/lib/postgresql/data\n"

	if err := CheckResolvedMounts([]byte(inside), "/ws/run"); err != nil {
		t.Errorf("a mount inside the workspace must be accepted: %v", err)
	}
	if err := CheckResolvedMounts([]byte(named), "/ws/run"); err != nil {
		t.Errorf("a named volume must be accepted: %v", err)
	}
	if err := CheckResolvedMounts([]byte(outside), "/ws/run"); err == nil {
		t.Error("a mount outside the workspace must be rejected")
	}
}

func TestWarnings(t *testing.T) {
	r, err := recipe.Load(writeRecipe(t, safeRecipe))
	if err != nil {
		t.Fatal(err)
	}
	got := Warnings(r, []byte(safeCompose))
	if len(got) != 2 {
		t.Fatalf("want a warning for the missing maintainer and one for the single check, got %v", got)
	}
	joined := strings.Join(got, "; ")
	for _, want := range []string{"maintainers", "check"} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings %q do not mention %q", joined, want)
		}
	}
}
