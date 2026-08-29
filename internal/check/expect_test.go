package check

import (
	"strings"
	"testing"

	"github.com/spelingbee/restored/internal/recipe"
)

func intp(v int) *int       { return &v }
func strp(v string) *string { return &v }
func boolp(v bool) *bool    { return &v }
func i64p(v int64) *int64   { return &v }

// TestEvaluate walks the expect vocabulary, one row per key, with an observation that
// meets it and one that does not.
func TestEvaluate(t *testing.T) {
	cases := []struct {
		name   string
		kind   string
		expect recipe.Expect
		obs    Observation
		want   bool // want pass
	}{
		{"status matches", "http", recipe.Expect{Status: intp(200)}, Observation{Status: intp(200)}, true},
		{"status differs", "http", recipe.Expect{Status: intp(200)}, Observation{Status: intp(500)}, false},
		{"status missing", "http", recipe.Expect{Status: intp(200)}, Observation{}, false},
		{"status_in matches", "http", recipe.Expect{StatusIn: []int{200, 302}}, Observation{Status: intp(302)}, true},
		{"status_in differs", "http", recipe.Expect{StatusIn: []int{200, 302}}, Observation{Status: intp(404)}, false},

		{"body matches", "http", recipe.Expect{BodyMatches: "(?i)<title>[^<]*gitea"},
			Observation{Body: "<html><title>Gitea: Git with a cup of tea</title>"}, true},
		{"body does not match", "http", recipe.Expect{BodyMatches: "(?i)gitea"},
			Observation{Body: "<html><title>Forgejo</title>"}, false},
		{"body_not_matches is satisfied", "http", recipe.Expect{BodyNotMatches: "(?i)install"},
			Observation{Body: "the dashboard"}, true},
		{"body_not_matches fires", "http", recipe.Expect{BodyNotMatches: "(?i)install"},
			Observation{Body: "redirecting to /install"}, false},

		{"json_path_equals matches", "http",
			recipe.Expect{JSONPath: "$.type", JSONPathEquals: strp("entryPage")},
			Observation{Body: `{"type":"entryPage","entryPage":null}`}, true},
		{"json_path_equals differs", "http",
			recipe.Expect{JSONPath: "$.type", JSONPathEquals: strp("entryPage")},
			Observation{Body: `{"type":"statusPage"}`}, false},
		{"json_path_len_min matches", "http",
			recipe.Expect{JSONPath: "$.data", JSONPathLenMin: intp(1)},
			Observation{Body: `{"data":[{"id":1}]}`}, true},
		{"json_path_len_min differs", "http",
			recipe.Expect{JSONPath: "$.data", JSONPathLenMin: intp(1)},
			Observation{Body: `{"data":[]}`}, false},
		{"json_path_int_min matches", "http",
			recipe.Expect{JSONPath: "$.count", JSONPathIntMin: intp(3)},
			Observation{Body: `{"count":7}`}, true},
		{"json_path on a body that is not JSON", "http",
			recipe.Expect{JSONPath: "$.data", JSONPathLenMin: intp(1)},
			Observation{Body: "<html>an error page</html>"}, false},
		{"json_path pointing nowhere", "http",
			recipe.Expect{JSONPath: "$.nope", JSONPathLenMin: intp(1)},
			Observation{Body: `{"data":[]}`}, false},

		{"exec exits zero by default", "exec", recipe.Expect{StdoutMatches: "ok"},
			Observation{ExitCode: intp(0), Stdout: "ok"}, true},
		{"exec exits non-zero", "exec", recipe.Expect{StdoutMatches: "ok"},
			Observation{ExitCode: intp(1), Stdout: "ok"}, false},
		{"exec exit_code matches", "exec", recipe.Expect{ExitCode: intp(2)},
			Observation{ExitCode: intp(2)}, true},
		{"exec stderr matches", "exec", recipe.Expect{StderrMatches: "warning"},
			Observation{ExitCode: intp(0), Stderr: "warning: nothing to do"}, true},

		{"scalar_equals matches", "sql", recipe.Expect{ScalarEquals: strp("ok")},
			Observation{Value: "ok", Rows: intp(1)}, true},
		{"scalar_equals differs", "sql", recipe.Expect{ScalarEquals: strp("ok")},
			Observation{Value: "malformed", Rows: intp(1)}, false},
		{"scalar_int_min matches", "sql", recipe.Expect{ScalarIntMin: intp(1)},
			Observation{Value: "7", Rows: intp(1)}, true},
		{"scalar_int_min differs", "sql", recipe.Expect{ScalarIntMin: intp(1)},
			Observation{Value: "0", Rows: intp(1)}, false},
		{"scalar_int_min on a non-number", "sql", recipe.Expect{ScalarIntMin: intp(1)},
			Observation{Value: "seven", Rows: intp(1)}, false},
		{"scalar_int_max matches", "sql", recipe.Expect{ScalarIntMax: intp(10)},
			Observation{Value: "7", Rows: intp(1)}, true},
		{"rows_min matches", "sql", recipe.Expect{RowsMin: intp(2)}, Observation{Rows: intp(3)}, true},
		{"rows_min differs", "sql", recipe.Expect{RowsMin: intp(2)}, Observation{Rows: intp(1)}, false},
		{"rows_max differs", "sql", recipe.Expect{RowsMax: intp(2)}, Observation{Rows: intp(3)}, false},
		{"a query that errored", "sql", recipe.Expect{ScalarIntMin: intp(1)},
			Observation{Error: `ERROR:  relation "repository" does not exist`}, false},

		{"exists matches", "file", recipe.Expect{Exists: boolp(true)}, Observation{Exists: boolp(true)}, true},
		{"exists differs", "file", recipe.Expect{Exists: boolp(true)}, Observation{Exists: boolp(false)}, false},
		{"is_dir matches", "file", recipe.Expect{IsDir: boolp(true)}, Observation{IsDir: boolp(true)}, true},
		{"is_dir differs", "file", recipe.Expect{IsDir: boolp(true)}, Observation{IsDir: boolp(false)}, false},
		{"not_empty matches", "file", recipe.Expect{NotEmpty: boolp(true)}, Observation{Entries: intp(3)}, true},
		{"not_empty differs", "file", recipe.Expect{NotEmpty: boolp(true)}, Observation{Entries: intp(0)}, false},
		{"size_min matches", "file", recipe.Expect{SizeMin: i64p(1024)}, Observation{Bytes: i64p(4096)}, true},
		{"size_min differs", "file", recipe.Expect{SizeMin: i64p(1024)}, Observation{Bytes: i64p(12)}, false},
		{"glob_min_count matches", "file",
			recipe.Expect{Glob: "*/*.git/HEAD", GlobMinCount: intp(1)}, Observation{Count: intp(7)}, true},
		{"glob_min_count differs", "file",
			recipe.Expect{Glob: "*/*.git/HEAD", GlobMinCount: intp(1)}, Observation{Count: intp(0)}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := tc.obs
			failures := Evaluate(tc.kind, tc.expect, &obs)
			passed := len(failures) == 0
			if passed != tc.want {
				t.Fatalf("passed = %v, want %v (failures: %v)", passed, tc.want, failures)
			}
			// A failing check must always say both halves, or the user is pushed into
			// --keep, which is the friction this tool exists to remove.
			for _, f := range failures {
				if f.Expect == "" || f.Got == "" {
					t.Errorf("a failure with an empty half: %+v", f)
				}
			}
		})
	}
}

