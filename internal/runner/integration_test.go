//go:build integration

// These tests need a real docker daemon. They live behind a build tag so that
// `go test ./...` stays hermetic for somebody who has just cloned the repository, and
// they skip with a reason rather than failing when docker is not there.
//
// There is deliberately no mocked docker API anywhere in this suite. Mocking the thing
// whose real behaviour is the entire risk surface would produce a green suite and a
// broken tool. See SPEC.md section 10.5.
package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spelingbee/restored/internal/compose"
	"github.com/spelingbee/restored/internal/recipe"
	"github.com/spelingbee/restored/internal/report"
)

const fixtureRecipe = "../../testdata/recipes/fixture"

// goodDump is what a working nightly dump looks like.
const goodDump = `--
-- PostgreSQL database dump
--
CREATE TABLE thing (id serial PRIMARY KEY, name text NOT NULL);
INSERT INTO thing (name) VALUES ('one');
INSERT INTO thing (name) VALUES ('two');
`

// emptyDump is what a dump of the wrong database looks like: valid SQL, loads without
// complaint, and carries none of the application's tables.
const emptyDump = `--
-- PostgreSQL database dump
--
SET statement_timeout = 0;
`

func requireDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := compose.Preflight(ctx, false); err != nil {
		t.Skipf("skipping: %v", err)
	}
}

// tree writes the already-restored filesystem the dir source reads.
func tree(t *testing.T, dump string) string {
	t.Helper()
	root := t.TempDir()
	site := filepath.Join(root, "srv", "fixture", "site")
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatal(err)
	}
	page := "<!doctype html><title>fixture</title><p>restored fixture page</p>\n"
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "srv", "fixture", "db.sql"), []byte(dump), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runFixture(t *testing.T, from string, mutate func(*Options)) (*report.Report, *Kept, error) {
	t.Helper()
	rec, err := recipe.Load(fixtureRecipe)
	if err != nil {
		t.Fatal(err)
	}
	opts := Options{
		Recipe:     rec,
		SourceKind: "dir",
		From:       from,
		Timeout:    10 * time.Minute,
		Version:    "0.0.0-test",
	}
	if mutate != nil {
		mutate(&opts)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	return Run(ctx, opts)
}

func TestRunnerPassesOnAGoodBackup(t *testing.T) {
	requireDocker(t)

	rep, kept, err := runFixture(t, tree(t, goodDump), nil)
	if err != nil {
		t.Fatalf("the run failed as a tool error: %v", err)
	}
	if kept != nil {
		t.Errorf("nothing should have been kept: %+v", kept)
	}
	if rep.Verdict != report.VerdictPass {
		t.Fatalf("verdict = %s, exit %d, error %q\nchecks: %+v",
			rep.Verdict, rep.ExitCode, rep.Error, rep.Checks)
	}
	if rep.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", rep.ExitCode)
	}
	if rep.Summary.ChecksTotal != 3 || rep.Summary.ChecksPassed != 3 {
		t.Errorf("summary = %+v, want 3 of 3", rep.Summary)
	}
	if !rep.Run.WorkspaceRemoved {
		t.Error("the workspace was not removed")
	}
	if _, err := os.Stat(rep.Run.Workspace); !os.IsNotExist(err) {
		t.Errorf("the workspace is still on disk: %v", err)
	}

	// The report has to be self-contained: a copy pasted into an issue must say what
	// was restored and what was observed.
	if len(rep.Inputs) != 2 {
		t.Fatalf("inputs = %+v", rep.Inputs)
	}
	for _, in := range rep.Inputs {
		if in.Bytes == 0 {
			t.Errorf("input %q reports 0 bytes", in.Name)
		}
	}
	if rep.Inputs[1].DetectedFormat != "plain" {
		t.Errorf("dump format = %q, want plain", rep.Inputs[1].DetectedFormat)
	}
	for _, c := range rep.Checks {
		if c.Status != "pass" {
			t.Errorf("check %q failed: %+v", c.ID, c.Failures)
		}
	}
	stages := map[string]string{}
	for _, s := range rep.Stages {
		stages[s.Name] = s.Status
	}
	for _, want := range []string{"restore", "compose", "load db", "ready"} {
		if stages[want] != "ok" {
			t.Errorf("stage %q = %q, want ok", want, stages[want])
		}
	}
}

