package report_test

import (
	"os/exec"
	"strings"
	"testing"
)

// SPEC.md 13.1: "internal/report is a pure function of its input struct. It does no
// I/O beyond writing to a supplied io.Writer, and it never reaches back into check or
// compose."
//
// That was written down, repeated verbatim in the package comment, and false: report
// imported check, and therefore compose (which shells out to docker) and sqlite
// (which links a SQL driver). It was found by reading the import graph, so the import
// graph is what this test reads. See DECISIONS.md ADR-062 and
// docs/review/architecture.md ARCH-03.
func TestReportDependsOnNoExecutionPackage(t *testing.T) {
	go_, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain on PATH, so the import graph cannot be read")
	}
	out, err := exec.Command(go_, "list", "-deps", "github.com/spelingbee/drillback/internal/report").Output()
	if err != nil {
		t.Skipf("go list failed, so the import graph cannot be read: %v", err)
	}

	// Anything that shells out, opens a socket, or links a database driver. The rule
	// is about what report is allowed to be *made of*, not about what it prints.
	forbidden := []string{
		"internal/check",
		"internal/compose",
		"internal/sqlite",
		"internal/runner",
		"internal/harness",
		"internal/loader",
		"internal/probe",
		"internal/workspace",
		"internal/source/restic",
		"internal/source/dir",
		"internal/cli",
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		for _, bad := range forbidden {
			if dep == "github.com/spelingbee/drillback/"+bad {
				t.Errorf("internal/report depends on %s.\n"+
					"SPEC.md 13.1 says it never reaches back into an execution package: the\n"+
					"report's JSON is a public contract, and a wire type defined inside a\n"+
					"package that runs things is one rename away from a silent breaking\n"+
					"change. Put the shared type in internal/observe, which imports nothing.", bad)
			}
		}
	}
}
