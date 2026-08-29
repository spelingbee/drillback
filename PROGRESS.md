# Progress

**Read this file first.** It is the handover between sessions: what exists, what is
next, what is stuck. Update it at the end of every session, and whenever a claim in it
stops being true.

Rules for this file:

- Every "tests pass" or "it works" claim must name the **exact command** and include the
  **tail of its actual output**. A claim without evidence is a guess, and a guess in a
  handover document is worse than a gap.
- Convert relative dates to absolute ones. "last week" is meaningless to the next
  session.
- Do not delete history. Append to the session log; move items between *Current state*,
  *Next steps* and *Blocked* rather than dropping them.

---

## Current state

**Phase:** the core works. `restored check` runs end to end against a real restic
repository and against an already-restored tree, with two bundled recipes.
**Version:** unreleased. No tags. `0.1.0-dev` is what the binary reports.
**Module:** `github.com/spelingbee/restored` (ADR-036 - see *Open questions*, the name
is still a human's to confirm and the rename is one grep).
**Language of record:** English, everywhere, and now enforced by
`scripts/lint-english.sh`.

What works, and is proved by a command below:

| Capability | State |
|---|---|
| `restored check --recipe <bundled\|dir> --source restic --from <repo>` | works, PASS and RESTORE UNUSABLE both reached against real stacks |
| `restored check --source dir --from <tree>` | works |
| `restored recipe validate [--strict] [--json]` | works, schema + safety schema + the three Go rules |
| `restored recipe show [--format] [--compose] [--inputs-only]` | works |
| `restored recipe init` | works; the scaffolded recipe validates as it comes out |
| `restored version [--json]` | works, and exits 0 with docker and restic absent |
| Isolation | enforced: no privileged, no host namespaces, no published ports, no bind outside the workspace, internal networks only |
| Report | TTY renderer with an ASCII fallback and `NO_COLOR`, plus the JSON document of SPEC.md 5.2 |
| Hints | 17 rules, embedded, `--hints FILE` for extra rules matched first |
| Nudge | built, and printed only when all five conditions in SPEC.md 8.1 hold |
| Teardown | `compose down -v --remove-orphans` plus the workspace, on every exit path |

What does **not** work yet, deliberately:

- **`restored recipe test`** - the round-trip harness of SPEC.md section 7. The command
  exists, says it is not implemented, and exits 2 (ADR-046). This is the biggest hole:
  it is the piece that lets a stranger's recipe be trusted without a maintainer reading
  it, and until it lands a recipe is proved only by `scripts/demo.sh`.
- **`restored.yaml`, `--target`, `--all`, `--config`** - `internal/config` is not
  written and the flags are not registered, so an invocation using one fails loudly
  rather than silently doing nothing (ADR-045). `restored check --help` therefore does
  not match SPEC.md section 2.
- **The four other recipes** for the v0.1 gate of six (ADR-033): Vaultwarden,
  Paperless-ngx, Miniflux, Nextcloud.
- **CI beyond lint and unit.** `ci.yml` has the lint and unit jobs. `recipes.yml`,
  `smoke.yml`, `recipe-health.yml` and `release.yml` are unwritten, and the integration
  job is not wired into CI even though the tests exist and pass locally.
- **`CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `docs/recipes.md`,
  `docs/security.md`, `install.sh`.** README.md exists.

Repository layout now matches SPEC.md section 13, plus `internal/runner` (ADR-038),
`internal/sqlite`, and `assets.go` at the root (ADR-037).

---

## Session log

### Session 1 — 2026-08-30 — Specification

**Goal:** produce the specification and the project's working documents. No application
code.

**Done:**

1. **Name check.** Queried GitHub (authenticated `gh search repos` + `gh api users/…`),
   npm, PyPI, crates.io, Homebrew formula and cask APIs, Docker Hub user and library
   endpoints, and RDAP/DNS for `.dev`, `.sh`, `.io`, across all five candidate names.
   Findings written to `docs/name-check.md` with a ranked recommendation. Nothing was
   renamed.

   Three findings drove the recommendation:
   - `restore-drill` is one hyphen from [`ahmadpiran/restoredrill`](https://github.com/ahmadpiran/restoredrill)
     — 87 stars, Go, MIT, actively maintained, and doing very nearly the same thing for
     PostgreSQL. Rejected.
   - `restored` is legally and technically usable — no exact repo collision above 100
     stars, and the Homebrew formula name is free — but the GitHub user, the Docker Hub
     user, the npm package, `restored.dev` and `restored.io` are all taken, and the word
     reads as a Unix daemon (`restore` + `d`; `restored` is a real Apple iOS daemon).
     Usable, with a discoverability cost.
   - `drillback` is the only candidate clean on every registry and all three TLDs.

2. **`SPEC.md`.** Written in full: exact `--help` output for every command; a
   `restored.yaml` with four targets and two restic sources plus a cron line; complete
   `recipe.yaml` + `compose.yaml` for Gitea + PostgreSQL and for Uptime Kuma (SQLite),
   with seed SQL; the recipe JSON Schema and a separate compose-safety schema encoding
   the isolation rules; the run lifecycle as an eight-state machine with a per-state
   budget and exit-code table; TTY report mocks for PASS and RESTORE UNUSABLE, both
   labelled as mocks that must never be copied into the README; the JSON report with a
   stability contract; a 16-rule `hints.yaml`; the round-trip harness including the
   throwaway restic setup, the 20-minute budget and the CI recipe-selection shell; the
   nudge with its 6,000-character fallback; the threat model split into mitigated,
   accepted, and explicit non-mitigations; the testing pyramid; five CI workflows with
   runtime budgets; the release process; the repository tree with package-boundary
   rules; and the v0.2/v0.3 roadmap.

3. **`DECISIONS.md`.** 35 ADRs. The 12 fixed decisions recorded verbatim in intent, plus
   23 decisions made while writing the spec — every point where the brief left a choice.

4. **`PROGRESS.md`** and **`CLAUDE.md`** written.

5. `git init`, `LICENSE` (Apache-2.0, fetched from apache.org and the appendix copyright
   line filled in), `.gitignore`, `.editorconfig`.

**Not done, deliberately:** no application code, no `go.mod`, no `recipes/`, no
`schema/`, no workflows. Session 1's constraint.

**Evidence.** There is no code, so there is no build or test claim. The documents
themselves were machine-checked, and the checks found and fixed one real bug.

Every fenced JSON and YAML block in SPEC.md was parsed; both schemas were checked as
draft 2020-12; both example recipes were validated against `recipe.schema.json` and both
example compose files against `compose-safety.schema.json`; and twelve synthetic
dangerous compose files were checked to confirm the safety schema rejects them.

```text
$ python - <<'EOF'   # extract the fenced blocks from SPEC.md and validate them
[schema] recipe.schema: valid draft 2020-12
[schema] compose-safety.schema: valid draft 2020-12
[valid]  gitea/recipe.yaml
[valid]  uptime-kuma/recipe.yaml
[valid]  gitea/compose.yaml
[valid]  uptime-kuma/compose.yaml

negative controls (each MUST be rejected):
  rejected   privileged
  rejected   ports
  rejected   network_mode host
  rejected   pid host
  rejected   host bind mount
  rejected   long bind mount
  rejected   non-internal net
  rejected   image :latest
  rejected   cap_add SYS_ADMIN
  rejected   build
  rejected   devices
  rejected   seccomp unconf

all dangerous constructs rejected: True
```

**Bug found and fixed by this check:** the first draft of `$defs/internalURL` rejected
every example recipe, because its pattern allowed only a numeric port and every recipe
writes `http://gitea:{{ .vars.gitea_port }}/`. The fix was to permit `{{ ... }}` in the
port and path, and it exposed a design point that was implicit and is now written down
as SPEC.md § 3.4.1: the **recipe** schema validates the file *as written*, with
templates unexpanded, so that editors and the independent-validator CI job can use it,
while the **compose-safety** schema validates *after* interpolation, because its whole
purpose is to inspect resolved values.

The 16 hint rules were also checked: unique ids, all four required fields present, every
`match` compiles as a regular expression, and the error string in the RESTORE UNUSABLE
mock (SPEC.md § 5.1) is matched first by `postgres/relation-missing` — the rule that
mock claims fires. Rule ordering and the mock agree.

```text
hint rules: 16
all rules have id/match/title/text and compile: True
first rule matching the RESTORE-UNUSABLE mock error: postgres/relation-missing
spec claims: postgres/relation-missing -> MATCH
```

Note for session 2: this validation was ad-hoc Python. It must become the `schema` CI
job (SPEC.md § 11.1) and the hint fixture tests (§ 10.1), against the extracted files
rather than against fenced blocks in a Markdown document.

Other commands run this session were read-only research (`gh search repos`, `gh api`,
`curl` against public registry APIs, `nslookup`) plus `git init` and file writes.

### Session 2 - 2026-08-30 - The core

**Goal:** `restored check` end to end, two recipes, a real demo, tests, a README.

**Environment.** Windows 11 host, Docker Desktop 29.5.2 with the WSL2 Linux engine,
docker compose v5.1.3. Go 1.27.0 from a portable toolchain at
`C:\My\Projects\Work\gotool\go`, which is not on the default PATH: every command below
was run with `export PATH="/c/My/Projects/Work/gotool/go/bin:/c/Users/kadyr/go/bin:$PATH"`.
`restic 0.19.1` and `golangci-lint 2.13.2` were installed into `~/go/bin` during this
session (see *Toolchain* below).

The preconditions in the brief asked for `docker compose version` to report v2. It
reports **v5.1.3**, which is Docker Desktop's current compose. Every command this project
uses - `up -d --pull`, `exec -T`, `logs --tail`, `down -v --remove-orphans`, `config
--services` - behaves as specified, and the whole suite passes against it.

**Done, in order:**

1. **Scaffold.** `go.mod` (ADR-036), the package tree, `Makefile`, `.golangci.yml`,
   `.gitattributes`, `.goreleaser.yaml`, `.github/workflows/ci.yml`.
2. **Extraction, not retyping.** `schema/recipe.schema.json`,
   `schema/compose-safety.schema.json`, `docs/hints.yaml`, `recipes/gitea/**` and
   `recipes/uptime-kuma/**` were pulled out of SPEC.md's fenced blocks by a script, so
   the specification and the shipped files could not disagree on day one (ADR-034).
3. **`internal/recipe`** - types, loader, YAML-tag rejection, JSON Schema validation,
   the restricted template context, and input resolution including `within:`.
4. **`internal/recipe/safety`** - the compose safety schema and the three Go-only rules,
   plus interpolation and the resolved-mount containment check.
5. **`internal/workspace`, `internal/compose`, `internal/source/{restic,dir}`,
   `internal/loader`, `internal/sqlite`, `internal/check`, `internal/probe`,
   `internal/report`, `internal/hints`, `internal/nudge`, `internal/runner`,
   `internal/cli`.**
6. **The demos.** `scripts/demo.sh`, `scripts/demo-broken.sh`, `scripts/demo-kuma.sh`,
   `scripts/capture-demo.sh`, `scripts/lib.sh`, `scripts/lint-english.sh`.
7. **README.md**, with its terminal blocks spliced in from `docs/demo/*.txt` by
   `capture-demo.sh`. Nothing in it is hand-written.
8. **Tests.** Unit tests for the schema, resolution, safety, snapshot selection, the
   report, the hints, the expect vocabulary, dump detection, sanitisation and the
   scaffold. Integration tests behind the `integration` tag for the runner and for all
   three demos.

**Five things reality contradicted the specification about**, each fixed in both places
and each with an ADR: the compose safety schema cannot run after interpolation
(ADR-039); a dump must be loaded before the application first connects (ADR-041); the
Gitea recipe's data mount and repository path were wrong (ADR-042); `PGOPTIONS` with
postmaster settings breaks the postgres image's own initialisation; and Uptime Kuma
answers `/` with a 302.

**Two bugs the tests found**, both silent in normal use: templating skipped map values
entirely, so `inputs.db.load.user` reached psql as the literal `{{ .vars.db_user }}`;
and a nested input kept the recipe's default path when its parent was overridden, so
`--input data=/mnt/backup` moved the directory and left the database behind.

---

#### Evidence

Unit suite, with the race detector. The host has no C toolchain, so `-race` cannot run
on it (`cgo: C compiler "gcc" not found`); it was run in the same Linux container image
CI will use.

```text
$ docker run --rm -v "C:/My/Projects/Work/restored:/src" \
    -v "C:/Users/kadyr/go/pkg/mod:/go/pkg/mod" -w /src golang:1.27 go test ./... -race
ok  	github.com/spelingbee/restored/internal/check	1.163s
ok  	github.com/spelingbee/restored/internal/cli	1.241s
ok  	github.com/spelingbee/restored/internal/hints	1.147s
ok  	github.com/spelingbee/restored/internal/loader	1.164s
ok  	github.com/spelingbee/restored/internal/recipe	1.324s
ok  	github.com/spelingbee/restored/internal/recipe/safety	1.210s
ok  	github.com/spelingbee/restored/internal/report	1.180s
ok  	github.com/spelingbee/restored/internal/source/restic	1.050s
ok  	github.com/spelingbee/restored/internal/workspace	1.068s
```

Full suite including integration, on the host, against the real daemon.
`internal/runner` is 269 seconds because it stands up Gitea, PostgreSQL and Uptime Kuma
for real, five times.

```text
$ go test -tags integration ./... -timeout 40m
ok  	github.com/spelingbee/restored/internal/check	2.854s
ok  	github.com/spelingbee/restored/internal/cli	1.644s
ok  	github.com/spelingbee/restored/internal/hints	2.341s
ok  	github.com/spelingbee/restored/internal/loader	3.875s
ok  	github.com/spelingbee/restored/internal/recipe	2.320s
ok  	github.com/spelingbee/restored/internal/recipe/safety	3.158s
ok  	github.com/spelingbee/restored/internal/report	2.810s
ok  	github.com/spelingbee/restored/internal/runner	269.612s
ok  	github.com/spelingbee/restored/internal/source/restic	1.503s
ok  	github.com/spelingbee/restored/internal/workspace	1.406s
```

Workspace sanitisation - the security-critical path - **skips on this host**, because
Windows will not create a symlink for an unprivileged user. It was cross-compiled and
run on Linux to make sure it is actually covered:

```text
$ CGO_ENABLED=0 GOOS=linux go test -c -o /tmp/workspace.test ./internal/workspace
$ docker run --rm -v /tmp:/t alpine:3.20 /t/workspace.test -test.v
--- PASS: TestNewCreatesTheTree (0.00s)
--- PASS: TestRunIDIsUsableEverywhere (0.00s)
--- PASS: TestContains (0.00s)
--- PASS: TestSanitiseNeutralisesEscapingSymlinks (0.01s)
--- PASS: TestSanitiseRefusesToLeaveTheWorkspace (0.00s)
--- PASS: TestMeasure (0.00s)
--- PASS: TestCopyTree (0.00s)
PASS
```

Lint:

```text
$ gofmt -l .
$ go vet ./...
$ golangci-lint run
0 issues.
$ golangci-lint run --build-tags integration
0 issues.
$ ./scripts/lint-english.sh
lint-english: ok
```

The bundled recipes, against the schema and the safety rules:

```text
$ ./bin/restored recipe validate ./recipes/gitea ./recipes/uptime-kuma --strict
ok       ./recipes/gitea
ok       ./recipes/uptime-kuma
$ echo $?
0
```

The demos, run twice in a row to prove they are idempotent, and the broken one's exit
code checked:

```text
$ ./scripts/demo.sh          ; echo exit=$?     # run 1
exit=0
$ ./scripts/demo.sh          ; echo exit=$?     # run 2
exit=0
$ ./scripts/demo-broken.sh   ; echo exit=$?
exit=1
$ ./scripts/demo-kuma.sh     ; echo exit=$?
exit=0
$ docker ps -aq --filter "label=com.restored.run" | wc -l
0
$ ls -d "$TMPDIR"/restored-* 2>/dev/null || echo "no workspaces left"
no workspaces left
```

The reports those runs printed are in `docs/demo/pass.txt`, `docs/demo/fail.txt` and
`docs/demo/kuma.txt`, captured by `scripts/capture-demo.sh` and spliced into README.md
by the same script. Nothing there was typed by hand.

---

#### Toolchain

Recorded here because the next session on this machine will need it, and because
CLAUDE.md's command list assumes it.

- **Go 1.27.0** - portable toolchain already present at
  `C:\My\Projects\Work\gotool\go\bin`. Not on PATH by default.
- **restic 0.19.1** - downloaded from the GitHub release
  (`gh release download v0.19.1 --repo restic/restic --pattern
  'restic_0.19.1_windows_amd64.zip'`), unzipped, and copied to
  `C:\Users\kadyr\go\bin\restic.exe`.
- **golangci-lint 2.13.2** - `go install
  github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`.
- **No C toolchain**, so `go test -race` needs the container command shown above.
- **No `make`.** The Makefile is correct and is what CI runs; on this host, type the
  command a target wraps. CLAUDE.md lists the four that matter.

Pinned images the project pulls: `curlimages/curl:8.16.0` (the check helper),
`gitea/gitea:1.22.6`, `postgres:16.4-alpine`, `louislam/uptime-kuma:1.23.16-alpine`,
`keinos/sqlite3:3.46.0`, `restic/restic:0.19.1`, `nginx:1.27-alpine` (the test fixture).

---

#### A mistake, recorded

An over-broad `git add -A` while installing restic committed
`restic_0.19.1_windows_amd64.zip` and the 31 MB binary it contained into the first two
commits of this session. They were removed with `git filter-branch` over
`35a3e83..HEAD` before anything was pushed, the reflog was expired and the repository
garbage collected (`.git` is 286 KB), and `.gitignore` now covers both paths. Nothing
was ever published. Check with `git log --stat | grep -i restic_` if in doubt.


---

## Next steps

In order. Each is sized to be finishable and committable on its own.

### Session 3 - the round-trip harness

`internal/harness` and `restored recipe test`, exactly as SPEC.md section 7 describes
it. This is the largest remaining hole and the one the contribution flow rests on:
stage A must prove a recipe's checks fail against an empty stack, stage B must prove
they pass against a real round trip. Both bundled recipes already have the `test:`
section the harness needs; it is parsed and validated today and nothing executes it.

Note for whoever writes it: `${RESTORED_TEST_ASSETS}` and `${RESTORED_EXPORT}` already
resolve to `<workspace>/test-assets` and `<workspace>/export`, which exist and are
empty during `check`. The harness has to copy the recipe's `test/` directory into the
first one, and collect `produces:` outputs out of the second.

### Session 4 - configuration and the rest of the CLI surface

`internal/config`: `restored.yaml`, sources, targets, the precedence chain, `--config`,
`--target`, `--all`, and the `--all` report shape. Then diff `--help` against SPEC.md
section 2 and fix whichever of the two is wrong (ADR-045).

### Session 5 - CI, and the four remaining recipes

`recipes.yml` with the changed-recipe selection of SPEC.md 7.5, the integration job in
`ci.yml`, `smoke.yml`, and `recipe-health.yml`. Then Vaultwarden, Paperless-ngx,
Miniflux and Nextcloud, to reach the six-recipe gate (ADR-033). Nextcloud is the one
that will test whether the isolation rules hold.

### Then

`CONTRIBUTING.md` (the README already points at it), `SECURITY.md`, `docs/recipes.md`,
`install.sh`, the ghcr image, and the release checklist. None of the publishing steps
happen without a human: see CLAUDE.md's stop points.

---

## Blocked

| # | Item | Since | Blocking | Needed from |
|---|---|---|---|---|
| - | Nothing. | - | - | - |

B-1, the project name and the GitHub owner, was resolved by this session under the
brief's instruction to decide rather than ask, and is recorded as ADR-036. It is listed
below as an open question rather than as a blocker, because the code exists and the
rename is mechanical.

---

## Open questions for a human

1. **The name and the owner, still.** `github.com/spelingbee/restored` is in `go.mod`,
   in both schema `$id`s, and in `internal/nudge`. `docs/name-check.md` recommends
   `drillback` because it is the only candidate clean on every registry and all three
   TLDs, and `restored` costs discoverability. Changing it now is
   `grep -rl spelingbee/restored | xargs sed -i` plus a `go mod edit`; changing it after
   anything is published is not. This is the cheapest hour this project will ever have
   to spend. See ADR-036.
2. **ADR-023 - is a failed ready probe exit 1 or exit 2?** Unchanged from session 1, and
   now implemented as exit 1. Session 2 saw one real instance of it: an Uptime Kuma
   ready probe that failed because the recipe expected a 200 and the application answers
   a 302. That was a recipe bug reported as an unusable restore, which is exactly the
   false alarm the question is about. The recipe is fixed; the question stands.
3. **ADR-030 - should digest pinning be required rather than encouraged?** Unchanged.
4. **ADR-033 - are six recipes the right gate for v0.1?** Two exist and both round-trip.
5. **Is Nextcloud achievable inside the isolation rules?** Still unknown, and still the
   recipe most likely to force an exception.
6. **New: `restored recipe test` is not built.** Until it is, a recipe cannot be
   accepted from a stranger on evidence alone, which is the mechanism the success metric
   depends on. Session 3 is the whole answer, and nothing else should jump the queue.