// A dump that loads cleanly and carries none of the application's tables is the
// failure this tool exists to find. It is a verdict about the backup: exit 1, not 2.
func TestRunnerFailsOnADumpOfTheWrongDatabase(t *testing.T) {
	requireDocker(t)

	rep, _, err := runFixture(t, tree(t, emptyDump), nil)
	if err != nil {
		t.Fatalf("this must be a verdict, not a tool error: %v", err)
	}
	if rep.Verdict != report.VerdictUnusable {
		t.Fatalf("verdict = %s, want %s", rep.Verdict, report.VerdictUnusable)
	}
	if rep.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", rep.ExitCode)
	}
	if rep.Summary.ChecksFailed != 1 {
		t.Errorf("summary = %+v, want exactly the SQL check to fail", rep.Summary)
	}

	var sqlCheck *report.Check
	for i := range rep.Checks {
		if rep.Checks[i].ID == "rows-in-db" {
			sqlCheck = &rep.Checks[i]
		}
	}
	if sqlCheck == nil || sqlCheck.Status != "fail" {
		t.Fatalf("the SQL check did not fail: %+v", rep.Checks)
	}
	if !strings.Contains(sqlCheck.Observed.Error, "does not exist") {
		t.Errorf("observed error = %q, want the missing relation", sqlCheck.Observed.Error)
	}
	if rep.Hint == nil {
		t.Fatal("no hint was offered for a missing relation")
	}
	if rep.Hint.ID != "postgres/relation-missing" {
		t.Errorf("hint = %q, want postgres/relation-missing", rep.Hint.ID)
	}
	if len(rep.Logs) == 0 {
		t.Error("no service logs were collected for a failing run")
	}
}

// Teardown is registered before the first resource exists and runs on every exit path.
// Nothing restored created may outlive the process.
func TestTeardownRemovesEverything(t *testing.T) {
	requireDocker(t)

	rep, _, err := runFixture(t, tree(t, goodDump), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"container", "network", "volume"} {
		if left := labelled(t, kind, rep.Run.ID); left != "" {
			t.Errorf("%s objects survived teardown: %q", kind, left)
		}
	}
}

func TestKeepRetainsTheWorkspaceAndSaysHow(t *testing.T) {
	requireDocker(t)

	rep, kept, err := runFixture(t, tree(t, goodDump), func(o *Options) { o.Keep = true })
	if err != nil {
		t.Fatal(err)
	}
	if kept == nil {
		t.Fatal("--keep returned nothing to clean up by hand")
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "compose", "-p", kept.Project,
			"-f", filepath.Join(kept.Workspace, "compose.yaml"),
			"down", "-v", "--remove-orphans").Run()
		_ = os.RemoveAll(kept.Workspace)
	})

	if rep.Run.WorkspaceRemoved {
		t.Error("the report claims the workspace was removed")
	}
	if _, err := os.Stat(kept.Workspace); err != nil {
		t.Errorf("the workspace was not kept: %v", err)
	}
	// The interpolated compose file is the thing a user pokes at afterwards.
	if _, err := os.Stat(filepath.Join(kept.Workspace, "compose.yaml")); err != nil {
		t.Errorf("no compose.yaml in the kept workspace: %v", err)
	}
	if left := labelled(t, "container", rep.Run.ID); left == "" {
		t.Error("--keep tore the containers down anyway")
	}
}

// A missing required input is a tool error, not a verdict: restored could not perform
// the test, and the backup is unproven rather than proven bad.
func TestMissingInputIsAToolError(t *testing.T) {
	requireDocker(t)

	root := tree(t, goodDump)
	if err := os.Remove(filepath.Join(root, "srv", "fixture", "db.sql")); err != nil {
		t.Fatal(err)
	}
	rep, _, err := runFixture(t, root, nil)
	if err == nil {
		t.Fatal("a missing required input must be an error")
	}
	if !strings.Contains(err.Error(), "db") {
		t.Errorf("error %q does not name the input", err)
	}
	if rep.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2", rep.ExitCode)
	}
}

func labelled(t *testing.T, kind, runID string) string {
	t.Helper()
	args := []string{kind, "ls", "-q", "--filter", "label=" + compose.LabelRun + "=" + runID}
	if kind == "container" {
		args = []string{"ps", "-aq", "--filter", "label=" + compose.LabelRun + "=" + runID}
	}
	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		t.Fatalf("docker %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}
