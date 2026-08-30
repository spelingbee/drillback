# Architecture review — restored

Date 2026-08-30, commit `d5c2f6c` (clean tree, branch `main`).
Inspected: `SPEC.md` (13.1 in particular), `DECISIONS.md` (ADR-001..055), `PROGRESS.md`,
and every non-test `.go` file under `internal/`, `cmd/`, `tools/`, plus `schema/` and
`docs/hints.yaml`.
Ran: `gofmt -l .` (clean), `go vet ./...` (clean), `golangci-lint run` (0 issues),
`go test ./... -cover`, `go list -deps`, `go list -f '{{join .Imports}}'` for the graph.
Did NOT run: anything that starts a container (`recipe test`, `-tags integration`,
`scripts/demo*.sh`) or `-race` (no C compiler on this host). Docker-path claims below are
read from the code and are labelled as such.

## Summary

| Severity | Count |
|---|---|
| P0 | 1 |
| P1 | 4 |
| P2 | 7 |
| P3 | 4 |

The design gets the hard part right. Isolation is a structural property, not a habit:
the compose safety schema runs before interpolation, `CheckResolvedMounts` runs after it,
and `internal/workspace` really does hold the containment predicate that everything else
asks. Teardown is registered before the first resource exists and uses
`context.WithoutCancel`, which is the correct idiom and is rarer than it should be. The
round-trip harness genuinely re-enters `runner.Run` for its last step rather than
reimplementing it, so stage B cannot pass while the real path is broken — that is the
load-bearing decision of the whole project and it is honoured in code.

Where it will hurt is at the two seams the roadmap depends on. First, `internal/source`
is not a seam at all: it is a bag of structs, and `restic` is special-cased through
`runner.materialise`, through `compose.Preflight(ctx, needRestic bool)`, and through the
`--source` string in `cli`. ADR-004 asserts the interface "exists in v0.1 with only two
implementations"; it does not exist, and borg and kopia will each be a patch to the
lifecycle package rather than a new file. Second, the 1-vs-2 exit split — the contract
the product is sold on — is drawn by *stage*, not by *cause*, so `restored` blames the
user's backup for its own timeout under the default flags. That is the one finding I
would block a release on.

The rest is the ordinary cost of a fast second half: two lifecycle implementations
(`runner` and `harness/stageb`) that share helpers by copy, no test seam at the docker
boundary so five packages sit at 0%, and errors that are all strings, which is exactly
the thing a notifier will need and not have.

## Findings

### ARCH-01 (P0) A run-level timeout is reported as "your backup is unusable"

**Where:** `internal/runner/runner.go:93`, `internal/runner/runner.go:257-267`,
`internal/runner/runner.go:336`, against `internal/cli/cli.go:118-122` and
`internal/cli/check.go:71-74`.

**What:** `runCtx` carries the whole-run `--timeout`. Every stage from LOAD DUMPS onward
routes its error through `fail()`, which unconditionally sets
`rep.Verdict = report.VerdictUnusable` and `rep.ExitCode = 1`. A cancelled `runCtx` is
indistinguishable from a real failure at that point: `probe.Run` writes
`res.Error = ctx.Err().Error()` (`internal/probe/probe.go:53-55`), the ready stage is
marked failed, and `fail("ready", ...)` renders **RESTORE UNUSABLE** with exit 1. The
exit-code split is drawn by *which stage* failed, never by *why*.

`restored --help` (`internal/cli/cli.go:121`) states the opposite in the shipped binary:

```
  2   tool or runtime error — docker missing, restic failed, recipe invalid, timeout
      before any check could run
```

The defaults make this reachable rather than theoretical. `--timeout 15m`,
`--restore-timeout 10m`, `--ready-timeout 5m`: the restore budget plus the ready budget
already consume the entire run budget, leaving nothing for COMPOSE UP, LOAD DUMPS or the
checks. SPEC.md 4.2's own per-state budgets sum to roughly 27 minutes against a 15-minute
default.

**Scenario:** A user with a 40 GB Nextcloud snapshot runs
`restored check --recipe nextcloud --source restic --from /mnt/backups`. The restore takes
9 minutes (inside its own 10-minute budget, so it succeeds). Compose up takes 2 minutes,
the Postgres dump loads in 3. `runCtx` expires 1 minute into the ready probes. The report
says `RESTORE UNUSABLE`, the process exits 1, and the cron job that wraps it pages
somebody at 03:00 about a backup that is in fact fine. Nothing in the TTY output or the
JSON report contains the word "timeout".

