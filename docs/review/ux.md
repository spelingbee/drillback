# UX review — restored

Date 2026-08-30, commit `d5c2f6c2d1fa8e5fff0fb5315f1e707604db4365`, branch `main`, clean tree.
Built `go build -o restored.exe ./cmd/restored` into a scratch directory; everything below was
run from there against a scratch working directory, never against the repository tree.
Ran: every `--help`, `version`, `version --json`, `recipe validate` (valid, invalid, malformed,
unsafe, missing), `recipe show` (all flags and formats), `recipe init` (plain, `--db`, `--compose`,
every rejection), `recipe test` on load failures only, and ~35 wrong invocations of `check`.
Did NOT run: anything that starts a container — `recipe test` past recipe load, the demo scripts,
integration tests. For those paths I read `internal/runner`, `internal/harness` and `docs/demo/*.txt`.
This host has docker 29.5.2 and no restic, which made the "restic missing" degradation real rather
than simulated; "docker missing" and "daemon unreachable" were simulated with a stripped `PATH`
and a bogus `DOCKER_HOST`, both of which fail in `compose.Preflight` before any container exists.

## Summary

| severity | count |
|---|---|
| P0 | 2 |
| P1 | 4 |
| P2 | 8 |
| P3 | 3 |
| total | 17 |

The CLI gets a great deal right, and most of it is right in the places that are hard. The safety
rejections are the best error messages in the tool: `service "app" uses \`ports\`, which restored
does not allow: restored never publishes a port; checks run from a helper container on the run's
internal network` says what happened, why, and what the tool does instead, in one line. `recipe
init --compose` is genuinely excellent — it summarises what it inferred, calls itself "a proposal,
not a recipe", and hands back a numbered four-step list ending in the command CI runs. The
per-recipe READMEs map real deployments to input names better than most tools map anything. Exit
codes are a real contract, the 1/2 split is honoured everywhere I could reach, colour is correctly
gated on a TTY, and no escape byte reached a pipe in any invocation.

The single worst moment a new user will have is the second command they type. They run
`restored check --recipe gitea`, their backup does not keep Gitea at `/srv/gitea/data`, and they
get exactly one line: `required input "data": no matching files found for /srv/gitea/data in the
backup`. No next step, no `--input` example, no `restic ls`. What makes this a P0 rather than a
wording nit is that the fix is already written and shipped: `docs/hints.yaml` contains a rule whose
`match` is literally `no matching files found for`, whose text is "Point the input at your path
with `--input <name>=/your/path`", and whose command is `restic ls latest | head -50`. It never
fires, because `attachHint` is only called on the two success-shaped return paths in `runner.Run`.
The same is true of the docker-daemon rule, the disk-space rule, the image-pull rule and the
permissions rule. Six of the seventeen hint rules are dead in the exact situations they were
written for, and the user gets raw Go plumbing instead. Behind that sits the second P0: there is no
way to get the binary from the README at all.

## Time to first PASS

The user: a self-hoster with a Synology, restic at `/volume1/backups/restic`, Gitea in docker
compose, backups going nightly. Minutes are from the tool's own documentation, not from a
sympathetic reading of it.

| # | Step | Minutes | Where it breaks down |
|---|---|---|---|
| 1 | Read README top screen | 0.5 | Works. The tagline, the three-sentence summary and the pre-release note all land inside 20 seconds. |
| 2 | Look for how to install | 3-8 | **Dead end.** README has no install section, no `go install`, no release link, no `brew`, no `scripts/install.sh` (it does not exist). The word "install" appears twice in README.md and neither is an instruction. The only build steps are in CONTRIBUTING.md under a heading called "Add a recipe in 10 minutes", which a user verifying a backup has no reason to open. UX-02. |
| 3 | Install Go 1.27, clone, `go build` | 5-25 | Not stated as a prerequisite anywhere a user looks. Neither is docker or restic. |
| 4 | `restored version` | 0.5 | Works well. Prints docker, compose and restic versions, or `not found`, and stays exit 0. This is the best onboarding surface in the tool and nothing points at it. |
| 5 | `restored recipe show gitea --inputs-only` | 1 | Prints `data  dir  required  /srv/gitea/data` with no column header and no line saying "override with `--input data=...`". The user has to infer the contract. UX-11. |
| 6 | Find their own paths | 3-10 | `recipes/gitea/README.md` answers this properly, and the quick start does not link to it. |
| 7 | `export RESTIC_REPOSITORY` / `RESTIC_PASSWORD_FILE` | 1 | The README quick start shows this. `restored check --help` no longer does — the `Environment:` block SPEC.md section 2.2 specifies was dropped. UX-11. |
| 8 | `restored check --recipe gitea` | 1 | **Dead end.** `required input "data": no matching files found ...` and nothing else. UX-03, UX-01. |
| 9 | Work out `restic ls latest`, retry with two `--input` flags | 5-15 | Unaided. `check --help` contains no `--input` example; SPEC has one and SPEC is not shipped to users. |
| 10 | First real run, image pulls included | 2-5 | Should work. |

**Steps: 10. Decisions the user must make unaided: 5** (how to obtain the binary; restic or dir;
which snapshot; the real path for `data`; the real path for `db`). **Realistic time to first PASS:
25-50 minutes**, against a README that says "One command, about a minute". The tool's own guidance
covers roughly ten of those minutes; the rest is the user reverse-engineering things the tool
already knows and does not say.

Required knowledge is not stated before it is needed at three points: the toolchain (step 2/3),
the input-name contract (step 5), and the restic environment variables (step 7).

---

## Findings

### UX-01 (P0) The hint catalog never fires on any failure that returns an error

