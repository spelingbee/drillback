package report

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spelingbee/drillback/internal/check"
	"github.com/spelingbee/drillback/internal/recipe"
	"github.com/spelingbee/drillback/internal/source"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func intp(v int) *int       { return &v }
func strp(v string) *string { return &v }
func boolp(v bool) *bool    { return &v }
func i64p(v int64) *int64   { return &v }
func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// passing is a whole report, built by hand, so that rendering can be tested as the
// pure function of a struct that it is.
func passing() *Report {
	return &Report{
		SchemaVersion: SchemaVersion,
		Tool:          Tool{Name: "drillback", Version: "0.1.0", Commit: "3f9a1c4e"},
		Run: Run{
			ID: "k7m2q9xf", ComposeProject: "drillback-k7m2q9xf",
			StartedAt: "2026-09-14T02:31:08Z", FinishedAt: "2026-09-14T02:32:10Z",
			DurationMS: 62000, WorkspaceRemoved: true,
		},
		Verdict: VerdictPass, ExitCode: 0,
		Recipe: RecipeInfo{Name: "gitea", Title: "Gitea + PostgreSQL", Source: "bundled", Digest: "sha256:0f6d"},
		Source: source.Descriptor{
			Kind: "restic", Repository: "sftp:backup@nas.lan:/srv/restic",
			Snapshot: &source.Snapshot{
				ID: "4a7f1c2e", ShortID: "4a7f1c2e", Time: mustTime("2026-09-13T02:14:07Z"),
				Hostname: "hypervisor", Tags: []string{"gitea"}, SelectedBy: "latest",
			},
		},
		Inputs: []Input{
			{Name: "data", Kind: "dir", BackupPath: "/srv/gitea/data", Bytes: 1932735283, Files: 14203, Origin: "recipe_default"},
			{Name: "db", Kind: "postgres-dump", BackupPath: "/srv/gitea/db.sql", Bytes: 44145868, Files: 1, DetectedFormat: "plain", Origin: "recipe_default"},
		},
		Stages: []Stage{
			{Name: "restore", Status: "ok", DurationMS: 18400, Note: "2 inputs"},
			{Name: "compose", Status: "ok", DurationMS: 6100, Services: []string{"db", "gitea"}, Note: "2 services"},
			{Name: "load db", Status: "ok", DurationMS: 11700, Note: "db: psql, 0 stderr lines"},
			{Name: "ready", Status: "ok", DurationMS: 22900, Note: "postgres accepts connections, gitea answers"},
		},
		Checks: []Check{
			{ID: "web-ui-renders", Title: "The web UI renders the instance home page", Kind: "http",
				Status: "pass", DurationMS: 210,
				Expect:   recipe.Expect{Status: intp(200), BodyMatches: "(?i)<title>[^<]*gitea"},
				Observed: check.Observation{Status: intp(200), BodyBytes: intp(18422), Matched: boolp(true)}},
			{ID: "repos-in-db", Title: "The database contains at least one repository row", Kind: "sql",
				Status: "pass", DurationMS: 40, Query: "SELECT count(*) FROM repository;",
				Expect:   recipe.Expect{ScalarIntMin: intp(1)},
				Observed: check.Observation{Rows: intp(1), Value: "7", Summary: "7"}},
			{ID: "repo-files-on-disk", Title: "At least one bare repository exists on disk", Kind: "file",
				Status: "pass", DurationMS: 120,
				Expect: recipe.Expect{Exists: boolp(true), Glob: "*/*.git/HEAD", GlobMinCount: intp(1)},
				Observed: check.Observation{
					Exists: boolp(true), IsDir: boolp(true), Count: intp(7),
					Summary: "7 matches for */*.git/HEAD"}},
		},
		Summary: Summary{ChecksTotal: 3, ChecksPassed: 3},
	}
}

func unusable() *Report {
	r := passing()
	r.Run.ID = "q4x8b1na"
	r.Run.ComposeProject = "drillback-q4x8b1na"
	r.Run.DurationMS = 58412
	r.Verdict = VerdictUnusable
	r.ExitCode = 1
	r.Inputs[1].Bytes = 90419
	r.Checks[1].Status = "fail"
	r.Checks[1].Observed = check.Observation{
		Error: "ERROR:  relation \"repository\" does not exist\nLINE 1: SELECT count(*) FROM repository;",
	}
	r.Checks[1].Failures = []check.Failure{{
		Expect: "scalar_int_min: 1",
		Got:    "ERROR:  relation \"repository\" does not exist\nLINE 1: SELECT count(*) FROM repository;",
	}}
	r.Summary = Summary{ChecksTotal: 3, ChecksPassed: 2, ChecksFailed: 1}
	r.Hint = &Hint{
		ID:        "postgres/relation-missing",
		MatchedOn: "checks[1].observed.error",
		Title:     "The dump loaded but the application's tables are missing",
		Text: "psql reported no error, yet the table the application needs is not there. " +
			"A dump taken with `pg_dump --schema-only`, from the wrong database, or narrowed " +
			"with `--table`, produces exactly this.",
		Commands: []string{
			"grep -ci 'CREATE TABLE' /srv/gitea/db.sql",
			"pg_restore --list /srv/gitea/db.sql | head -30",
		},
	}
	r.Warnings = []Warning{{
		Code:   "symlink_escaped_workspace",
		Detail: "inputs/data/log/current -> /var/log/gitea (neutralised)",
	}}
	return r
}