// Every unmet key is reported, not only the first: a report that stops at the first
// failure hides the shape of the problem.
func TestEvaluateReportsEveryUnmetKey(t *testing.T) {
	obs := Observation{Status: intp(500), Body: "<html>error</html>"}
	failures := Evaluate("http", recipe.Expect{
		Status:      intp(200),
		BodyMatches: "(?i)gitea",
	}, &obs)
	if len(failures) != 2 {
		t.Fatalf("got %d failures, want 2: %v", len(failures), failures)
	}
}

func TestJSONPathLookup(t *testing.T) {
	doc := map[string]any{
		"type": "entryPage",
		"data": []any{map[string]any{"id": float64(1), "name": "drill-repo"}},
		"meta": map[string]any{"total": float64(7)},
	}
	ok := []struct {
		expr string
		want string
	}{
		{"$.type", "entryPage"},
		{"$.meta.total", "7"},
		{"$.data[0].name", "drill-repo"},
		{`$["meta"]["total"]`, "7"},
		{"$", "an object with 3 keys"},
	}
	for _, tc := range ok {
		v, err := Lookup(doc, tc.expr)
		if err != nil {
			t.Errorf("Lookup(%q): %v", tc.expr, err)
			continue
		}
		if got := describe(v); got != tc.want {
			t.Errorf("Lookup(%q) = %q, want %q", tc.expr, got, tc.want)
		}
	}

	bad := []struct{ expr, want string }{
		{"$.nope", "no key"},
		{"$.data[9]", "out of range"},
		{"$.type.nested", "not an object"},
		{"$.meta[0]", "not an array"},
		{"$.data[", "unclosed"},
		{"$data", "expected . or ["},
	}
	for _, tc := range bad {
		_, err := Lookup(doc, tc.expr)
		if err == nil {
			t.Errorf("Lookup(%q) must fail", tc.expr)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Lookup(%q) error %q does not mention %q", tc.expr, err, tc.want)
		}
	}
}

// HostPath is what lets a `file` check look at what a service sees, without entering
// a container and without any path leaving the workspace.
func TestHostPath(t *testing.T) {
	e := &Executor{Mounts: []Mount{
		{Service: "gitea", ContainerPath: "/data", HostPath: "/ws/inputs/data"},
		{Service: "kuma", ContainerPath: "/app/data", HostPath: "/ws/inputs/kdata"},
	}}
	cases := []struct{ service, path, want string }{
		{"gitea", "/data", "/ws/inputs/data"},
		{"gitea", "/data/git/repositories", "/ws/inputs/data/git/repositories"},
		{"kuma", "/app/data/kuma.db", "/ws/inputs/kdata/kuma.db"},
	}
	for _, tc := range cases {
		got, err := e.HostPath(tc.service, tc.path)
		if err != nil {
			t.Errorf("HostPath(%q, %q): %v", tc.service, tc.path, err)
			continue
		}
		if slash(got) != tc.want {
			t.Errorf("HostPath(%q, %q) = %q, want %q", tc.service, tc.path, slash(got), tc.want)
		}
	}

	// A path no input is mounted at is an error a recipe author can act on, not a
	// silent read of somewhere else.
	if _, err := e.HostPath("gitea", "/etc/passwd"); err == nil {
		t.Error("a path outside every mount must be refused")
	}
	if _, err := e.HostPath("nosuch", "/data"); err == nil {
		t.Error("a path in a service with no mounts must be refused")
	}
}