**Where:** `internal/runner/runner.go:263` and `internal/runner/runner.go:362` are the only two
calls to `attachHint`, and both are on paths that return `nil` error.
`internal/runner/runner.go:180-189` (restore), `:226` (compose up), `:89-91` (preflight) all
return `rep, kept, err` without it. `internal/cli/check.go:149` then suppresses the report:
`if runErr == nil { rep.WriteTTY(...) }`.

**What the user sees:**

```
$ DOCKER_HOST=tcp://127.0.0.1:1 restored check --recipe gitea --source dir --from .
restored: cannot reach the docker daemon: error during connect: Get "http://127.0.0.1:1/v1.54/version": dial tcp 127.0.0.1:1: connectex: No connection could be made because the target machine actively refused it.
EXIT=2
```

```
$ restored check --recipe gitea --source dir --from .
restored: required input "data": no matching files found for /srv/gitea/data in the backup
EXIT=2
```

The `--report` JSON written on that second run contains no `hint` key at all:

```json
  "stages": [
    {
      "name": "restore",
      "status": "failed",
      "duration_ms": 0,
      "error": "required input \"data\": no matching files found for /srv/gitea/data in the backup"
    }
  ],
  "checks": null,
  "summary": { "checks_total": 0, "checks_passed": 0, "checks_failed": 0, "checks_skipped": 0 }
```

**Why it is wrong:** `docs/hints.yaml` already answers both of these. `restore/path-not-in-snapshot`
matches `(?i)(no matching files found for|...)` and its text is "Recipe defaults are a guess at the
most common layout. Point the input at your path with `--input <name>=/your/path` ... use
`restic ls latest | head -50`". `docker/daemon-unreachable` matches
`(?i)(cannot connect to the docker daemon|docker daemon is not running|...)` and its text names
`DOCKER_HOST`, rootless sockets and WSL2 integration. Neither can ever run. The same is true of
`workspace/no-space` (ENOSPC arrives from `materialise`), `compose/image-pull-failed` and
`compose/port-conflict` (both arrive as `upErr` at `:226`), and `permissions/eacces`. Six of
seventeen rules, and specifically the six aimed at first-run failures, are unreachable code. The
hint mechanism is the largest single UX investment in the repository and it is switched off for
the class of user it was built for. It also means CONTRIBUTING.md's claim that a hint rule is a
cheap contribution is only true for rules about check failures.

**Proposed fix:** call `attachHint` on the error paths too, and print the report before the error
line. Concretely, in `runner.Run`, replace each bare `return rep, kept, err` after the workspace
exists with a small helper:

```go
// toolError records a stage failure, offers a hint for it, and returns exit 2.
// A hint is presentation only: it never changes the verdict. See SPEC.md 6.1.
toolError := func(e error) (*report.Report, *Kept, error) {
    rep.Error = e.Error()
    o.attachHint(rep, res)
    finish(rep, started)
    return rep, kept, e
}
```

and in `internal/cli/check.go:149`, render the report whenever `rep.Stages` is non-empty rather
than only when `runErr == nil`, so the LIKELY CAUSE block reaches the user on stage failures.
For the preflight case, which runs before `res` exists, match the catalog against the error string
alone.

Because `docker/daemon-unreachable` matches "cannot connect to the docker daemon" but
`compose.Preflight` emits "cannot reach the docker daemon", also change
`internal/compose/env.go:38` to:

```go
return fmt.Errorf("cannot connect to the docker daemon: %s",
    firstLine(errOutput(ctx, "docker", "version", "--format", "{{.Server.Version}}")))
```

### UX-02 (P0) There is no way to install the tool from the README

**Where:** README.md:143 (`## Quick start`, which opens with `export RESTIC_REPOSITORY=...` and
then calls `restored` as if it were on `PATH`). No install section exists; README.md:121 mentions
`make build` only inside the "reproduce the demo" block; `scripts/install.sh` does not exist.

**What the user sees:** README.md:143-150, in full — the whole of what the document says about
getting started, sitting between the two demo blocks and the recipe table:

```
    ## Quick start

    export RESTIC_REPOSITORY=/mnt/backups/restic
    export RESTIC_PASSWORD_FILE=/etc/restic/pass
    restored recipe show gitea --inputs-only          # which paths does this recipe want?
    restored check --recipe gitea --input data=/srv/gitea --input db=/srv/dumps/gitea.sql
    echo $?                                           # 0 pass, 1 unusable, 2 tool error
```

`grep -n install README.md` returns two lines, neither of which is an instruction:

```
294:contribution types; the bot that maintains it is not installed, because installing an
306:  harness, six bundled recipes, CI, and the install paths.
```

**Why it is wrong:** the README is the launch artifact, and step one of every reader's journey is
missing. The prerequisites (docker with compose v2, restic, Go 1.27) are stated only in
CONTRIBUTING.md section 0, under a heading addressed to recipe contributors. A user who wants to
verify a backup has to guess that "how do I add a recipe" is where "how do I get the binary" lives.
This also makes README.md:306's roadmap line ("six bundled recipes ... and the install paths")
read as a description of shipped state when neither is true — five recipes exist and no install
path does.

**Proposed fix:** add this immediately after the pre-release note and before "What it looks like",
so it is on the first screen:

~~~markdown
## Install

There are no released binaries yet. You need **docker** with compose v2, **restic**, and
**Go 1.27**:

```sh
git clone https://github.com/spelingbee/restored && cd restored
go build -o bin/restored ./cmd/restored
./bin/restored version     # confirms docker, compose and restic are all reachable
```

`restored version` never fails for a missing dependency, so it is the first thing to run and
the thing to paste into a bug report.
~~~

and move `## Quick start` up to sit directly beneath it, with the two demo blocks after both.

### UX-03 (P1) The most likely first failure gives the user nothing to do