func TestGoldenTTY(t *testing.T) {
	cases := []struct {
		name string
		rep  *Report
		opts Options
	}{
		{"pass", passing(), Options{}},
		{"pass-ascii", passing(), Options{ASCII: true}},
		{"unusable", unusable(), Options{}},
		{"unusable-colour", unusable(), Options{Color: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			if err := tc.rep.WriteTTY(&b, tc.opts); err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", tc.name+".txt")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, b.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run `go test ./internal/report -update` to create it)", err)
			}
			if got := b.String(); got != string(want) {
				t.Errorf("rendering changed.\n--- want ---\n%s\n--- got ---\n%s", want, got)
			}
		})
	}
}

// The verdict has to read identically through NO_COLOR, through `| cat`, and in a
// screenshot pasted into an issue. Colour is an enhancement, never the only signal.
func TestVerdictIsReadableWithoutColour(t *testing.T) {
	var plain, coloured bytes.Buffer
	if err := unusable().WriteTTY(&plain, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := unusable().WriteTTY(&coloured, Options{Color: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.String(), "\x1b[") {
		t.Error("the plain rendering contains an escape sequence")
	}
	if !strings.Contains(plain.String(), "RESTORE UNUSABLE") {
		t.Error("the plain rendering does not say RESTORE UNUSABLE")
	}
	stripped := stripANSI(coloured.String())
	if stripped != plain.String() {
		t.Error("the coloured rendering says something different once the colour is removed")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestJSONReportIsStable(t *testing.T) {
	var b bytes.Buffer
	if err := unusable().WriteJSON(&b); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b.Bytes(), &doc); err != nil {
		t.Fatalf("the report is not valid JSON: %v", err)
	}

	// These are the fields automation is expected to depend on. They are frozen for
	// v0.x, so a rename here is a breaking change and this test is the alarm.
	if doc["schema_version"] != float64(SchemaVersion) {
		t.Errorf("schema_version = %v", doc["schema_version"])
	}
	if doc["verdict"] != "RESTORE_UNUSABLE" {
		t.Errorf("verdict = %v", doc["verdict"])
	}
	if doc["exit_code"] != float64(1) {
		t.Errorf("exit_code = %v", doc["exit_code"])
	}
	summary, ok := doc["summary"].(map[string]any)
	if !ok {
		t.Fatal("no summary object")
	}
	for _, k := range []string{"checks_total", "checks_passed", "checks_failed", "checks_skipped"} {
		if _, ok := summary[k]; !ok {
			t.Errorf("summary has no %q", k)
		}
	}
	checks, ok := doc["checks"].([]any)
	if !ok || len(checks) != 3 {
		t.Fatalf("checks = %v", doc["checks"])
	}
	first, _ := checks[0].(map[string]any)
	if first["status"] != "pass" {
		t.Errorf("checks[0].status = %v", first["status"])
	}

	// An observation only carries the fields its kind produced: an empty field in the
	// report is a claim that the check looked and saw nothing.
	observed, _ := first["observed"].(map[string]any)
	if _, ok := observed["rows"]; ok {
		t.Error("an http check reported a row count")
	}
}

func TestHumanUnits(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"}, {999, "999 B"}, {1024, "1.0 KiB"},
		{90419, "88.3 KiB"}, {1932735283, "1.8 GiB"},
	}
	for _, tc := range cases {
		if got := humanBytes(tc.bytes); got != tc.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
	durations := []struct {
		ms   int64
		want string
	}{
		{210, "0.21s"}, {6100, "6.1s"}, {62000, "1m 02s"}, {58412, "58.4s"},
	}
	for _, tc := range durations {
		if got := duration(tc.ms); got != tc.want {
			t.Errorf("duration(%d) = %q, want %q", tc.ms, got, tc.want)
		}
	}
	if got := thousands(14203); got != "14,203" {
		t.Errorf("thousands(14203) = %q", got)
	}
}

var _ = strp
var _ = i64p