**Proposed fix:** Make the cause, not the stage, decide the code. In `fail()`, test
`errors.Is(runCtx.Err(), context.DeadlineExceeded)` (and `context.Canceled` for SIGINT)
before setting the verdict; on a deadline set `VerdictError`/exit 2 with the message
"restored ran out of its --timeout budget in stage <name>", which is what `_ = stage` at
`runner.go:265` was evidently meant for. Separately, either raise the default `--timeout`
above the sum of the sub-budgets or derive the sub-budgets from it the way
`harness.budgets()` already does (`internal/harness/harness.go:133-139`) — that function
is the right pattern and the runner should use it.

---

### ARCH-02 (P1) `internal/source` is not an interface; restic is special-cased through the lifecycle

**Where:** `internal/source/source.go:1-33` (three structs, no interface, no method),
`internal/runner/runner.go:554-587`, `internal/compose/env.go:33`.

**What:** `runner.materialise` switches on `o.SourceKind` and calls the restic package's
four free functions directly (`ListSnapshots`, `Select`, `Restore`, `Locate`), then
carries restic-specific behaviour past the switch: `runner.go:612-622` uses
`os.Rename` "because the restore directory is the workspace's own" only when
`o.SourceKind == "restic"`, and `runner.go:646` removes `ws.RestoreDir()` which only the
restic path ever populates. The dependency check is
`compose.Preflight(ctx context.Context, needRestic bool)` — a boolean naming one specific
source, in the package that drives docker.

ADR-004 records the opposite as a consequence: "borg and kopia land in v0.2 behind the
same `source` interface, which is why that interface exists in v0.1 with only two
implementations." I am not re-litigating ADR-004; I am reporting that its stated
consequence is not true of the code, which is the condition CLAUDE.md asks to be flagged.

**Scenario:** Adding borg in v0.2 means editing: `internal/runner/runner.go` (a third
`case` plus its own materialise-and-locate branch and its own rename/copy rule),
`internal/compose/env.go` (`needRestic` becomes meaningless), `internal/cli/check.go:64`
(the `--source` help string), `internal/source/source.go` (`Descriptor.Snapshot` is
restic-shaped: `ShortID`, `Tags`, `Paths`), and `internal/harness/stageb.go:241`
(`withResticEnv`). That is five packages for what the roadmap describes as "behind the
same interface". Two sources with a `switch` is fine; four sources with a `switch` in the
state machine is where the lifecycle package stops being about the lifecycle.

**Proposed fix:** Define in `internal/source`:

```go
type Source interface {
    Preflight(ctx context.Context) error
    Describe() Descriptor
    Materialise(ctx context.Context, reqs []Request, into Dir) ([]Warning, error)
}
```

where `Dir` is a small interface satisfied by `*workspace.Workspace` (it already has
`RestoreDir`, `Contains`, `Sanitise`, `Relax`). Move the rename-vs-copy decision and the
`RestoreDir` cleanup into the restic implementation, since only it knows the restore
directory is disposable. `runner.materialise` then becomes `src.Materialise(...)` and
`Preflight` loses its bool. Do this before borg, not with it.

---

### ARCH-03 (P1) `internal/report` imports `internal/check`, violating SPEC 13.1

**Where:** `internal/report/report.go:10` (`import ".../internal/check"`),
`internal/report/report.go:114` and `:117`.

**What:** SPEC.md 13.1 says: "`internal/report` is a pure function of its input struct. It
does no I/O beyond writing to a supplied `io.Writer`, and it never reaches back into
`check` or `compose`." The file's own package comment repeats it verbatim. The import
graph says otherwise:

```
$ go list -deps ./internal/report | grep spelingbee
github.com/spelingbee/restored/internal/compose
github.com/spelingbee/restored
github.com/spelingbee/restored/internal/recipe
github.com/spelingbee/restored/internal/sqlite
github.com/spelingbee/restored/internal/check
github.com/spelingbee/restored/internal/source
github.com/spelingbee/restored/internal/report
```

The I/O half of the rule holds — `grep -n '\bos\.\|filepath\.' internal/report/*.go`
returns nothing outside tests. The import half does not: `report` pulls in `check`, and
therefore `compose` (which shells out to docker) and `sqlite` (which links
`modernc.org/sqlite`).

The concrete harm is not the compile time. `report.Check.Observed` is typed
`check.Observation`, so the **public JSON schema of the report is defined by a struct that
lives in an execution package**. `report.SchemaVersion` promises "within a major version
fields are only added, never removed or retyped" (`internal/report/report.go:16-18`), but
nothing stops a contributor renaming a field on `Observation` for an internal reason.