**Where:** `internal/runner/runner.go:604-606`.

**What the user sees:**

```
$ restored check --recipe gitea --source dir --from .
restored: required input "data": no matching files found for /srv/gitea/data in the backup
EXIT=2
```

**Why it is wrong:** this is the default outcome for anyone whose layout is not the recipe
author's guess, which is most people — `recipes/gitea/README.md` itself lists three common layouts
and only one of them is `/srv/gitea/data`. The message names the problem and stops. It does not
name the flag that fixes it, does not mention that `restored recipe show gitea --inputs-only`
lists the input names, and does not say how to find out what the snapshot actually contains. It is
the single highest-traffic error string in the tool. UX-01 would put the hint back underneath it;
this finding is about the line itself, which should stand alone in a `--json` consumer's `stages[].error`.

**Proposed fix:** replace `internal/runner/runner.go:604-606` with:

```go
return desc, warnings, fmt.Errorf(
    "required input %q: nothing at %s in the backup.\n"+
        "  Recipe default paths are a guess. Point this input at your path with\n"+
        "    --input %s=/your/path\n"+
        "  `restored recipe show %s --inputs-only` lists every input this recipe wants.",
    in.Name, in.BackupPath, in.Name, res.Recipe.Metadata.Name)
```

### UX-04 (P1) Five of seven commands print `check`'s exit codes in their `--help`, and they are wrong there

**Where:** `internal/cli/cli.go:117-125` appends the footer to the root help template, which cobra
inherits into every subcommand. Only `recipe test` overrides it (`internal/cli/recipetest.go:117`).

**What the user sees:** the tail of `restored recipe validate --help`, `restored recipe show --help`,
`restored recipe init --help` and `restored version --help`, identically:

```
Exit codes:
  0   all checks passed
  1   restore unusable — one or more checks failed, or the app never became ready
  2   tool or runtime error — docker missing, restic failed, recipe invalid, timeout
      before any check could run

Docs: https://github.com/spelingbee/restored
```

**Why it is wrong:** `recipe validate` returns 0 or 2 and never 1 — SPEC.md section 2.4 says so
explicitly and the code agrees (`internal/cli/recipe.go:69` returns `ExitError` for any bad
recipe). Same for `show`, `init` and `version`. Anyone writing `if [ $? -eq 1 ]` around
`recipe validate` because the help told them 1 means "checks failed" has written a branch that can
never be taken. `restored version --help` promising "all checks passed" as its exit 0 is simply
false. It also makes the footer look like boilerplate, which trains the reader to skip the one
place it is accurate.

**Proposed fix:** attach the footer per command instead of to the root template. For
`recipe validate`, `recipe show` and `recipe init`:

```
Exit codes:
  0   ok
  2   the recipe is invalid, or the command could not be run as written

Docs: https://github.com/spelingbee/restored
```

`restored version --help` should carry no exit-code footer at all, because it is documented never
to fail. Keep the current text on `restored --help` and `restored check --help` only, and add the
missing row there:

```
  130 interrupted — a second Ctrl-C during teardown; the workspace and the compose
      project may still exist
```

### UX-05 (P1) `--json` is a global flag that silently does nothing on half the commands

**Where:** `internal/cli/cli.go:110` declares `--json` persistently on the root.
`internal/cli/recipe.go:200` (`_ = g` in `recipe show`) and `internal/cli/recipe.go:300`
(`recipe init`) never read it. `recipe show` uses `--format json` instead
(`internal/cli/recipe.go:186`).

**What the user sees:**

```
$ restored recipe show gitea --json
metadata:
    name: gitea
    title: Gitea + PostgreSQL
    description: |
        Verifies that a Gitea backup restores: the web UI renders, the repository and user rows are in the database, and at least one bare repository is present on disk.
    maintainers:
        - '@example-handle'
    upstream: https://github.com/go-gitea/gitea
EXIT=0
```

```
$ restored recipe init zz --json
restored: recipe name "zz": expected 3 to 40 lower-case letters, digits and hyphens
EXIT=2
```

**Why it is wrong:** the flag is advertised in `recipe show --help` under `Global Flags` with the
text "Emit the machine-readable report on stdout", it is accepted without complaint, and it emits
YAML. A script that pipes `restored recipe show gitea --json | jq` gets a parse error with no
explanation anywhere in the tool. This is also the direct answer to "is `--json` vs `--format json`
coherent": no. Two flag names mean JSON, one of them works on `check`, `recipe validate` and
`recipe test`, the other works on `recipe show`, and on `recipe show` the first is a silent no-op.
`version` adds a third case: it declares its own local `--json` (`internal/cli/version.go`) that
shadows the global one with a different description.

**Proposed fix:** make `recipe show` honour the global flag, treating it as `--format json`:

```go
if g.json {
    format = "json"
}
```

placed at the top of the `RunE`, and change the `--format` flag's help to
`"yaml|json (default \"yaml\"; --json is equivalent to --format json)"`. On `recipe init`, which has
no machine-readable output to give, reject it rather than ignore it:

```go
if g.json {
    return fail(ExitError, "--json: recipe init has no machine-readable output; it writes files and prints what it wrote")
}
```

### UX-06 (P1) `schema_version` is the number 1 in one report and the string "1" in the other

**Where:** `internal/report/report.go:18` (`const SchemaVersion = 1`, typed `int` at
`report.go:30`) against `internal/harness/render.go:15` (`const SchemaVersion = "1"`, typed
`string` at `render.go:20`).

**What the user sees:**

```
$ restored check --recipe gitea --source dir --from . --json
{
  "schema_version": 1,
  "tool": { "name": "restored", "version": "0.1.0-dev" },
```

```
$ restored recipe test ./nosuchdir --json
{
  "schema_version": "1",
  "tool": { "name": "restored", "version": "0.1.0-dev" },
```

