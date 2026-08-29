//go:build integration

package runner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The bundled recipes are exercised through the demo scripts rather than through a Go
// re-implementation of them.
//
// Each script stands up the application the way its users run it, puts real data in
// it, backs that data up with restic, destroys the stack, and then runs `restored
// check`. That whole sequence is the thing worth testing, and having one copy of it
// means the scripts a contributor runs and the scripts CI runs cannot drift apart.
func TestBundledRecipesRoundTrip(t *testing.T) {
	requireDocker(t)
	requireRestic(t)

	root := repoRoot(t)
	bin := buildBinary(t, root)

	cases := []struct {
		name   string
		script string
		want   int
	}{
		{"gitea", "scripts/demo.sh", 0},
		{"gitea with a dump of the wrong database", "scripts/demo-broken.sh", 1},
		{"uptime-kuma", "scripts/demo-kuma.sh", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("sh", tc.script)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "RESTORED_BIN="+bin)
			out, err := cmd.CombinedOutput()

			got := 0
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				got = ee.ExitCode()
			} else if err != nil {
				t.Fatalf("running %s: %v\n%s", tc.script, err, out)
			}
			if got != tc.want {
				t.Fatalf("%s exited %d, want %d\n%s", tc.script, got, tc.want, out)
			}
			if !strings.Contains(string(out), "restored check") {
				t.Errorf("%s never reached the check stage\n%s", tc.script, out)
			}
		})
	}
}

func requireRestic(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("restic"); err != nil {
		t.Skip("skipping: restic is not on PATH")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("skipping: no POSIX shell to run the demo scripts with")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// buildBinary builds the binary the demo scripts drive, so the test exercises the
// working tree rather than whatever happens to be in bin/.
func buildBinary(t *testing.T, root string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "restored")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/restored")
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the binary: %v\n%s", err, b)
	}
	return out
}