**Scenario:** Someone implementing the `mysql-dump` input kind in v0.2 renames
`Observation.Rows` to `Observation.RowCount` because "rows" is ambiguous next to MySQL's
affected-rows. `go build` passes, `go test ./...` passes if the golden files are
regenerated with `-update`, and every downstream JSON consumer of `checks[].observed.rows`
breaks silently at schema_version 1.

**Proposed fix:** Declare the wire type in `report` (`report.Observation`, with the JSON
tags) and give `check` a `func (o Observation) Report() report.Observation` — or, if the
dependency direction must stay, move `Observation` and `Failure` into a leaf package
(`internal/observe`) that both import. Then the golden-file tests really are testing a
pure function, and re-typing a report field becomes a change to `report`, where the
stability contract is written down.

---

### ARCH-04 (P1) Four of the 17 hint rules can never fire, and a tool error prints no report at all

**Where:** `internal/runner/runner.go:263` and `:362` (the only two `attachHint` call
sites), `internal/cli/check.go:149-153`.

**What:** `attachHint` is reached only from `fail()` and from the checks-failed branch —
that is, only on exit-1 paths. Every exit-2 path returns before it:
`runner.go:185` (restore failed), `runner.go:226` (compose up failed), `runner.go:89`
(preflight failed, before a report exists at all). And in `cli`, the TTY report is
rendered only `if runErr == nil`, so on a tool error the user sees exactly one line:
`restored: <error>`.

Cross-referencing `docs/hints.yaml`, these rules are unreachable during `restored check`:

- `docker/daemon-unreachable` — only `compose.Preflight` produces that text, and it
  returns at `runner.go:90` before the report exists.
- `compose/image-pull-failed` — only `cli.Up` produces it; `runner.go:226` returns the
  error without a hint.
- `restore/path-not-in-snapshot` — the message is built at `runner.go:604-605` and
  returned at `runner.go:185` without a hint.
- `workspace/no-space` — `workspace.New` fails at `runner.go:97`, before the deferred
  block and before any hint could attach.

ADR-011 sells the hint catalog as "the easiest useful contribution to restored". Four of
the seventeen rules a contributor could read as examples are dead code.