**Why it is wrong:** SPEC.md section 5.2 documents one field, `"schema_version": 1`, and calls it
"the stability contract". Two documents from the same binary, with the same field name and the
same `tool` block, disagree on its type. A CI job that does `jq -e '.schema_version == 1'` passes
against `check` and fails against `recipe test`, for a reason nothing in the output explains. It is
also the kind of thing that is free to fix now and a breaking change after the first tag, which is
what makes it P1 rather than P2. SPEC.md documents no harness JSON schema at all, so there is no
document to point at when a contributor asks which one is right.

**Proposed fix:** change `internal/harness/render.go:15-20` to match `report`:

```go
// SchemaVersion is the harness report's stability contract, separate from the check
// report's but deliberately the same type and the same field name. See SPEC.md 5.2.
const SchemaVersion = 1

type Report struct {
	SchemaVersion int `json:"schema_version"`
```

and update `internal/cli/recipetest.go:55` accordingly. Then add a short "5.4 Harness JSON report"
to SPEC.md documenting the `recipe test` document, so the two contracts are written down in the
same place.

### UX-07 (P2) `recipe validate --strict` prints `ok` and exits 2, and its warnings do not name the recipe

**Where:** `internal/cli/recipe.go:111-125` (`printFinding`), `internal/cli/recipe.go:52-55`
(the `bad` flag that turns warnings into exit 2).

**What the user sees:**

```
$ restored recipe validate ./recipes/myapp ./recipes/myapp3 --strict
ok       ./recipes/myapp
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
ok       ./recipes/myapp3
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
EXIT=2
```

**Why it is wrong:** two problems in six lines. The verdict column says `ok` on a run that exits 2,
so the human output and the exit code contradict each other and nothing on screen says `--strict`
is the reason. And the warning lines carry no recipe path, so with a glob over seven recipes the
reader cannot tell which recipe each warning belongs to — the `ok` line above it is the only clue,
and warnings that follow the last recipe look like they belong to the whole run. In `--json` the
attribution is correct (warnings nest inside the finding), which makes the human output strictly
worse than the machine output.

**Proposed fix:** in `printFinding`, mark the strict verdict and indent warnings under their
recipe:

```go
func printFinding(cmd *cobra.Command, f finding, strict bool) {
	w := cmd.OutOrStdout()
	switch {
	case !f.Valid:
		fmt.Fprintf(w, "INVALID  %s\n", f.Recipe)
		for _, e := range f.Errors {
			fmt.Fprintf(w, "         %s\n", e)
		}
	case strict && len(f.Warnings) > 0:
		fmt.Fprintf(w, "WARN     %s\n", f.Recipe)
	default:
		fmt.Fprintf(w, "ok       %s\n", f.Recipe)
	}
	for _, warn := range f.Warnings {
		fmt.Fprintf(w, "         warning: %s\n", warn)
	}
}
```

and print one closing line when `strict` caused the failure:

```
--strict: warnings are failures. 2 recipes carry warnings; fix them or drop --strict.
```

### UX-08 (P2) `recipe init` tells the user to run a command that exits 2, and never mentions `recipe test`

**Where:** `internal/cli/recipe.go:327-330`.

**What the user sees:**

```
$ restored recipe init myapp
Wrote recipes\myapp

Next:
  1. make the checks data-sensitive: a check that passes against an empty
     database proves nothing about a restore
  2. restored recipe validate recipes\myapp --strict
EXIT=0

$ restored recipe validate ./recipes/myapp --strict
ok       ./recipes/myapp
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
EXIT=2
```

