package compose

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Versions is what the runtime reports about itself. An empty field means the
// dependency was not found, which `restored version` prints as "not found" and
// `restored check` treats as a tool error.
type Versions struct {
	Docker  string
	Compose string
	Restic  string
}

// Probe asks docker, docker compose and restic what they are. It never fails: a
// missing dependency is an empty string, so `restored version` stays usable as a
// bug-report command on a machine where nothing is installed.
func Probe(ctx context.Context) Versions {
	return Versions{
		Docker:  firstLine(output(ctx, "docker", "version", "--format", "{{.Server.Version}}")),
		Compose: strings.TrimPrefix(firstLine(output(ctx, "docker", "compose", "version", "--short")), "v"),
		Restic:  resticVersion(firstLine(output(ctx, "restic", "version"))),
	}
}

// Preflight is the RESOLVE-stage dependency check. needRestic is false when the run
// reads an already-restored tree.
func Preflight(ctx context.Context, needRestic bool) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not on PATH: restored needs docker and docker compose v2")
	}
	if out := output(ctx, "docker", "version", "--format", "{{.Server.Version}}"); out == "" {
		return fmt.Errorf("cannot reach the docker daemon: %s",
			firstLine(errOutput(ctx, "docker", "version", "--format", "{{.Server.Version}}")))
	}
	if out := output(ctx, "docker", "compose", "version", "--short"); out == "" {
		return fmt.Errorf("docker compose v2 is not available: `docker compose version` failed")
	}
	if needRestic {
		if _, err := exec.LookPath("restic"); err != nil {
			return fmt.Errorf("restic is not on PATH: --source restic needs the restic binary")
		}
	}
	return nil
}

// probeTimeout bounds a single version query. `docker version` against an
// unreachable daemon does not fail fast: with a docker context pointing at a host
// that is down, or a remote TCP daemon behind a dropped route, the CLI waits on the
// connection. Preflight runs before the run context exists (it has to - it is what
// decides whether there can be a run), so without a deadline of its own a cron job
// with a stale docker context hangs past its --timeout and forever. `restored
// version` is worse, because its whole purpose is to be runnable when the
// environment is broken. See docs/review/architecture.md ARCH-05.
//
// Ten seconds is enormous for a version query and short enough that a hung daemon is
// reported rather than waited on.
const probeTimeout = 10 * time.Second

func output(ctx context.Context, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func errOutput(ctx context.Context, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stderr = &out
	_ = cmd.Run()
	return strings.TrimSpace(out.String())
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// resticVersion pulls the number out of "restic 0.19.1 compiled with go1.26.4 ...".
func resticVersion(s string) string {
	f := strings.Fields(s)
	if len(f) >= 2 && f[0] == "restic" {
		return f[1]
	}
	return s
}