**Scenario:** A user's `RESTIC_PASSWORD_FILE` points at a file they rotated. `restic
snapshots` fails, `materialise` returns, and the user gets
`restored: restic snapshots: exit status 1: Fatal: wrong password or no key found`. No
stages table, no workspace path, no hint, and — because `--report` still writes JSON at
`check.go:160` while the TTY does not — a JSON consumer sees a report the human never
saw. A contributor who then adds a `restic/wrong-password` rule to `hints.yaml` will find
it never fires and have no idea why.

**Proposed fix:** Call `attachHint` once, from a single place: move it into the deferred
block in `runner.Run` (or into a small `finalise(rep, res, err)` helper) so that every
exit path — including the ones that return a `VerdictError` report — gets a hint match.
In `cli/check.go`, render the TTY report whenever `rep.Run.ID != ""`, and print `runErr`
after it rather than instead of it. Then add a `restic/wrong-password` rule, which is the
single most common exit-2 cause the catalog is missing.

---

### ARCH-05 (P1) The docker preflight probe has no deadline and no timeout flag reaches it

**Where:** `internal/compose/env.go:33` (`Preflight`), `:52` (`output`), `:62`
(`errOutput`), `:23` (`Probe`); called from `internal/runner/runner.go:89` and
`internal/cli/version.go:21`.

**What:** `runner.Run` calls `compose.Preflight(ctx, ...)` **before** it builds `runCtx`
with `o.Timeout` (`runner.go:89` vs `:93`). The `ctx` it passes is the process context
from `interruptContext()`, which has a cancel but no deadline. `output()` runs
`exec.CommandContext(ctx, "docker", "version", ...)` and blocks on `cmd.Run()`.
`docker version` against an unreachable daemon does not fail fast: with a `DOCKER_HOST`
pointing at an SSH context whose host is down, or a remote TCP daemon behind a dropped
route, the CLI waits on the connection. `restored version` has the same shape and is
worse, because `Probe` runs three such commands and the command's whole purpose is to be
runnable when the environment is broken.

**Scenario:** A user with a stale `docker context use remote-nas` runs
`restored check --recipe gitea --timeout 5m` from a cron job. The job does not exit after
5 minutes; it hangs on `docker version` until the TCP stack gives up, or forever. The
`--timeout` flag they set is not yet in scope. Next night's cron overlaps. Nothing in the
workspace or the compose labels exists yet, so there is no orphan to find — just an
accumulating pile of `restored` processes.

**Proposed fix:** Give `Preflight` and `Probe` their own bounded contexts —
`context.WithTimeout(ctx, 10*time.Second)` per command is generous for a version query —
and move the `runCtx` construction in `runner.Run` above the `Preflight` call so the
user's `--timeout` is the outer bound from the first syscall. Neither change needs a new
flag.

---

### ARCH-06 (P2) restic's command lines and stderr never reach the run's debug log

**Where:** `internal/runner/runner.go:108-116` (the `debug` multi-writer),
`internal/runner/runner.go:557` (`Debug: o.Debug`).

**What:** The runner builds `debug` as `io.MultiWriter(o.Debug, logFile)` where `logFile`
is `<workspace>/logs/debug.log`, and hands that to `compose.Client`. It then hands the
**raw** `o.Debug` to `resticsource.Options`. `o.Debug` is nil unless the user passed
`--log-level debug` (`internal/cli/check.go:109-111`), so on a default run restic's
argv and stderr go nowhere at all, while every docker invocation is recorded.

**Scenario:** A restore fails halfway through a 30 GB snapshot. The user reruns with
`--keep` to inspect, opens `<workspace>/logs/debug.log`, and finds the docker preflight
and nothing else — the restic command that actually failed, and its stderr, were written
to a nil writer. They cannot even see which `--include` patterns were passed, which is the
single most useful fact when a required input is "not found in the backup".

**Proposed fix:** One line: pass `debug` instead of `o.Debug` at `runner.go:557`. The
`restic.Options.Debug` doc comment already promises "receives restic's stderr and the
command lines. Never its environment", and `restic.run` (`internal/source/restic/restic.go:51-57`)
honours that, so there is no secret-leak risk in routing it to the workspace log.

---

### ARCH-07 (P2) Two lifecycle implementations that share code by copy

**Where:** `internal/runner/runner.go:443-456` vs `internal/harness/stageb.go:531-544`
(byte-identical `mountsOf` / `harnessMounts`); `runner.go:508-527` vs `stageb.go:572-585`
(`collectLogs` / `Stage.Logs`); `runner.go:128-155` vs `stageb.go:70-90` (the teardown
defer); `runner.go:109` vs `stageb.go:48` (the debug-log plumbing); `runner.go:529-534`
vs `stageb.go:605-610` (`plural`).

**What:** `harness.stageB` is a second, hand-maintained copy of the runner's setup and
teardown. `mountsOf` and `harnessMounts` are the same 13 lines with a different name —
they both decide where a `file` check looks on the host, from `mount.into`. The two
teardown blocks differ in one respect that is not obviously deliberate: the runner honours
`KeepOnFail` (`runner.go:129`), the harness does not (`stageb.go:76` tests only `o.Keep`).
The two log collectors differ in another: the runner truncates trailing newlines and keys
by service, the harness writes a header block.

**Scenario:** SPEC.md 3.3 gains long-syntax mount support in v0.2, so `mount.into` can be
`service:path:ro`. Someone fixes `runner.mountsOf` to split on the last colon. Every
`file` check in `restored check` now works, and every `file` check in `recipe test` stage
B silently resolves to the wrong host path — and stage B is the gate that decides whether
a stranger's recipe is merged.

**Proposed fix:** Move `mountsOf`, `plural`, and the log collector to a shared location
(`internal/check` already owns `check.Mount`, so `check.MountsOf(res)` is the natural
home; the log collector belongs on `compose.Client`). Extract the teardown block into
`workspace.Teardown{Compose *compose.Client, WS *workspace.Workspace, Keep bool}` with one
`Run(ctx)` method, and have both call sites defer it. That also settles the
`KeepOnFail` asymmetry by making it one decision instead of two.

---

### ARCH-08 (P2) No test seam at the docker boundary: `probe`, `runner`, `compose`, `sqlite`, `dir` are all 0%

**Where:** `internal/check/run.go:35-40` (`Executor.Compose *compose.Client` — a concrete
struct), and the resulting coverage:

```
$ go test ./... -cover
	github.com/spelingbee/restored/internal/compose		coverage: 0.0% of statements
	github.com/spelingbee/restored/internal/probe		coverage: 0.0% of statements
	github.com/spelingbee/restored/internal/runner		coverage: 0.0% of statements
	github.com/spelingbee/restored/internal/source/dir	coverage: 0.0% of statements
	github.com/spelingbee/restored/internal/sqlite		coverage: 0.0% of statements