**Why it is wrong:** the second command the tool tells a first-time contributor to type fails, for
a reason the scaffold created on purpose (`maintainers: []`, with the comment "Put your GitHub
handle here"). A contributor two minutes into their first PR gets a red exit from following
instructions. The list also stops one step short: it never names `restored recipe test`, which is
the gate CI actually applies and the thing the "no data-sensitive check" advice in step 1 is
preparing them for. The `--compose` variant of the same command
(`internal/cli/scaffoldcompose.go`) gets all of this right and prints a numbered four-step list
ending in `restored recipe test recipes\fromcompose     # this is what CI runs`, which is exactly
the shape the plain path should have.

**Proposed fix:** replace `internal/cli/recipe.go:327-330` with the same shape as the `--compose`
path:

```go
fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n\nThis is a skeleton, not a recipe yet. Next:\n\n"+
	"  1. put your GitHub handle in metadata.maintainers, and fill in the TODO markers\n"+
	"     in %s\n"+
	"  2. make one check data-sensitive: a check that passes against an empty database\n"+
	"     proves nothing about a restore, and the round-trip harness will refuse it\n"+
	"  3. restored recipe validate %s --strict\n"+
	"  4. restored recipe test %s     # this is what CI runs\n",
	target, filepath.Join(target, "recipe.yaml"), target, target)
```

### UX-09 (P2) `restored recipe validate ./recipes/*` — the command CLAUDE.md and SPEC both give — exits 2

**Where:** `internal/cli/recipe.go:48` (`cobra.MinimumNArgs(1)`, every argument treated as a
recipe), `recipes/README.md` sitting in the globbed directory. Documented at CLAUDE.md:53 and
SPEC.md section 2.4 ("`restored recipe validate ./recipes/*` ... `--strict`  # what CI runs").

**What the user sees:**

```
$ restored recipe validate ./recipes/* --strict
INVALID  C:/My/Projects/Work/restored/recipes/README.md
         recipe "C:/My/Projects/Work/restored/recipes/README.md": parsing YAML: yaml: line 6: could not find expected ':'
ok       C:/My/Projects/Work/restored/recipes/TEMPLATE
ok       C:/My/Projects/Work/restored/recipes/gitea
ok       C:/My/Projects/Work/restored/recipes/nextcloud
ok       C:/My/Projects/Work/restored/recipes/paperless-ngx
ok       C:/My/Projects/Work/restored/recipes/uptime-kuma
ok       C:/My/Projects/Work/restored/recipes/vaultwarden
EXIT=2
```

**Why it is wrong:** the command in the project's own contributor documentation is red on a clean
tree, and the failure is reported as a YAML syntax error in a Markdown file rather than as "that is
not a recipe". CI dodges it by naming recipes individually (`.github/workflows/ci.yml:101`), so
nothing catches it. A contributor who runs the documented command before opening a PR concludes
they broke something. The same shape produces the message for an empty directory:

```
$ restored recipe validate ./bad/empty
INVALID  ./bad/empty
         reading recipe "bad\\empty\\recipe.yaml": open bad\empty\recipe.yaml: The system cannot find the file specified.
EXIT=2
```

**Proposed fix:** skip arguments that are plainly not recipes, and say so once. In the
`validate` `RunE`, before `validateOne`:

```go
// A glob over recipes/ picks up README.md and other non-recipes. Silently skipping
// them makes the documented `validate ./recipes/*` work on a clean tree.
if skip := notARecipe(ref); skip != "" {
	fmt.Fprintf(cmd.ErrOrStderr(), "skipped   %s (%s)\n", ref, skip)
	continue
}
```

where `notARecipe` returns `"not a directory and not a recipe.yaml"` for a regular file whose base
name is not `recipe.yaml`, and `"no recipe.yaml here"` for a directory without one. And replace the
raw `open ...` error for a directory with:

```
no recipe.yaml in "./bad/empty" — a recipe is a directory holding recipe.yaml and compose.yaml.
`restored recipe init <name>` scaffolds one.
```

### UX-10 (P2) Raw OS and Go plumbing errors reach the user in five places

**Where:** `internal/recipe` load errors surfaced verbatim by `internal/cli/recipe.go:83`,
`internal/cli/check.go:105`, `internal/cli/recipetest.go`; `internal/compose/env.go:38`;
`internal/runner/runner.go` workspace creation.

**What the user sees, five separate invocations:**

```
$ restored check --recipe gitea --source dir --from /nonexistent/tree
restored: --from "C:/Program Files/Git/nonexistent/tree": GetFileAttributesEx C:/Program Files/Git/nonexistent/tree: The system cannot find the path specified.

$ restored recipe validate ./recipes/nosuch
INVALID  ./recipes/nosuch
         reading recipe "./recipes/nosuch": GetFileAttributesEx ./recipes/nosuch: The system cannot find the file specified.

$ restored recipe test ./nosuchdir
  reading recipe "./nosuchdir": GetFileAttributesEx ./nosuchdir: The system
  cannot find the file specified.

$ restored check --recipe gitea --source dir --from . --workspace /nonexistent-parent
restored: creating workspace "C:\\Program Files\\Git\\nonexistent-parent\\restored-7whgkcmq": mkdir C:\Program Files\Git\nonexistent-parent: Access is denied.

$ DOCKER_HOST=tcp://127.0.0.1:1 restored check --recipe gitea --source dir --from .
restored: cannot reach the docker daemon: error during connect: Get "http://127.0.0.1:1/v1.54/version": dial tcp 127.0.0.1:1: connectex: No connection could be made because the target machine actively refused it.
```

**Why it is wrong:** `GetFileAttributesEx` is a Win32 API name; it means nothing to the person
typing the command, and it is the first thing they will paste into a search engine. Note also that
`%q` on a Windows path doubles every separator, so the path the user is shown
(`"bad\\empty\\recipe.yaml"`) is not the path they typed and not a path that exists. The
daemon-unreachable line is four clauses of transport detail wrapping one fact. Every one of these
is an `os.Stat` or `os.MkdirAll` whose only interesting outcome is "not there" or "not allowed",
and Go's `errors.Is(err, fs.ErrNotExist)` distinguishes them for free.

**Proposed fix:** wrap the three path checks with a shared helper and stop printing the syscall.
For `--from`:

```go
if err := dirsource.Check(o.From); err != nil {
    if errors.Is(err, fs.ErrNotExist) {
        return desc, nil, fmt.Errorf("--from %s: no such directory", o.From)
    }
    return desc, nil, fmt.Errorf("--from %s: %w", o.From, err)
}
```

For `--workspace`:

```
--workspace /nonexistent-parent: cannot create a run workspace there (permission denied).
Point --workspace at a directory you can write to with room for the restored data.
```

Use `%s` rather than `%q` for filesystem paths throughout `internal/cli` and `internal/runner`, so
Windows separators are not doubled. And truncate the docker transport error to its first clause:

```go
return fmt.Errorf("cannot connect to the docker daemon at %s: %s",
    dockerHost(), firstClause(errOutput(...)))
```

### UX-11 (P2) `check --help` dropped the `Environment:` block and every `--input` example

**Where:** `internal/cli/check.go:46-57` (`Long` and `Example`), against SPEC.md section 2.2.
Also `internal/cli/recipe.go:249-262` (`writeInputTable`, no header row) and the absence of
`Example` on `recipe validate`, `recipe show` and `recipe init` (`internal/cli/recipe.go:40`,
`:130`, `:296`).

**What the user sees:**

```
$ restored check --help
...
Examples:
  # a local recipe directory, against a tree that is already restored on disk
  restored check --recipe ./recipes/uptime-kuma --source dir --from /mnt/export/uk

  # restic repository from the environment, bundled recipe, latest snapshot
  export RESTIC_REPOSITORY=/mnt/backups/restic
  export RESTIC_PASSWORD_FILE=/etc/restic/pass
  restored check --recipe gitea
```

```
$ restored recipe show gitea --inputs-only
data  dir            required  /srv/gitea/data
db    postgres-dump  required  /srv/gitea/db.sql
```

`restored recipe validate --help`, `restored recipe show --help` and `restored recipe init --help`
have no `Examples:` section at all, though SPEC.md sections 2.4, 2.5 and 2.6 each specify three.

**Why it is wrong:** three things a user needs are documented in SPEC.md, which does not ship to
users, and nowhere in `--help`, which does. First, `--input name=path` — the flag that fixes the
single most common failure (UX-03) — appears in no example. Second, the `Environment:` block naming
`RESTIC_PASSWORD_FILE`, `RESTIC_PASSWORD_COMMAND` and the pass-through of `AWS_*`/`B2_*` is gone,
so a user with an S3 restic repository has no way to learn from the tool that their credentials are
forwarded. Third, the `--inputs-only` table — which its own flag help calls "the fastest way to
answer which paths does this recipe want from my backup" — has no header, so the four columns are
a guess, and nothing tells the reader that column one is what goes on the left of `--input`.
Ordering compounds it: the first example shown is `--source dir`, the rarer of the two sources.

**Proposed fix:** restore the SPEC examples in SPEC's order, restic first, and add the `--input`
one. In `internal/cli/check.go:52`:

```go
Example: "  # restic repository from the environment, bundled recipe, latest snapshot\n" +
	"  export RESTIC_REPOSITORY=/mnt/backups/restic\n" +
	"  export RESTIC_PASSWORD_FILE=/etc/restic/pass\n" +
	"  restored check --recipe gitea\n\n" +
	"  # your paths are not the recipe's defaults: remap the inputs\n" +
	"  #   restored recipe show gitea --inputs-only   lists the names\n" +
	"  restored check --recipe gitea \\\n" +
	"      --input data=/volume1/docker/gitea/data \\\n" +
	"      --input db=/volume1/backups/dumps/gitea.sql\n\n" +
	"  # only snapshots this host wrote, tagged \"gitea\"\n" +
	"  restored check --recipe gitea --host hypervisor --tag gitea\n\n" +
	"  # a local recipe directory, against a tree that is already restored on disk\n" +
	"  restored check --recipe ./recipes/uptime-kuma --source dir --from /mnt/export/uk",
```

and append to `Long`:

```
Environment:
  RESTIC_REPOSITORY, RESTIC_PASSWORD, RESTIC_PASSWORD_FILE, RESTIC_PASSWORD_COMMAND and
  the backend variables restic itself reads (AWS_*, B2_*, AZURE_*, ...) are passed through
  to restic unchanged. restored never parses or logs their values.
```

Give `writeInputTable` a header and a footer:

```
NAME  KIND           REQUIRED  PATH IN THE BACKUP
data  dir            required  /srv/gitea/data
db    postgres-dump  required  /srv/gitea/db.sql

Override any of these with --input NAME=/your/path.
```

and add the SPEC examples to `validate`, `show` and `init`.

### UX-12 (P2) The ASCII fallback is undocumented, unreachable by flag, and incomplete

**Where:** `internal/cli/check.go:147` and `internal/cli/recipetest.go:88`
(`ASCII: os.Getenv("RESTORED_ASCII") != ""`, the only two places it is set);
`internal/harness/render.go:72` and `internal/harness/render.go:125` and
`internal/nudge/nudge.go:42`, which hard-code Unicode and never consult `Options.ASCII`.
`RESTORED_ASCII` appears nowhere outside those two lines — not in `--help`, README, SPEC or the
recipe docs.

**What the user sees:** `--no-color` and `NO_COLOR=1` correctly strip every escape byte (verified
through `cat -v`; nothing leaks in a pipe), but neither changes a glyph, and `RESTORED_ASCII=1`
does not reach the harness renderer at all:

```
$ RESTORED_ASCII=1 restored recipe test ./nosuchdir
  ────
  1 recipe: 0 passed, 0 failed, 1 errored, in 0ms
```

The fallback itself is good where it is wired up — `internal/report/testdata/pass-ascii.txt`
lines up in exactly the same columns as `pass.txt`:

```
  +  web-ui-renders      The web UI renders the instance home page       0.21s
  +  repos-in-db         The database contains at least one repository   0.04s
                         row -> 7
  PASS  3/3 checks  |  total 1m 02s  |  teardown ok
```

**Why it is wrong:** there is a working, tested, well-laid-out ASCII rendering that no user can
discover. A Windows console on a legacy code page, an older CI log viewer, or a terminal with a
font missing U+2714 gets mojibake where the verdict glyph should be, and neither `--help` nor
`NO_COLOR` nor `--no-color` offers a way out. Separately, three renderers ignore the option even
when it is set: the harness rule (`────`), the harness phase separator (`·`), and the nudge box
rule, so `RESTORED_ASCII=1` produces a half-ASCII page. `internal/report/tty.go:10` promises "the
verdict reads identically through NO_COLOR, through `| cat`, and in a screenshot" — which is true
of colour and not of glyphs.

**Proposed fix:** add a real flag next to `--no-color` in `internal/cli/cli.go:113`:

```go
root.PersistentFlags().BoolVar(&g.ascii, "ascii", false,
	"Draw the report with ASCII only (+ x -> |), for terminals without the report glyphs")
```

set `ASCII: g.ascii || os.Getenv("RESTORED_ASCII") != ""` in both call sites, and default it to
true when the platform is Windows and `WT_SESSION` is unset. Then thread `report.Options` through
the two harness renderers and the nudge:

```go
rule := "----"
if !o.ASCII {
	rule = "────"
}
```

with the same treatment for `phaseLine`'s separator and `nudge.Build`, which should take an
`ASCII bool` on its `Input`.

### UX-13 (P2) The README quick start remaps `data` to a path its own recipe README contradicts

**Where:** README.md:148-149 against `recipes/gitea/README.md:12` and `recipes/gitea/recipe.yaml`.

**What the user sees:**

```sh
restored recipe show gitea --inputs-only          # which paths does this recipe want?
restored check --recipe gitea --input data=/srv/gitea --input db=/srv/dumps/gitea.sql
```

but the command on the line above prints:

```
$ restored recipe show gitea --inputs-only
data  dir            required  /srv/gitea/data
db    postgres-dump  required  /srv/gitea/db.sql
```

and `recipes/gitea/README.md` says:

```
| `data` | the whole `/data` directory, including `git/repositories/` | `/srv/gitea/data` |
```

**Why it is wrong:** the README's two adjacent lines disagree. The first asks the tool which paths
it wants and is told `/srv/gitea/data`; the second overrides that to `/srv/gitea`, its parent, for
no stated reason. Since `data` is mounted at `gitea:/data` and the checks look for
`*/*.git/HEAD` under it, the README's own command would mount a directory whose child is `data/`
rather than the data directory, and fail. This is the most-copied command in the document, and it
teaches the mistake the recipe README exists to prevent. It also undercuts the `--inputs-only`
step immediately above it: the user learns that the answer the tool gives is not the answer to use.

**Proposed fix:** make the two lines agree and say why the override is there. README.md:148-149:

```sh
restored recipe show gitea --inputs-only          # which paths does this recipe want?
# Those are the recipe's guesses. Point each one at your layout:
restored check --recipe gitea \
    --input data=/srv/gitea/data \
    --input db=/srv/dumps/gitea.sql
echo $?                                           # 0 pass, 1 unusable, 2 tool error
```

and add one line under the block: ``Each recipe's README maps the common deployments to these
names — see [`recipes/gitea/`](recipes/gitea/).``

### UX-14 (P2) JSON error reports carry `null` arrays and an empty `run` block

**Where:** `internal/report/report.go:37` and `:39` (`Inputs []Input` and `Checks []Check` with no
`omitempty` and no initialisation), `internal/report/report.go:59` (`FinishedAt string`),
`internal/report/report.go:51` (`Commit string` with `omitempty`), `internal/runner/runner.go:74-86`
(the report constructed before `rep.Run` is populated).

**What the user sees, from `--report` on a preflight failure:**

```json
{
  "schema_version": 1,
  "tool": { "name": "restored", "version": "0.1.0-dev" },
  "run": {
    "id": "",
    "compose_project": "",
    "started_at": "",
    "finished_at": "",
    "duration_ms": 0,
    "workspace_removed": false
  },
  "verdict": "ERROR",
  "exit_code": 2,
```

and from `--report` on a restore failure:

```json
  "source": { "kind": "" },
  "inputs": null,
  "stages": [ { "name": "restore", "status": "failed", "duration_ms": 0, "error": "..." } ],
  "checks": null,
  "summary": { "checks_total": 0, "checks_passed": 0, "checks_failed": 0, "checks_skipped": 0 }
```

**Why it is wrong:** SPEC.md section 5.2 shows `inputs` and `checks` as arrays and names
`checks[].status` one of the four fields "frozen for v0.x" that automation is expected to depend
on. A consumer that iterates `report["checks"]` gets a `TypeError` in Python and a silent skip in
`jq`, on the exact runs a monitoring system most needs to read. `"started_at": ""` is not a
timestamp, and `"finished_at": ""` appears even on the restore-failure path where the run demonstrably
started. `tool.commit` is documented in SPEC and elided by `omitempty`, so the field a bug report
most needs is the one missing. And `"exit_code": 2` with an empty `run.id` gives a cron job nothing
to correlate against its logs.

**Proposed fix:** initialise the slices where the report is built
(`internal/runner/runner.go:74`):

```go
rep = &report.Report{
	SchemaVersion: report.SchemaVersion,
	Inputs:        []report.Input{},
	Stages:        []report.Stage{},
	Checks:        []report.Check{},
	...
}
```

set `rep.Run.StartedAt` at the same point rather than after the workspace exists, always set
`FinishedAt` and `DurationMS` on every exit path (move `finish(rep, started)` into the deferred
teardown), and drop `omitempty` from `Tool.Commit` so an unset commit is `""` rather than absent.
Add one sentence to SPEC.md section 5.2: "Array fields are always arrays, never `null`, including
on a run that failed before they could be populated."

### UX-15 (P3) `--keep` prints a POSIX cleanup command and does not say what it kept

**Where:** `internal/cli/check.go:175-177`.

**What the user sees, on Windows:**

```
$ restored check --recipe gitea --keep --keep-on-fail --source dir --from /nonexistent

Kept for inspection:
  workspace:        C:\Users\kadyr\AppData\Local\Temp\restored-2e2ix6sc
  compose project:  restored-2e2ix6sc

Clean up with:
  docker compose -p restored-2e2ix6sc down -v --remove-orphans
  rm -rf C:\Users\kadyr\AppData\Local\Temp\restored-2e2ix6sc
restored: --from "C:/Program Files/Git/nonexistent": GetFileAttributesEx C:/Program Files/Git/nonexistent: The system cannot find the file specified.
EXIT=2
```

The directory is real and stable enough to `cd` into:

```
$ ls C:/Users/kadyr/AppData/Local/Temp/restored-2e2ix6sc
export  inputs  logs  restore  test-assets
```

**Why it is wrong:** three smaller things in one block. `rm -rf` is not a command on the platform
whose path is printed one line above it. The block names the workspace but not what is in it, so
the user does not learn that `inputs/` holds the restored tree and `logs/debug.log` holds every
command the run ran — which is the entire reason to pass `--keep`. And it prints on a run that
failed before compose ever started, so it offers a `docker compose -p ... down` for a project that
never existed. The path itself is fine: absolute, stable, and `cd`-able.

**Proposed fix:** name the contents, and choose the remove command by platform:

```go
rm := "rm -rf " + kept.Workspace
if runtime.GOOS == "windows" {
	rm = `Remove-Item -Recurse -Force "` + kept.Workspace + `"`
}
fmt.Fprintf(human, "\nKept for inspection:\n"+
	"  workspace:        %s\n"+
	"    inputs/           the restored tree, exactly as the stack saw it\n"+
	"    logs/debug.log    every command this run ran, with its output\n"+
	"  compose project:  %s\n\n"+
	"Clean up with:\n  docker compose -p %s down -v --remove-orphans\n  %s\n",
	kept.Workspace, kept.Project, kept.Project, rm)
```

and skip the `docker compose` line entirely when the stack never came up (`composeUp == false`,
which `Kept` should carry).

### UX-16 (P3) Four flag combinations are accepted without complaint

**Where:** `internal/cli/check.go:77-78` (`--keep` and `--keep-on-fail` both settable),
`internal/cli/cli.go:112` (`--log-level` never validated; only "debug" and "trace" are read, at
`internal/cli/check.go:108`), `internal/cli/cli.go:129-140` (`keyValues` silently last-wins on a
repeated key), `internal/cli/recipe.go:96` (`--set` rejection does not list the variables).

**What the user sees:**

```
$ restored --log-level shout check --recipe gitea --source dir --from .
restored: required input "data": no matching files found for /srv/gitea/data in the backup
EXIT=2

$ restored recipe show gitea --inputs-only --input data=/a --input data=/b
restored: input "data": path "B:/" is not absolute; paths inside a backup are absolute POSIX paths
EXIT=2

$ restored recipe show gitea --set nosuchvar=1
restored: --set nosuchvar: recipe "gitea" has no variable "nosuchvar"
EXIT=2
```

**Why it is wrong:** `--log-level shout` is a typo the tool swallows, so a user debugging a hang
with `--log-level dbug` gets silence and concludes the flag does nothing. A repeated `--input data=`
takes the last value with no warning, which on a long cron line is a silent wrong answer.
`--keep --keep-on-fail` together is harmless but means one of the two flags is doing nothing, and
the tool could say which. And the `--set` rejection is the only one of the two "unknown name"
errors that does not list the valid names: the `--input` equivalent says `(has: data, db)` and
`--set` says nothing, though `recipe show` has the list in hand.

**Proposed fix:** validate the log level in a root `PersistentPreRunE`:

```go
switch g.logLevel {
case "trace", "debug", "info", "warn", "error":
default:
	return fail(ExitError, "--log-level %q: expected trace, debug, info, warn or error", g.logLevel)
}
```

reject a repeated key in `keyValues`:

```go
if _, dup := out[k]; dup {
	return nil, fmt.Errorf("--%s %s: given twice; pass each name once", flag, k)
}
```

list the variables in the `--set` error, mirroring `--input`:

```go
return fail(ExitError, "--set %s: recipe %q has no variable %q (has: %s)",
	k, rec.Metadata.Name, k, strings.Join(varNames(rec), ", "))
```

and note in `--keep-on-fail`'s help that `--keep` wins:
`"Tear down on PASS, keep everything on failure (ignored if --keep is given)"`.

### UX-17 (P3) Report width is fixed at 78 columns and `COLUMNS` is ignored

**Where:** `internal/report/tty.go:19` (`const Width = 78`), `internal/harness/render.go:87`
(`truncate(st.Title, 52)`), `internal/nudge/nudge.go:39-42` (`width = 74` when unset, and
`check.go` never sets it).

**What the user sees, at `COLUMNS=40`, unchanged from the default:**

```
$ COLUMNS=40 restored recipe test ./nosuchdir

recipe test ./nosuchdir

  reading recipe "./nosuchdir": GetFileAttributesEx ./nosuchdir: The system
  cannot find the file specified.

  ERROR  ./nosuchdir in 0ms
```

and the harness truncating a stage title that would have fitted a wide terminal:

```
  stage A  negative: the checks must fail against an empty sta… ERROR      1.8s
```

**Why it is wrong:** a fixed width is a defensible choice — it makes the golden files stable and
`docs/demo/*.txt` reproducible, and 78 fits an 80-column terminal. But the consequence is that a
stage title is truncated with an ellipsis at 52 characters on a 200-column terminal where it would
have fitted, and long check titles wrap in the middle of a sentence for readers who did not need
them to. `nudge.Input` already has a `Width` field for exactly this and `internal/cli/check.go:224`
never passes it, so the field is dead. This is a P3 because nothing is unreadable and nothing is
lost except the tail of a stage title, which is repeated in full in the error text below it.

**Proposed fix:** keep 78 as the floor and the default, and widen when the terminal is wider and
output is a TTY. Add to `internal/report/tty.go`:

```go
// Width is the column the report is laid out against. 78 is the floor, so an
// 80-column terminal and the golden files agree; a wider TTY gets a wider report,
// capped so a very wide terminal does not produce unreadable line lengths.
func widthFor(w io.Writer) int {
	const min, max = 78, 110
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return min
	}
	c, _, err := term.GetSize(int(f.Fd()))
	...
}
```

carried on `Options` as a field so `report.WriteTTY`, `harness.WriteTTY` and `nudge.Build` all use
one number, and left at `Width` exactly when output is not a terminal — which keeps
`scripts/capture-demo.sh` and every golden file byte-identical.