ok  	github.com/spelingbee/restored/internal/harness	coverage: 9.2% of statements
ok  	github.com/spelingbee/restored/internal/loader	coverage: 13.7% of statements
```

**What:** ADR-031 decides there is no mocked Docker API, and I am not re-opening that. But
ADR-031 is about not faking the *daemon*; it does not require that pure logic sitting
above the daemon be untestable. `check.Executor` holds a `*compose.Client` by value type,
so nothing above it can be exercised without a real docker binary on PATH, even where the
code under test never touches docker:

- `probe.Run`'s retry arithmetic — the interval/deadline interaction at
  `internal/probe/probe.go:33-72`, and `RunAll`'s shared-budget carve-up at `:117-135` —
  is pure scheduling logic and has no test.
- `loader.waitForPostgres`'s `notRunningLimit` back-off (`internal/loader/loader.go:124-158`)
  is pure and has no test.
- `internal/sqlite` needs no docker at all. `dsn()` (`internal/sqlite/sqlite.go:63-70`)
  builds `file:<path>?mode=ro&_pragma=...` via `url.URL{Opaque:}`; on Windows that path
  starts with a drive letter and the resulting DSN is untested on any platform.
- `internal/source/dir.Locate` (`internal/source/dir/dir.go:17-20`) is four lines that
  decide where user data is read from, and has no test.
- `internal/compose`'s pure helpers — `composeArgs`, `firstLines`, `Result.Combined`,
  `Bind.arg`, `resticVersion` — have none either.

**Scenario:** A contributor tightens `probe.RunAll` so a probe cannot start with under
five seconds left. They cannot write a test for it without docker, so they do not; CI's
`unit` job is green; the regression surfaces three weeks later as an intermittent
`recipe-health.yml` failure that costs a maintainer an afternoon to bisect.

**Scenario (sqlite, read from code — not run):** `mode=ro` refuses to replay a WAL. The
uptime-kuma recipe deliberately restores `kuma.db` together with its `-wal` companion
(`recipes/uptime-kuma/README.md:29-38`), and a SQLite file whose WAL has uncheckpointed
frames needs recovery on open. In read-only mode modernc returns
`SQLITE_READONLY_RECOVERY` rather than reading through the WAL, which would surface as a
failing `sql` check on a restore that is in fact complete — the exact false alarm the tool
exists to remove. A twenty-line test in `internal/sqlite` (write a db, leave a hot WAL,
open it) would settle this in either direction without docker. I could not run it here.

**Proposed fix:** Give `check.Executor` a two-method interface instead of the struct:

```go
type Runtime interface {
    Exec(ctx context.Context, o compose.ExecOptions) (compose.Result, error)
    RunHelper(ctx context.Context, o compose.RunOptions) (compose.Result, error)
}
```

`*compose.Client` satisfies it unchanged, no production behaviour moves, and `probe`,
`check` and `loader` become unit-testable against a canned `Runtime` — which is testing
*restored's* logic, not faking Docker's, and so is on the right side of ADR-031. Add the
`internal/sqlite` and `internal/source/dir` tests regardless; they need no seam at all.

---

### ARCH-09 (P2) A killed run leaves a full copy of the user's backup in the temp directory, and nothing finds it

**Where:** `internal/workspace/workspace.go:37-65` (workspace under `os.TempDir()` by
default), `internal/cli/cli.go:80-96` (the signal handler), SPEC.md 4.4's orphan story.

**What:** SPEC.md 4.4 promises orphans are recoverable: "every object `restored` creates
carries the label `com.restored.run=<runid>`, so `docker ps -aq --filter
label=com.restored.run` finds them all." That covers containers, networks and volumes. It
does not cover the workspace, which is where the restored backup actually lives. On
`SIGKILL`, an OOM kill, a power loss, or a `docker compose` that wedges past the
two-minute teardown budget, `<tmp>/restored-<runid>/` survives with the user's data in it
and there is no command in this build that lists or removes it. `restored doctor` is v0.2
(SPEC.md 14).

The second `SIGINT` path is a designed instance of the same thing: `cli.go:88-90` prints
"leaving the workspace and the compose project in place" and calls `os.Exit(130)` — but
it prints neither the workspace path nor the project name, because the handler goroutine
has no access to either. SPEC.md 4.4 explicitly requires that it does: "after printing the
workspace path and the compose project name so nothing is silently orphaned."

**Scenario:** A nightly `restored check --all`-style cron over a 40 GB Nextcloud backup is
OOM-killed by the host. Every night. After two weeks `/tmp` (or `/var/tmp`) holds fourteen
0700 directories with a complete copy of somebody's document store, the disk fills, and
the fifteenth run fails at PREPARE with a message about space rather than about the
fourteen copies.

**Proposed fix:** Two small pieces. (1) Make the interrupt handler carry the run identity:
have `runner.Run` publish `ws.Root` and `ws.ProjectName()` into a package-level
`atomic.Pointer` (or accept an `OnStart func(Kept)` callback) that `cli.interruptContext`
reads, so the second-Ctrl-C message says what was left, as 4.4 requires. (2) Add
`restored clean [--older-than 24h] [--dry-run]` that lists `restored-*` directories under
the workspace parent alongside `docker ps -aq --filter label=com.restored.run` and removes
both. It is perhaps eighty lines, it needs no new dependency, and it closes the only path
by which this tool leaves user data on disk.

---

### ARCH-10 (P2) `recipe test` grows the user's restic cache without bound

**Where:** `internal/source/restic/restic.go:44-57` (no `--no-cache`, no `RESTIC_CACHE_DIR`),
`internal/harness/stageb.go:213` and `:241-255` (a brand-new repository per stage B run).

**What:** The harness creates a throwaway restic repository inside the workspace
(`<ws>/repo`) and then runs the *host's* restic against it through `runner.Run`. The
harness's own restic container invocations pass `--no-cache` (`stageb.go:443` and `:466`),
but the host restic in `internal/source/restic` does not, and nothing sets
`RESTIC_CACHE_DIR`. restic keys its cache by repository id, and every stage B creates a
fresh repository with a fresh id. The workspace — including `<ws>/repo` — is deleted at
teardown; the cache entry under `~/.cache/restic/<id>/` is not, and nothing will ever
reference it again.

**Scenario:** `make recipe-test` runs five recipes. Each stage B leaves one orphaned cache
directory. A maintainer iterating on a recipe runs it twenty times in an afternoon and
ends with a hundred dead cache directories, sized in proportion to the seeded data. CI
runners are ephemeral so this never shows up in `recipes.yml`; it only bites the person
doing the work. It is also, incidentally, a write outside the run workspace, which
CLAUDE.md's isolation section says does not happen.

**Proposed fix:** Add `CacheDir string` to `restic.Options` and have the harness point it
at `<ws>/cache` so it dies with the workspace; leave it empty for a user's real
repository, where the cache is a feature. If that is more than wanted, `--no-cache` on the
harness path alone is a one-field change and costs nothing on a repository that is
seconds old.

---

### ARCH-11 (P2) Every failure is an untyped string, which is the wall the notifiers will hit

**Where:** the whole tree — `errors.As` appears exactly twice
(`internal/compose/compose.go:93` for `*exec.ExitError`, `internal/cli/cli.go:68` for
`*exitError`), and there is not one exported sentinel or error type in `internal/`.

**What:** Causes are distinguished by string prefix or not at all.
`internal/loader/loader.go:144` does `strings.Contains(last, "is not running")` against
docker's English output to decide whether to keep waiting — that is a control-flow
decision made by matching a message docker is free to reword. `cli/check.go:180` collapses
every runner error into `fail(ExitError, "%v", runErr)`, so nothing downstream can tell
"docker is not installed" from "the recipe is invalid" from "restic had the wrong
password", even though those want three different responses.

**Scenario (v0.2 notifiers):** The roadmap wants ntfy/Gotify/healthchecks.io posting "the
verdict and a link to the report". healthchecks.io distinguishes a *failure* ping from a
*log* ping, and the whole point of the 1/2 split is that a tool error should not page
anybody. The notifier call site is `cli/check.go` and has: an `error` whose only API is
`Error() string`, and a `*report.Report` that on the exit-2 paths was never rendered and
carries `Verdict: ERROR` with no stages (see ARCH-04). To route correctly it will have to
either re-derive the cause by matching on error text, or the runner will have to grow a
new field — and by then three notifier implementations will depend on whichever was
chosen.

**Proposed fix:** Introduce a small typed error in `internal/runner` now, while there is
one consumer:

```go
type Error struct { Stage string; Kind Kind; Err error }  // Kind: Environment|Recipe|Source|Runtime|Timeout
func (e *Error) Unwrap() error
```

Wrap at the five `return rep, kept, err` sites in `runner.Run` and record `Kind` in the
report as `error_kind`. `cli` keeps mapping to exit codes exactly as it does; the notifier
then has something to switch on that is not English prose. This also gives ARCH-01 its
`Timeout` kind for free. Replace the `strings.Contains(last, "is not running")` check with
the compose exit code, which `compose.Result` already carries.

---

### ARCH-12 (P2) The harness runs a second report renderer, and the check report is not shown when it matters most

**Where:** `internal/harness/render.go:1-60` (a whole second `Report`, `WriteTTY`,
`WriteJSON` and `SchemaVersion`), which imports `internal/report` only for
`report.Options`.

**What:** There are two report packages: `internal/report` for `check`, and an unnamed one
inside `internal/harness` for `recipe test`. They have separate schema versions, separate
TTY renderers, separate colour handling, and separate JSON writers. The harness one
reaches into the check one only for the `Options{Color, ASCII}` struct. Meanwhile the
harness discards the inner check report entirely: `stageb.go:265-276` reads
`rep.Summary.ChecksFailed` and `failedIDs(rep)` and throws the rest away, so when stage B
fails the contributor is told "3 of 5 checks failed after a real round trip" and never
sees which expectation missed by what — the information is in `rep.Checks[].Failures` and
is dropped.

**Scenario:** A first-time contributor's recipe fails stage B. `restored recipe test
./recipes/mine` tells them the round trip did not restore and lists three check ids. To
see the `expect`/`got` pairs they have to reconstruct the run by hand from `st.Command`,
which means finding a restic repository that the harness has already deleted. That is
exactly the moment the project can least afford to be unhelpful — it is the success
metric.

**Proposed fix:** Embed the inner `*report.Report` in `harness.Stage` (`json:"check_report,omitempty"`)
and, in the harness TTY renderer, call the existing `rep.WriteTTY` indented under the
failed stage B rather than summarising it. That deletes duplicate rendering code and hands
the contributor the report the tool already built.

---

### ARCH-13 (P3) Hint selection prioritises subject order over rule order, which is not what SPEC 6.1 specifies

**Where:** `internal/hints/hints.go:118-134`.

**What:** `Match` loops subjects on the outside and rules on the inside, so the first
*subject* with any match wins, and the catalog's ordering — which SPEC.md 6.1 and
`docs/hints.yaml`'s own header call the mechanism ("Matched in order; first match wins",
"rules are written most specific first") — only applies within one subject. The subject
order is fixed in `runner.attachHint` (`runner.go:676-715`): check errors, then bodies,
then stderr, then failures, then `rep.Error`, then warnings, then logs.

**Scenario:** A restore is missing `kuma.db-wal`. The specific rule `sqlite/wal-missing`
would match the service log; the generic `db/tables-empty` matches
`checks[0].failures` ("expected rows_min: 1, got rows: 0"), which is an earlier subject.
`db/tables-empty` wins and the user is told to check whether the dump was of the right
database — for a SQLite recipe with no dump. The catalog ordering that was supposed to
prevent this was never consulted.

**Proposed fix:** Swap the loops: rules outside, subjects inside. Rule order then means
what the catalog says it means, and the change is two lines.

---

### ARCH-14 (P3) `internal/workspace` does not own three paths inside the workspace

**Where:** `internal/harness/stageb.go:195` (`filepath.Join(ws.Root, "staging")`),
`stageb.go:213` (`filepath.Join(ws.Root, "repo")`), `internal/runner/runner.go:109` and
`internal/harness/stageb.go:48` (`filepath.Join(ws.LogsDir(), "debug.log")`).

**What:** SPEC.md 13.1: "`internal/workspace` owns every path. No other package
constructs a path inside the run directory; they ask for one. This is the mechanism that
makes 'nothing outside the workspace' a structural property rather than a habit." Four
sites construct one directly. Every one of them is currently correct, which is precisely
why it will not stay that way: the rule exists so correctness does not depend on the
author remembering.

**Scenario:** v0.2 adds a second export format and someone writes
`filepath.Join(ws.Root, "..", "shared-cache")` because two stage-B runs want to share
pulled layers. `Contains` is never consulted, `CheckResolvedMounts` never sees it because
it is not a compose mount, and the tool writes outside its workspace for the first time.

**Proposed fix:** Add `StagingDir()`, `RepoDir()` and `DebugLog()` to `*Workspace`
alongside the five accessors already there, and make `Contains` the only way any path
inside the root is produced. Five minutes; it keeps the invariant mechanical.

---

### ARCH-15 (P3) Four discarded parameters and one dead struct field

**Where:** `internal/runner/runner.go:265` (`_ = stage`),
`internal/cli/recipe.go:107` (`_ = strict`), `internal/cli/recipe.go:200` (`_ = g`),
`internal/check/jsonpath.go:66` (`_ = expr`), `internal/compose/compose.go:210`
(`RunOptions.Timeout`, declared and never read — `RunHelper` at `:214-221` ignores it and
neither caller sets it).

**What:** Each is a parameter or field threaded through an API and then thrown away.
`_ = stage` is the most consequential: it is the stage name at the moment the run decides
the verdict, discarded, and it is exactly the value ARCH-01 needs. `_ = strict` is
harmless (the caller at `recipe.go:55` applies `strict` itself) but means
`validateOne(ref string, strict bool)` lies about its contract. `RunOptions.Timeout`
suggests helper containers are time-boxed by the option; they are not — the bound comes
from the caller's context (`internal/check/run.go:124`, `:169`), which is correct but is
not what the field says.

**Scenario:** Someone adds a `--strict` behaviour that only makes sense per-recipe and
implements it inside `validateOne`, where the parameter already is. It compiles, the tests
pass, and it does nothing, because the caller's `bad` computation is what actually decides
the exit code.

**Proposed fix:** Delete the four parameters and the field, or use them. `golangci-lint`
does not catch these because `_ =` is an explicit use; enabling `unparam` would.

---

### ARCH-16 (P3) The recipe format is described in four places

**Where:** `schema/recipe.schema.json`, `internal/recipe/types.go`,
`internal/cli/scaffold.go:15-213` and `internal/cli/scaffoldcompose.go:28-266` (two
independent string-template generators), `recipes/TEMPLATE/recipe.yaml`.

**What:** Both scaffolders emit `recipe.yaml` as hand-written Go string templates rather
than by marshalling `recipe.Recipe`, so a field added to the format has to be added in
four places by hand. This is mitigated — and the mitigation is good — by
`TestScaffoldedRecipesValidate` (`internal/cli/scaffold_test.go:16`) and
`TestScaffoldFromComposeValidates` (`internal/cli/detect_test.go:93`), which round-trip
the generated files through `recipe.Load` and `safety.Validate`. So drift is caught, and
this is a cost rather than a defect.

**Scenario:** v0.2 adds `mysql-dump`. The schema, the types, `scaffold.go`'s `--db`
switch, `scaffoldcompose.go`'s `detectedDB.Kind` handling, and `recipes/TEMPLATE` all
change. The tests will catch a broken result but will not catch `--db mysql-dump` being
supported by one scaffolder and not the other.

**Proposed fix:** Nothing urgent. If it is touched anyway, have `scaffoldFromCompose` build
a `recipe.Recipe` and hand it to one shared renderer that also produces
`recipes/TEMPLATE`, so the template becomes generated output covered by
`make check-generated` rather than a fourth hand-maintained copy.

---

## Extension readiness

| Extension | Cost today | Why |
|---|---|---|
| **borg source** | Hard | No `Source` interface exists (ARCH-02). Adding borg means a third `case` in `runner.materialise`, a restic-shaped `source.Descriptor` to generalise (`ShortID`/`Tags`/`Paths`), `Preflight(ctx, needRestic bool)` to redesign, and the restic-only rename/`RestoreDir` special cases at `runner.go:612-646` to disentangle — five packages for what ADR-004 calls "behind the same interface". |
| **kopia source** | Hard, then easy | Identical to borg and no cheaper: the second one only becomes cheap once the first has paid for the interface. Kopia's snapshot model (no `--include`, a different restore-target shape) will also force `source.Request` to grow, and today that struct is consumed directly by the runner rather than by an implementation. |
| **Notifiers** | Medium | The call site is genuinely cheap — one place in `cli/check.go` after `runner.Run` returns, with the report and the exit code both in hand. What is missing is anything to route on: every failure is an untyped string (ARCH-11), and on the exit-2 paths the report is a stub that was never rendered and carries no hint (ARCH-04). A notifier written today would either post "ERROR" with no cause, or match on English error text. |

**The single change that would most reduce the cost:** define
`source.Source` as a real interface (`Preflight`, `Describe`, `Materialise`) implemented by
`restic` and `dir`, and collapse `runner.materialise`'s `switch o.SourceKind` into one
call through it — ARCH-02's fix. It converts borg and kopia from lifecycle surgery into
new files, removes the `needRestic` bool from `internal/compose`, and forces
`source.Descriptor` to become source-agnostic before three sources are depending on its
current shape. It is also the cheapest it will ever be: there are exactly two
implementations and one call site today.

For notifiers specifically the enabling change is different and smaller — ARCH-11's typed
`runner.Error` with a `Kind` — and it is worth doing at the same time, because ARCH-01's
fix needs the `Timeout` kind anyway.
