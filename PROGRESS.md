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

**Phase:** reviewed, hardened, and ready to release - up to the tag, which is a human's
(stop point 1). Five independent reviewers went through the repository before strangers
could; all 9 P0 and all 21 P1 findings are fixed; the remaining 38 are written up as
`help wanted` issues waiting for the repository to be public.
**Version:** unreleased. No tags. `0.1.0-dev` is what a local build reports; the release
version comes from the tag through ldflags and from nowhere else.
**Module:** `github.com/spelingbee/restored` (ADR-036 - see *Open questions*, the name
is still a human's to confirm and the rename is one grep).
**Language of record:** English, everywhere, enforced by `scripts/lint-english.sh`.

What works, and is proved by a command below:

| Capability | State |
|---|---|
| `restored check --recipe <bundled\|dir> --source restic --from <repo>` | works, PASS and RESTORE UNUSABLE both reached against real stacks |
| `restored check --source dir --from <tree>` | works |
| `restored recipe validate [--strict] [--json]` | works, schema + safety schema + the three Go rules |
| `restored recipe show [--format] [--compose] [--inputs-only]` | works |
| `restored recipe init` | works; the scaffolded recipe validates as it comes out |
| `restored recipe init --compose <file>` | works; reads a real compose file and proposes a recipe that validates |
| **`restored recipe test [--stage a\|b\|both] [--keep] [--timeout] [--report] [--json]`** | **works; all twenty recipes pass both stages** |
| `restored version [--json]` | works, and exits 0 with docker and restic absent |
| Isolation | enforced: no privileged, no host namespaces, no published ports, no bind outside the workspace, internal networks only |
| Report | TTY renderer with an ASCII fallback and `NO_COLOR`, plus the JSON document of SPEC.md 5.2; the harness has its own report and its own schema version |
| Hints | 18 rules, embedded, `--hints FILE` for extra rules matched first; matched on every exit path since ADR-058 |
| Nudge | built, tested, printed only when all five conditions in SPEC.md 8.1 hold; `--no-nudge` and `defaults.nudge: false` both silence it |
| Teardown | `compose down -v --remove-orphans` plus the workspace, on every exit path |
| CI | `ci.yml` (lint, generated-file diff, unit on three platforms, integration), `recipes.yml` (changed recipes, one verdict each), `recipe-health.yml` (weekly, opens and closes issues), `release.yml` (goreleaser skeleton, draft only) |
| Contributor path | `CONTRIBUTING.md`, `recipes/TEMPLATE`, four issue templates, a PR checklist, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CODEOWNERS`, dependabot |

Bundled recipes, **twenty** of them since session 5: **beszel**, **changedetection**,
**convertx**, **filebrowser**, **freshrss**, **gitea**, **gogs**, **gotify**,
**listmonk**, **mealie**, **memos**, **n8n**, **navidrome**, **nextcloud**,
**open-webui**, **paperless-ngx**, **siyuan**, **trilium**, **uptime-kuma**,
**vaultwarden**. `recipes/TEMPLATE` ships in
the binary too but is deliberately not in the registry: `BundledNames` skips any
directory whose name is not a legal recipe name.

What does **not** work yet, deliberately:

- **`restored.yaml`, `--target`, `--all`, `--config`** - `internal/config` is not
  written and the flags are not registered, so an invocation using one fails loudly
  rather than silently doing nothing (ADR-045). `restored check --help` therefore does
  not match SPEC.md section 2. The one exception is `defaults.nudge`, which
  `internal/nudge` reads through a deliberately narrow one-key reader so that a user
  who has written `nudge: false` is believed today.
- **`smoke.yml`**, the fresh-clone test of SPEC.md 11.3. The `unit` job proves
  `go test ./...` is green with no docker and no restic, which is most of what it was
  for; session 4's fresh-clone review walked the rest of it by hand.
- **`docs/recipes.md`, `docs/security.md`.** `CHANGELOG.md`, `install.sh`,
  `docs/docker.md`, `docs/homebrew-tap.md`, `docs/release-checklist.md` and
  `docs/recipe-spec.md` all exist now.
- **The 38 P2 and P3 findings** in `docs/review/backlog.md`. Deliberately not fixed:
  they are the contributor entry points, and `scripts/backlog-issues.sh` files them the
  hour the repository goes public.
- **Nothing is published.** No tag, no image pushed, no tap, no issues filed, no labels
  created, and the repository is not public. Those are stop points; see CLAUDE.md and
  `docs/release-checklist.md`, which is the ordered list.

Repository layout now matches SPEC.md section 13, plus `internal/runner` (ADR-038),
`internal/sqlite`, `internal/harness`, `tools/gen`, and `assets.go` at the root
(ADR-037).

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


### Session 3 - 2026-08-30 - The round trip, the recipes, and the contributor path

**Goal:** `restored recipe test`, `recipe init --compose`, three more recipes, CI, and
everything a stranger needs to go from "no recipe for my app" to a merged pull request
in one sitting.

**Environment.** Unchanged from session 2: Windows 11, Docker Desktop 29.5.2 with the
WSL2 engine, compose v5.1.3, Go 1.27.0 from `C:\My\Projects\Work\gotool\go`, restic
0.19.1 and golangci-lint 2.13.2 in `~/go/bin`. Every command below was run with

```sh
export PATH="/c/My/Projects/Work/gotool/go/bin:/c/Users/kadyr/go/bin:$PATH"
```

One thing worth writing down for the next session on this machine: any `docker run`
with an absolute container path needs `export MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'`
first, or Git Bash rewrites `-w /src` into `C:/Program Files/Git/src` before the daemon
sees it. CLAUDE.md now says so.

**Done, in order:**

1. **`internal/harness` and `restored recipe test`.** Stage A runs the ordinary `check`
   code path against a tree of empty inputs and requires that a check fails. Stage B
   starts a fresh stack, seeds it, exports what a backup would have taken, puts that
   into a throwaway restic repository, tears everything down, and runs an ordinary
   `restored check` against it. Per-phase timing, `--stage`, `--keep`, `--timeout`,
   `--report`, `--json`, and guaranteed cleanup.
2. **`restored recipe init --compose <file>`.** Reads a real compose file and proposes a
   recipe from it.
3. **The nudge** got the six tests it never had, and `defaults.nudge: false`.
4. **Three recipes**: vaultwarden, paperless-ngx, nextcloud. Plus `recipes/TEMPLATE`
   and a README for all five.
5. **CI**: `ci.yml` gained the integration job and a generated-file diff;
   `recipes.yml`, `recipe-health.yml` and `release.yml` are new.
6. **Contributor infrastructure**: `CONTRIBUTING.md`, `SECURITY.md`,
   `CODE_OF_CONDUCT.md`, `CODEOWNERS`, `.github/dependabot.yml`, four issue templates
   plus `config.yml`, a pull request template, `.all-contributorsrc`.
7. **Generated documentation**: `tools/gen` writes `docs/recipe-spec.md` from the JSON
   Schema and `recipes/README.md` from the registry, and splices the recipe table into
   `README.md`. CI fails on a diff.
8. **Launch tooling, none of it executed against a live repository**:
   `docs/recipes-wanted.txt`, `scripts/recipes-wanted.sh`, `scripts/labels.sh`,
   `scripts/contributors.sh`.

---

#### Six things reality contradicted the plan about

Each was measured, each is fixed in the code and in the document that was wrong, and
each has an ADR.

1. **restic has no path-rewriting flag.** `restic backup` resolves every argument to an
   absolute path on the machine it runs on, so a snapshot of the staging tree records
   `C:/Users/.../staging/srv/gitea/data` and stage B's `restored check` finds nothing at
   `/srv/gitea/data`. The harness now runs `restic/restic:0.19.1` in a container with
   each staged input bind-mounted at *its own* `default_path`, so the snapshot records
   exactly what the recipe declares and step 8 needs no `--input` override. ADR-051.

2. **Stage B cannot start from stage A's empty inputs.** SPEC.md 7.3 said "EMPTY inputs
   (same as stage A)". A zero-length `kuma.db` is a perfectly good empty *restore* and a
   hopeless empty *start*: Uptime Kuma crash-loops instead of running its migrations,
   and the ready probe burned its full budget three times before this was understood.
   Stage B now creates only the dir inputs and any non-dir input that compose
   bind-mounts. ADR-053, and SPEC.md 7.3 corrected.

3. **A container runs as a uid that is not the caller's.** On Linux a bind mount carries
   the host's permissions straight through, so a 0755 directory owned by the caller is a
   directory Gitea (uid 1000) or Nextcloud (uid 33) cannot write. Invisible on Windows,
   which ignores the mode; it would have been a CI-only failure. Empty inputs are now
   created 0777, and the workspace root was tightened from 0755 to **0700** to make that
   safe. ADR-054.

4. **A restored tree carries the ownership of the machine the backup came from.**
   Measured: a restored Nextcloud data directory arrives as `drwxrwx--- root root`, and
   Nextcloud answers 503 "your data directory is not writable". That is a fact about uid
   mapping, not about the backup, and reporting it as an unusable restore would be
   exactly the false alarm this tool exists to remove. `restored` now opens the modes of
   every restored input after sanitising it. ADR-055.

5. **The exit codes in SPEC.md 2.7 contradicted the brief.** A recipe whose checks all
   pass against an empty stack has not failed a test; it is not a test. That is the same
   class of defect as a recipe that does not match the schema, so it exits 2 with
   `recipe has no data-sensitive check`, and only a stage B failure exits 1. ADR-052,
   and SPEC.md 2.7 corrected.

6. **Vaultwarden refuses to start with an empty `SMTP_HOST`.** "Both `SMTP_HOST` and
   `SMTP_FROM` need to be set." An empty value is a value. Absent means "no mail".

---

#### Three things measured rather than assumed

Each of these would have shipped as a recipe that looks right and proves nothing.

- **Paperless-ngx creates two accounts before anybody signs up.** `consumer` and
  `AnonymousUser`, on a completely fresh install. `SELECT count(*) FROM auth_user` is
  therefore *not* a data-sensitive check, and stage A said so: it passed when it should
  have failed. The recipe now excludes both by name, and says why.

  ```text
  $ docker compose -p restored-uhvswlz2 exec -T db psql -tAq -U paperless -d paperless \
      -c "select id,username,is_superuser,is_active from auth_user;"
  1|consumer|f|t
  2|AnonymousUser|f|t
  ```

- **Nextcloud's installer creates its own database role.** Given a superuser it makes
  `oc_<admin user>`, which then owns every table, and a plain `pg_dump` carries
  `ALTER ... OWNER TO oc_drilluser` into a database that has never heard of it. The
  export step uses `--no-owner --no-acl`; the recipe README says the same about your own
  dump; and `restored`'s existing `postgres/role-missing` hint already explains it.

- **`not_empty` counts directory entries.** It is never met by a regular file, so
  `exists: true, not_empty: true` on `rsa_key.pem` failed in both stages for a reason
  that had nothing to do with the restore. `size_min: 1` is the file-shaped
  expectation. Now stated in `docs/recipe-spec.md`, which is generated, so it cannot
  drift.

---

#### A mistake, recorded

`.all-contributorsrc` used emoji for two custom contribution types. `lint-english.sh`
scans what git *tracks*, so the check passed while the file was untracked and went red
the moment the commit added it — and the commit was made before that was noticed. The
next commit says so in its subject line rather than quietly amending it. The custom
types are gone; the built-in ones carry their own symbols.

---

#### Evidence

**The round trip, against every bundled recipe.** This is the whole session in one
command, and it is what `recipes.yml` runs one recipe per matrix job.

The report also prints, under each stage, the exact `restored check` command that stage
ran, so that any line of it can be reproduced by hand. Those lines are elided here and
nothing else is:

```text
$ ./bin/restored recipe test ./recipes/gitea ./recipes/nextcloud ./recipes/paperless-ngx \
    ./recipes/uptime-kuma ./recipes/vaultwarden
recipe test gitea (Gitea + PostgreSQL)
  stage A  negative: the checks must fail against an empty sta… PASS      25.3s
           4 of 5 checks failed against an empty stack: repos-in-db,
  stage B  round trip: seed, export, back up, restore, check    PASS      55.3s
           the round trip restored and all 5 checks passed
           up 2.3s · ready 8.0s · seed 4.3s · export 3.9s · restic 6.9s · down 2.8s · check 26.9s
  PASS   gitea in 1m21s
recipe test nextcloud (Nextcloud (PostgreSQL))
  stage A  negative: the checks must fail against an empty sta… PASS      31.4s
           3 of 6 checks failed against an empty stack: instance-installed,
  stage B  round trip: seed, export, back up, restore, check    PASS      1m21s
           the round trip restored and all 6 checks passed
           up 3.8s · ready 9.8s · seed 24.6s · export 3.4s · restic 6.8s · down 3.7s · check 28.8s
  PASS   nextcloud in 1m53s
recipe test paperless-ngx (Paperless-ngx (PostgreSQL + Redis))
  stage A  negative: the checks must fail against an empty sta… PASS      55.7s
           2 of 5 checks failed against an empty stack: users-present,
  stage B  round trip: seed, export, back up, restore, check    PASS      1m37s
           the round trip restored and all 5 checks passed
           up 2.3s · ready 31.0s · seed 6.0s · export 2.0s · restic 5.9s · down 8.7s · check 41.8s
  PASS   paperless-ngx in 2m34s
recipe test uptime-kuma (Uptime Kuma (SQLite))
  stage A  negative: the checks must fail against an empty sta… PASS      1m33s
           PASS-BY-STARTUP-REFUSAL: ready: kuma serves HTTP: curl: (7) Failed
           to connect to kuma port 3001 after 1 ms: Could not connect to
           server
  stage B  round trip: seed, export, back up, restore, check    PASS      53.5s
           the round trip restored and all 6 checks passed
           up 2.4s · ready 4.9s · seed 738ms · export 662ms · restic 6.0s · down 21.1s · check 17.7s
  PASS   uptime-kuma in 2m27s
recipe test vaultwarden (Vaultwarden (SQLite))
  stage A  negative: the checks must fail against an empty sta… PASS       6.6s
           1 of 5 checks failed against an empty stack: accounts-present
  stage B  round trip: seed, export, back up, restore, check    PASS      19.5s
           the round trip restored and all 5 checks passed
           up 2.0s · ready 849ms · seed 990ms · export 665ms · restic 6.0s · down 1.4s · check 7.6s
  PASS   vaultwarden in 26.9s
  5 recipes: 5 passed, 0 failed, 0 errored, in 8m43s

$ echo $?
0
$ docker ps -aq --filter "label=com.restored.run" | wc -l
0
$ ls -d "$TMPDIR"/restored-* 2>/dev/null || echo "no workspaces left"
no workspaces left
```

Round-trip time per recipe, both stages, on this machine:

| recipe | total | stage A | stage B | stage A verdict |
|---|---|---|---|---|
| vaultwarden | 26.9s | 6.6s | 19.5s | 1 of 5 checks failed |
| gitea | 1m21s | 25.3s | 55.3s | 4 of 5 checks failed |
| nextcloud | 1m53s | 31.4s | 1m21s | 3 of 6 checks failed |
| uptime-kuma | 2m27s | 1m33s | 53.5s | PASS-BY-STARTUP-REFUSAL |
| paperless-ngx | 2m34s | 55.7s | 1m37s | 2 of 5 checks failed |
| **all five** | **8m43s** | | | |

Uptime Kuma's stage A is 93 seconds of a ready probe waiting for a Kuma that was never
going to start: given a zero-length `kuma.db` it crash-loops rather than starting
empty. SPEC.md 7.2 calls that PASS-BY-STARTUP-REFUSAL and accepts it, and the report
names the stage that refused rather than merely reporting a pass.

Every recipe is inside the 15-minute budget `recipes.yml` gives a matrix job, the
slowest at a sixth of it.

**Full suite including integration, on the host, against the real daemon.**

```text
$ go test -tags integration ./... -timeout 40m
ok  	github.com/spelingbee/restored/internal/check	2.521s
ok  	github.com/spelingbee/restored/internal/cli	0.975s
ok  	github.com/spelingbee/restored/internal/harness	0.727s
ok  	github.com/spelingbee/restored/internal/hints	(cached)
ok  	github.com/spelingbee/restored/internal/loader	2.433s
ok  	github.com/spelingbee/restored/internal/nudge	1.451s
ok  	github.com/spelingbee/restored/internal/recipe	2.202s
ok  	github.com/spelingbee/restored/internal/recipe/safety	2.153s
ok  	github.com/spelingbee/restored/internal/report	2.352s
ok  	github.com/spelingbee/restored/internal/runner	252.438s
ok  	github.com/spelingbee/restored/internal/source/restic	(cached)
ok  	github.com/spelingbee/restored/internal/workspace	0.706s
```

**Unit suite with the race detector.** The host has no C toolchain, so this runs in the
image CI uses.

```text
$ export MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*"
$ docker run --rm -v "C:/My/Projects/Work/restored:/src" \
    -v "C:/Users/kadyr/go/pkg/mod:/go/pkg/mod" -w /src golang:1.27 go test ./... -race
ok  	github.com/spelingbee/restored/internal/check	1.217s
ok  	github.com/spelingbee/restored/internal/cli	1.567s
ok  	github.com/spelingbee/restored/internal/harness	1.294s
ok  	github.com/spelingbee/restored/internal/hints	1.190s
ok  	github.com/spelingbee/restored/internal/loader	1.225s
ok  	github.com/spelingbee/restored/internal/nudge	1.073s
ok  	github.com/spelingbee/restored/internal/recipe	1.622s
ok  	github.com/spelingbee/restored/internal/recipe/safety	1.339s
ok  	github.com/spelingbee/restored/internal/report	1.294s
ok  	github.com/spelingbee/restored/internal/source/restic	1.119s
ok  	github.com/spelingbee/restored/internal/workspace	1.105s
```

**Workspace permissions, the security-critical path**, cross-compiled and run on Linux
because Windows has no POSIX mode to assert and skips the symlink test.

```text
$ CGO_ENABLED=0 GOOS=linux go test -c -o "$TEMP/workspace.test" ./internal/workspace
$ docker run --rm -v "C:/Users/kadyr/AppData/Local/Temp:/t" alpine:3.20 /t/workspace.test -test.v
--- PASS: TestNewCreatesTheTree (0.01s)
--- PASS: TestRunIDIsUsableEverywhere (0.00s)
--- PASS: TestContains (0.00s)
--- PASS: TestSanitiseNeutralisesEscapingSymlinks (0.00s)
--- PASS: TestSanitiseRefusesToLeaveTheWorkspace (0.00s)
--- PASS: TestMeasure (0.00s)
--- PASS: TestCopyTree (0.00s)
--- PASS: TestRelaxOpensARestoredTree (0.00s)
--- PASS: TestWorkspaceRootIsPrivate (0.00s)
--- PASS: TestRelaxRefusesToLeaveTheWorkspace (0.00s)
PASS
```

**Lint, and the generated files.** The last command printing nothing is the point: the
three generated files match what the generator produces from the schema and the
registry.

```text
$ gofmt -l .
$ go vet ./...
$ golangci-lint run
0 issues.
$ golangci-lint run --build-tags integration
0 issues.
$ ./scripts/lint-english.sh
lint-english: ok
$ go run ./tools/gen recipe-spec > docs/recipe-spec.md
$ go run ./tools/gen recipes-index > recipes/README.md
$ go run ./tools/gen readme-table
$ git diff --stat -- docs/recipe-spec.md recipes/README.md README.md
```

**Every recipe against the schema and the safety rules**, including the template, which
validates but is deliberately not in the registry:

```text
$ ./bin/restored recipe validate ./recipes/gitea ./recipes/nextcloud \
    ./recipes/paperless-ngx ./recipes/uptime-kuma ./recipes/vaultwarden --strict
ok       ./recipes/gitea
ok       ./recipes/nextcloud
ok       ./recipes/paperless-ngx
ok       ./recipes/uptime-kuma
ok       ./recipes/vaultwarden
$ echo $?
0
$ ./bin/restored recipe validate ./recipes/TEMPLATE
ok       ./recipes/TEMPLATE
$ ./bin/restored check --recipe nosuch
restored: no bundled recipe named "nosuch" (bundled: gitea, nextcloud, paperless-ngx, uptime-kuma, vaultwarden)
```

**The demos**, re-run after the workspace permission changes, and the README recaptured
from them:

```text
$ ./scripts/demo.sh          ; echo exit=$?
exit=0
$ ./scripts/demo-broken.sh   ; echo exit=$?
exit=1
$ ./scripts/demo-kuma.sh     ; echo exit=$?
exit=0
$ ./scripts/capture-demo.sh
wrote:
  docs/demo/pass.txt     26 lines
  docs/demo/fail.txt     52 lines
  docs/demo/kuma.txt     27 lines
```

**`recipe init --compose` against a real Paperless-ngx compose file**, the fixture now
checked in at `testdata/compose/paperless.yml`:

```text
$ ./bin/restored recipe init demo-paperless --compose ./testdata/compose/paperless.yml --dir /tmp/initout
Wrote ...\initout\demo-paperless from ./testdata/compose/paperless.yml

  application:  webserver (ghcr.io/paperless-ngx/paperless-ngx), port 8000
  dir input:    data from webserver:/usr/src/paperless/data
  dir input:    media from webserver:/usr/src/paperless/media
  dir input:    export from webserver:/usr/src/paperless/export
  dir input:    consume from webserver:/usr/src/paperless/consume
  database:     postgres-dump in service db
  note:         service "broker" looks like a cache or a helper
                (docker.io/library/redis:7). It is kept so the application
                starts, but nothing about it is restored, and no check
                should depend on it.
$ ./bin/restored recipe validate /tmp/initout/demo-paperless; echo $?
ok       ...\initout\demo-paperless
warning  metadata.maintainers is empty: nobody is named as the contact for this recipe
0
```

The database's own data directory and the Redis volume are correctly *not* proposed as
inputs, the published port is gone, and `PAPERLESS_SECRET_KEY` became a literal.

**The launch tooling, dry-run only.** Nothing was filed, no label was created, and the
repository is still private.

```text
$ ./scripts/labels.sh | tail -3
would create or update  bug                #d73a4a  restored did something wrong
would create or update  enhancement        #a2eeef  Something restored should be able to do and cannot

This was a dry run. Nothing was created. Re-run with --apply.
$ ./scripts/recipes-wanted.sh --limit 3
would open  Recipe: n8n
would open  Recipe: open-webui
would open  Recipe: immich

stopping at --limit 3
3 issue(s), 0 already existed
$ ./scripts/contributors.sh --repo louislam/uptime-kuma --days 120 | tail -1
  44 distinct external contributor(s), 53 merged pull request(s)
```

That last command is this project's KPI, run against somebody else's repository because
this one has no contributors yet. Against `spelingbee/restored` it prints, correctly:

```text
No external contributor has had a pull request merged yet.
```

**Every workflow's shell was syntax-checked**, not only its YAML. The 30 `run:`
fragments were extracted, their GitHub expressions stubbed out, and each was passed
through `bash -n`. That found two bugs which would both have been silent in production:
`[ "$count" -eq 0 ] && mode=none` ends the script under `set -e` whenever the count is
not zero, and an `echo` after the harness in `recipe-health.yml` made every run report
success, so no issue would ever have been opened.

```text
$ for f in .wfcheck/*.sh; do bash -n "$f" || echo "SYNTAX $f"; done
shell syntax errors: 0
```

### Session 4 - 2026-08-30 - Five reviews, the fixes, and everything but the tag

The brief: find what is wrong before strangers do, fix it, and prepare the first public
release, stopping before the tag.

#### What the reviews found

Five reviewers worked from fresh contexts, read the repository independently, and
reported without fixing anything. Their reports are in `docs/review/`, one file each,
every finding with a file:line and either a real reproduction or a concrete scenario.

| review | P0 | P1 | P2 | P3 | the one that mattered most |
|---|---|---|---|---|---|
| security | 2 | 3 | 4 | 4 | a recipe variable injects arbitrary YAML into the compose file that runs |
| architecture | 1 | 4 | 7 | 4 | a run that exceeded `--timeout` was reported as `RESTORE UNUSABLE` |
| UX | 2 | 4 | 8 | 3 | the hint catalog could never fire on any failure that returned an error |
| maintainer | 3 | 5 | 7 | 3 | a recipe-only pull request always failed `ci / generated`, which CONTRIBUTING promised it would not |
| fresh-clone | 1 | 5 | 6 | 4 | the README never says how to install the tool |

**9 P0 and 21 P1, all fixed.** The 38 P2 and P3 findings are written up in
`docs/review/backlog.md` and filed by `scripts/backlog-issues.sh` once the repository is
public - deliberately not fixed, because they are the contributor entry points and a
project measured by external contributors should not arrive at launch having done every
job small enough for a stranger.

They corroborated each other where it counted, which is the useful signal from running
five. Three independently found that hints never fire on an error path. Security and
maintainer found the unvalidated top-level `volumes:` block from opposite directions -
one probing the validator, one reading what a maintainer would have to check by hand.

#### The two that were genuinely dangerous

Both were accepted by `restored recipe validate --strict` with exit 0, and both end in
root on the machine running the drill - including the GitHub runner that tests every
fork pull request.

**SEC-01, YAML injection through a recipe variable.** The safety schema runs on
`compose.yaml` as written, with the `${RESTORED_*}` placeholders intact (ADR-039), so
the only structural check happens before the values go in. A `vars` value containing a
line break therefore added `privileged: true`, `network_mode: host` and `pid: host` to a
service, after validation had passed. Fixed by asserting the invariant interpolation is
supposed to have - it replaces scalar values and changes nothing else about the
document - and by routing every caller through it, `recipe validate` included, so the
injection is refused where a maintainer looks. ADR-056.

**SEC-02, the deny-list losing to the compose specification.** The top-level `volumes:`
block was not validated at all, so a named volume with
`driver_opts: {type: none, device: /, o: bind}` was a bind mount of the host root, and
`device: /var/run` handed a container the Docker socket that SPEC.md 9.3 says restored
never mounts. The service body accepted every key nobody had listed, of which
`volumes_from: ["container:<name>"]` attaches a running container's volumes. The schema
is now an allow-list at the root, the service, the network and the volume. ADR-057.

#### The one that would have been most expensive

**ARCH-01.** Every stage past LOAD DUMPS routed through `fail()`, which set
`RESTORE_UNUSABLE` and exit 1 unconditionally - so a run that merely ran out of
`--timeout` accused a backup that may be perfectly good. The defaults guaranteed it:
`--timeout 15m` with a 10m restore budget and a 5m ready budget inside it left nothing
for compose up, the dump load and the checks.

Exit 1 is the number people put in cron jobs and alerting rules. It was wrong in the one
case where a false alarm costs the most: at 03:00, about a backup that was fine.
ADR-058.

#### What a terminal found that nothing else could

Rendering the demo GIF is the first time anything in this project ran `restored check`
on a real TTY. The recording hung, and the screenshot said why: after the `PASS`, the
contribution nudge had printed about three thousand characters of percent-encoded YAML,
wrapped across twenty lines, pushing the report off the screen.

Nothing had caught it in three sessions for a specific reason: **the nudge only fires on
a TTY, and every test, every golden file and all five reviewers captured output through
a pipe, where it does not fire at all.** The one output path every user sees was the
one path nothing exercised. The URL is gone (ADR-066), and `docs/demo/demo.tape` is now
the test that would have caught it.

The same run showed the nudge inviting a contribution of `gitea`, which ships in the
binary, because `rec.Bundled` is false for `--recipe ./recipes/gitea` - which is what
`scripts/demo.sh` does.

#### The release, up to the tag

- `.goreleaser.yaml`: six targets, archives, `checksums.txt`, an SBOM per archive, a
  changelog grouped from conventional commits, and a Homebrew cask with `skip_upload`.
- `install.sh`: OS and architecture detection, checksum verification against
  `checksums.txt`, `~/.local/bin` by default, and a refusal to run as root without
  `--system`.
- `Dockerfile` and `docs/docker.md`: the image NAS users need, and a document that says
  plainly that a mounted Docker socket is root on the host and that restored's own
  isolation rules do not cover it.
- `CHANGELOG.md`, `docs/homebrew-tap.md`, `docs/release-checklist.md`.
- `docs/demo/demo.gif`, rendered by `vhs` from the real `scripts/demo.sh` and
  `scripts/demo-broken.sh`.

Nothing is published. The checklist for what a human does next is
`docs/release-checklist.md`, and every item on it is a stop point.
#### Evidence

Every command below was run on this commit, on this host, and the output is its real
tail. Where something could not be run here, it says so and says why.

**The suite, and the race detector CI uses.** `-race` needs a C compiler, which this
host does not have, so it runs in the image CI runs:

```text
$ docker run --rm -v "C:/My/Projects/Work/restored:/src" \
    -v "C:/Users/kadyr/go/pkg/mod:/go/pkg/mod" -w /src golang:1.27 go test ./... -race
ok  	github.com/spelingbee/restored/internal/check           1.124s
ok  	github.com/spelingbee/restored/internal/cli             1.311s
ok  	github.com/spelingbee/restored/internal/harness         1.167s
ok  	github.com/spelingbee/restored/internal/hints           1.153s
ok  	github.com/spelingbee/restored/internal/loader          1.124s
ok  	github.com/spelingbee/restored/internal/nudge           1.031s
ok  	github.com/spelingbee/restored/internal/recipe          1.335s
ok  	github.com/spelingbee/restored/internal/recipe/safety   1.228s
ok  	github.com/spelingbee/restored/internal/report          1.674s
ok  	github.com/spelingbee/restored/internal/runner          1.154s
ok  	github.com/spelingbee/restored/internal/source/restic   1.031s
ok  	github.com/spelingbee/restored/internal/workspace       1.099s
RACE_EXIT=0
```

**The symlink-escape test, which skips on Windows and must not go unrun.** It is the
security-critical path, and the schema hardening this session touches the same threat
model:

```text
$ CGO_ENABLED=0 GOOS=linux go test -c -o /tmp/workspace.test ./internal/workspace
$ docker run --rm -v /tmp:/t alpine:3.20 /t/workspace.test -test.run 'Symlink|Sanitise' -test.v
=== RUN   TestSanitiseNeutralisesEscapingSymlinks
--- PASS: TestSanitiseNeutralisesEscapingSymlinks (0.01s)
=== RUN   TestSanitiseRefusesToLeaveTheWorkspace
--- PASS: TestSanitiseRefusesToLeaveTheWorkspace (0.00s)
PASS
```

**Every probe the security review left behind, re-run against the fixed validator.**
Seven of these were accepted with exit 0 before this session:

```text
$ for d in scratchpad/secrev/p*; do ./bin/restored recipe validate "$d" --strict; done
p01-privileged     exit=2  INVALID
p02-netmode        exit=2  INVALID
p03-absbind        exit=2  INVALID
p04-driveropts     exit=2  INVALID   <- was ok
p05-merge          exit=2  INVALID
p06-docksock       exit=2  INVALID   <- was ok
p07-volumesfrom    exit=2  INVALID   <- was ok
p08-extrahosts     exit=2  INVALID   <- was ok
p09-longbind       exit=2  INVALID
p10-longvolopts    exit=2  INVALID   <- was ok
p11-capadd         exit=2  INVALID
p12-secopt         exit=2  INVALID   <- was ok
p13-extnet         exit=2  INVALID
p14-externalvol    exit=2  INVALID   <- was ok
p15-yamlinject     exit=2  INVALID   <- was ok, and is SEC-01
p17-dotdot         exit=2  INVALID
```

and the injection now fails where a maintainer looks:

```text
$ ./bin/restored recipe validate .../p15-yamlinject --strict
INVALID  .../p15-yamlinject
         compose.yaml: the value of ${RESTORED_VAR_port} contains a line break, which
         would add lines to the compose file rather than fill in a value. Recipe
         variables are single-line values; use an input for anything larger
exit=2
```

**All five bundled recipes still validate under the allow-list.** They use seven
compose service keys between them, which is what makes the list credible rather than
merely tight:

```text
$ ./bin/restored recipe validate ./recipes/*/ --strict
ok       ./recipes/TEMPLATE/
ok       ./recipes/gitea/
ok       ./recipes/nextcloud/
ok       ./recipes/paperless-ngx/
ok       ./recipes/uptime-kuma/
ok       ./recipes/vaultwarden/
exit=0
```

**The highest-traffic error now arrives with a report and a hint that had never once
been printed** (ARCH-04, UX-01, UX-03):

```text
$ NO_COLOR=1 ./bin/restored check --recipe gitea --source dir --from <empty tree>
restored 0.1.0-dev - recipe gitea - run uf2dwuts

  restore    FAILED     0.00s
             required input "data": no matching files found for
             /srv/gitea/data in the backup.

  ERROR  0/0 checks  -  total 0.54s  -  teardown ok

  required input "data": no matching files found for /srv/gitea/data in the backup.
  A recipe default path is a guess at your layout. Point this input
  at the path your backup actually uses:
      --input data=/your/path
  `restored recipe show gitea --inputs-only` lists every input this recipe wants.

  LIKELY CAUSE
    The recipe's default path is not where your backup keeps that data
    ...
      restic ls latest | head -50
                                         (hint: restore/path-not-in-snapshot)
EXIT=2
```

Before this session that invocation printed one line, no report, no hint, and the rule
`restore/path-not-in-snapshot` was unreachable code.

**The release build, end to end.**

```text
$ goreleaser check
  - 1 configuration file(s) validated

$ goreleaser release --snapshot --clean
  - archives: 6 (linux/darwin/windows x amd64/arm64)
  - software bill of materials: 6 catalogued with syft
  - calculating checksums
  - homebrew cask: writing dist\homebrew\Casks\restored.rb
  - release succeeded after 1m44s
```

**The Linux binary out of that archive, run through the real demo.** Not the host
build - the artifact:

```text
$ tar -xzf dist/restored_0.0.1-snapshot_linux_amd64.tar.gz -C /rel
$ docker run --rm -u 0 -v /var/run/docker.sock:/var/run/docker.sock \
    -v /var/lib/restored-demo:/var/lib/restored-demo -v .../restored:/src:ro -v /rel:/rel:ro \
    -e TMPDIR=/var/lib/restored-demo -e RESTORED_BIN=/rel/restored -w /src \
    --entrypoint sh restored:dev -c './scripts/demo.sh'

restored 0.0.1-snapshot - recipe gitea - run ffaczssj
  source     restic  /var/lib/restored-demo/tmp.jFkOFD/repo
  snapshot   1961a4a9  2026-08-30 02:38:11  host=demo-host  tags=[gitea]
  restore    ok          1.5s   2 inputs
  compose    ok         0.57s   2 services, db first for the dump
  load db    ok          2.2s   db: psql, 0 stderr lines
  ready      ok          4.7s   postgres accepts connections, gitea answers
  PASS  5/5 checks  -  total 10.5s  -  teardown ok
RELEASE_DEMO_EXIT=0
```

That is also the proof for `docs/docker.md`: a `restored check` from inside the image,
driving the host's daemon through a mounted socket, with the workspace mounted at the
same path on both sides.

**The image.**

```text
$ docker run --rm restored:dev version
restored 0.0.0-docker
  docker:    not found
  restic:    0.18.0
  recipes:   5 bundled
exit=0

$ docker run --rm --group-add 0 -v /var/run/docker.sock:/var/run/docker.sock restored:dev version
  docker:    29.5.2 (compose v2.36.2)

$ docker run --rm -v /var/run/docker.sock:/var/run/docker.sock restored:dev version
  docker:    not found        # the non-root user cannot read the socket without --group-add
```

**`install.sh`, every guard that does not need a release to exist.**

```text
$ docker run --rm alpine:3.22 sh /i.sh            # as root, no --system
install.sh: refusing to run as root without --system.
exit=1

$ docker run --rm alpine:3.22 sh /i.sh --nope
install.sh: unknown option --nope. Run install.sh --help.
exit=1

$ docker run --rm --network none -u 1000 alpine:3.22 sh /i.sh --version v0.1.0
Downloading restored_0.1.0_linux_amd64.tar.gz (v0.1.0, linux/amd64)...
install.sh: could not download https://github.com/spelingbee/restored/releases/download/v0.1.0/restored_0.1.0_linux_amd64.tar.gz
exit=1
```

The asset name it builds matches `.goreleaser.yaml`'s `name_template` exactly, which is
the coupling that would otherwise break silently. **Not tested: the actual download and
checksum verification against a real release**, because there is no release. That is
item 4 of the checklist in `docs/release-checklist.md`, to be done against the draft's
own assets before the draft is published.

**The round trip, all five recipes, in sequence.** This is what `recipes.yml` runs one
recipe per job:

```text
$ ./bin/restored recipe test ./recipes/gitea ./recipes/nextcloud \
    ./recipes/paperless-ngx ./recipes/uptime-kuma ./recipes/vaultwarden --timeout 20m

  stage A  negative: the checks must fail against an empty sta... PASS      13.9s
  stage B  round trip: seed, export, back up, restore, check     PASS      35.3s
  PASS   gitea in 49.7s
  PASS   nextcloud in 1m18s
  PASS   paperless-ngx in 1m43s
  PASS   uptime-kuma in 2m02s
  PASS   vaultwarden in 17.1s

  5 recipes: 5 passed, 0 failed, 0 errored, in 6m11s
exit=0
```

Every stage now also prints a reproduction command that still works after teardown -
`restored recipe test recipes\gitea --stage a --keep` - which is ADR-061.

**The integration suite**, which stands up Gitea, PostgreSQL and Uptime Kuma several
times, after the build-tag fix below:

```text
$ go test -tags integration ./... -timeout 30m
ok  	github.com/spelingbee/restored/internal/runner   170.991s
ok  	github.com/spelingbee/restored/internal/check    (cached)
ok  	github.com/spelingbee/restored/internal/harness  (cached)
ok  	github.com/spelingbee/restored/internal/report   0.777s
...
INTEGRATION_EXIT=0
```

**Nothing was left behind.**

```text
$ docker ps -aq --filter "label=com.restored.run" | wc -l
0
```

**The captured demo text, re-captured on this commit.** `scripts/capture-demo.sh` runs
all three demos for real and splices the result into README.md between its markers:

```text
$ ./scripts/capture-demo.sh
capturing ./scripts/demo.sh       -> docs/demo/pass.txt
capturing ./scripts/demo-broken.sh -> docs/demo/fail.txt
capturing ./scripts/demo-kuma.sh  -> docs/demo/kuma.txt
exit=0

$ git diff -- docs/demo/ | grep '^[-+]' | grep -v 'run id|snapshot|timing'
(nothing: only run ids, snapshot ids and durations differ)
```

**The demo GIF**, from a real run of both demos: 146 seconds at double speed, 776 KB,
ending on `RESTORE UNUSABLE 2/5 checks` with the `db/tables-empty` hint and `exit=1`.
Its last frame was inspected rather than assumed.

#### What the verification run caught

Worth recording, because it is the argument for running the whole thing rather than the
parts that are quick:

**`go test -tags integration ./...` did not compile.** ADR-063 removed the boolean from
`compose.Preflight`, and the call in `internal/runner/integration_test.go` still passed
one. That file is behind a build tag, so `gofmt`, `go vet`, `golangci-lint`, `go build`
and the entire unit suite compile past it without looking - all of them green, all
session. It would have gone red in CI's `integration` job and nowhere else.

`make lint` and CI's `lint` job now also run `go vet -tags integration ./...`.

**`scripts/lint-english.sh` reported the demo GIF's own bytes as non-English.** It
skipped binaries by an extension list, which is a list that is always one entry out of
date. It now asks `grep -I` whether the file is text.

Both are the same bug in different clothes: a check whose coverage depends on somebody
remembering to extend it.

### Session 5 - 2026-08-30 - The official-docs restore drill

**Goal:** for the most popular self-hosted applications, follow each one's *own* backup
documentation literally, take the backup it describes, restore it with `restored`, and
record whether the application comes back with its data. Not "does application X lose
data" - "if you follow the documentation as written, here is what you get back". Every
application tested also gets a recipe that passes `restored recipe test`.

**Done:**

1. **Fifteen applications, end to end.** Each has a folder under `docs/drill/` with the
   official documentation quoted as written (`docs.md`), the commands that were run and
   their real output (`steps.md`, `run.sh`), the reports the tool produced
   (`result*.txt`, `result*.json` - never hand-written), the verdict and the root cause
   with evidence (`result.md`), and a draft issue that has **not** been filed
   (`upstream-issue.md`).

   Verdicts on the primary documented reading: **10 PASS, 2 PARTIAL, 2 FAIL, 1
   SKIPPED**. Five secondary readings also failed. `docs/drill/summary.md` has the totals, the three
   most instructive cases, the five patterns that repeat, three headline options and the
   caveats; `docs/drill/README.md` is the table.

2. **Fifteen new recipes**, one per application, all passing both stages of
   `restored recipe test`. The registry is twenty recipes, which is well past the
   six-recipe gate of ADR-033.

3. **`docs/drill/SKIPPED.md`** - every application passed over, in three honest
   categories: too big for the budget (immich, appwrite, langfuse, signoz and five
   others), nothing a backup would be for (glance, homepage, dashy - configuration files
   an operator wrote), and not reached. Stirling-PDF is recorded as attempted and
   abandoned, with the exact reason.

4. **`scripts/lint-english.sh`** now skips `docs/drill/*/result*.json`. Those files are
   written by `restored --report` and quote another application's log lines verbatim;
   Navidrome prints elapsed times in microseconds and "us" is not what it prints.

**The three findings a human should look at first**, all reproduced twice from an empty
scratch directory:

- **Navidrome (FAIL).** The best backup page in the drill, and its restore does not
  work. `navidrome backup restore` refuses to start without an existing database - which
  is the state of every machine anybody restores onto - and once given one it reports
  `Restore complete` in six milliseconds and leaves the instance empty. `POST
  /auth/createAdmin` then answers 200, which Navidrome only does when it has no users at
  all. The backup file itself has the user, the playlist and the library row in it.
- **Gogs (FAIL).** `gogs backup` writes a complete archive and the official image runs
  it on a schedule if you enable it; `gogs restore --from` cannot run inside that image.
  It resolves the database path against its working directory rather than `GOGS_CUSTOM`
  and dies on `mkdir /app/gogs/data: permission denied`; made writable, it fails on
  `rename ... invalid cross-device link` after moving the live `/data/gogs` aside. Two
  open upstream issues (#4339, #7684) describe the same wall, so the draft is written as
  a comment on those rather than a third issue.
- **n8n (PARTIAL).** The only thing n8n's documentation calls a backup -
  `export:workflow --backup` plus `export:credentials --backup` - restores the workflows,
  no users, and a credential nobody can decrypt, because the encryption key is in
  `.n8n/config` and the export does not contain it. Separately, and reproducibly:
  `import:credentials --separate` aborts on the very directory the documented export
  commands write, because it tries to insert the workflow file sitting next to the
  credential as a credential.

**Not done:** Stirling-PDF, which was fourth on the list and the intended fifteenth. It
was pulled (3.38 GB), deployed and abandoned: its v2 interface is a JavaScript client,
`POST /login` answers 405 and HTTP Basic answers 401 on every `/api/v1/user/...`
endpoint, so nothing could be seeded through the application - and without seeding, a
restore check cannot tell a restored instance from a fresh one. No verdict was invented
for it; ConvertX took the fifteenth slot instead. `internal/config`, which was
the *previous* plan for session 5, is untouched and is still the next thing.

**Evidence.**

The whole registry, both stages, in one run:

```text
$ ./bin/restored.exe recipe test ./recipes/beszel ./recipes/changedetection     ./recipes/filebrowser ./recipes/freshrss ./recipes/gogs ./recipes/gotify     ./recipes/listmonk ./recipes/mealie ./recipes/memos ./recipes/n8n     ./recipes/navidrome ./recipes/open-webui ./recipes/siyuan ./recipes/trilium
  PASS   beszel in 39.4s
  PASS   changedetection in 1m34s
  PASS   filebrowser in 41.5s
  PASS   freshrss in 25.3s
  PASS   gogs in 47.0s
  PASS   gotify in 44.7s
  PASS   listmonk in 1m05s
  PASS   mealie in 1m09s
  PASS   memos in 39.0s
  PASS   n8n in 1m36s
  PASS   navidrome in 45.3s
  PASS   open-webui in 1m24s
  PASS   siyuan in 58.2s
  PASS   trilium in 57.2s

  14 recipes: 14 passed, 0 failed, 0 errored, in 13m28s
```

ConvertX arrived last and was run on its own:

```text
$ ./bin/restored.exe recipe test recipes/convertx
  stage A  ... PASS   4 of 6 checks failed against an empty stack
  stage B  ... PASS   the round trip restored and all 6 checks passed
  PASS   convertx in 1m19s
```

The five that existed before, unchanged and re-run to keep the "all twenty" claim above
honest:

```text
$ ./bin/restored.exe recipe test ./recipes/gitea ./recipes/nextcloud     ./recipes/paperless-ngx ./recipes/uptime-kuma ./recipes/vaultwarden
  PASS   gitea in 50.1s
  PASS   nextcloud in 1m20s
  PASS   paperless-ngx in 1m44s
  PASS   uptime-kuma in 2m00s
  PASS   vaultwarden in 16.4s

  5 recipes: 5 passed, 0 failed, 0 errored, in 6m13s
```

Everything else still green:

```text
$ go build ./... && go test ./...
(no output; every package ok or without tests)

$ gofmt -l .
(nothing)

$ go vet ./... && go vet -tags integration ./...
(nothing)

$ ./bin/restored.exe recipe validate ./recipes/*/ --strict
ok       ./recipes/TEMPLATE/
ok       ./recipes/beszel/
... 20 lines, all ok ...

$ ./scripts/lint-english.sh
lint-english: ok
```

The drill's own verdicts are not summarised here: each application's folder holds the
report the tool wrote, and `docs/drill/summary.md` holds the totals. Nothing in this
entry restates a verdict that is not in a `result*.txt` next to it.

### Session 6 - 2026-08-30 - The first weekly maintainer session, pre-launch

**Goal:** the weekly maintainer brief (`06-maintainer-week.md`): KPI, PR and issue
triage, registry hygiene, drill top-up, report. The brief says to run it once a week
*after launch* and opens with "The project is live." It is not: there is no git remote,
`spelingbee/restored` does not resolve on GitHub, and stop points 1, 3, 4 and 6 are all
still closed. The session ran the brief honestly against that state rather than
pretending either way.

**Done:**

1. **`docs/maintainer/2026-08-30.md`** - the first weekly report, evidence inline: KPI
   is 0 external contributors of 0 possible; zero PRs and zero issues because nowhere
   exists for one; registry hygiene verified; the drill queue trigger examined and not
   met; a 118-word drill post drafted and **not** posted; and the one decision that
   needs a human - launch, or schedule this brief to start after stop point 4, because
   every weekly report will be this report until then.
2. **Hygiene, all verified on commit 53804b7.** Generators re-run with no diff; all
   twenty recipe directories validate `--strict`; the unit suite green uncached;
   `gofmt`/`go vet` (both tag sets)/`lint-english` clean; and, as the weekly
   recipe-health substitute (CI has never run), the two fastest recipes round-tripped
   fresh today.
3. **MNT-19** added to `docs/review/backlog.md`, found by running the KPI script
   against the not-yet-existing repository: `contributors.sh` prints the GraphQL error
   and then reports zero contributors with exit 0, so after launch an API failure
   reads as "0 contributors" on a dashboard.
4. **A concurrent drill leg, found live and left alone.** While this session wrote its
   report, another session was writing a ConvertX drill leg into the same working tree:
   `docs/drill/convertx/` and `recipes/convertx/`, both untracked, the first file at
   20:40 and result files still landing at 20:52. Nothing of it is committed or claimed
   by this session, and the running session was messaged directly so the two would not
   collide in this file. Its verdict is its own session's to record, not this one's.

**Not done, deliberately:** no drill legs - the brief's trigger ("fewer than 3 untested
apps left in the queue") is not met, with seven-plus feasible applications recorded in
`docs/drill/SKIPPED.md`; no fixes beyond the report - the brief forbids feature work
and the backlog is contributor inventory; nothing filed, posted, merged or published.

**Evidence.** The report carries the full tails; the load-bearing ones:

```text
$ git remote -v
(nothing)
$ ./scripts/contributors.sh
no git remotes found
$ go test ./... -count=1
ok      github.com/spelingbee/restored/internal/check   1.863s
... 12 packages, all ok ...
$ ./bin/restored recipe validate ./recipes/*/ --strict; echo $?
... 20 lines, all ok ...
0
$ ./bin/restored recipe test ./recipes/vaultwarden ./recipes/freshrss
  2 recipes: 2 passed, 0 failed, 0 errored, in 38.9s
$ docker ps -aq --filter "label=com.restored.run" | wc -l
0
$ git diff --stat -- docs/recipe-spec.md recipes/README.md README.md
(nothing: the generated files are current)
```

---

## Next steps

In order. Each is sized to be finishable and committable on its own.

### Session 7 - `internal/config`, and the rest of the CLI surface

(This was session 5's plan, and then session 6's. Session 5 did the restore drill and
session 6 the first weekly maintainer run, each on a brief that arrived after this was
written, and both left this untouched. It is still the next coding session.)

`internal/config`: `restored.yaml`, sources, targets, the precedence chain, `--config`,
`--target`, `--all`, and the `--all` report shape. Then diff `--help` against SPEC.md
section 2 and fix whichever of the two is wrong (ADR-045).

One thing to fold in rather than work around: `internal/nudge.Silenced` already reads
`defaults.nudge` out of `restored.yaml`, through a deliberately narrow reader that looks
at that one key and uses SPEC.md 2.9's search order. It exists so a user who wrote
`nudge: false` is believed today. When `internal/config` lands it should take that over
and `nudge/config.go` should go; the behaviour must not change under anyone.

### The sixth recipe - done, and then some

**Answered by session 5.** The six-recipe gate of ADR-033 is met and passed: there are
twenty, and all twenty round-trip. Miniflux is still not among them, and session 4's
fresh-clone reviewer's recipe for it is still in `docs/review/fresh-clone.md` if
somebody wants a twentieth.

The fifteen that arrived with the drill were each written against the application's own
documentation and each proves itself, so `recipes/` is now a registry rather than a
sample. Two things in it are worth copying rather than reinventing: a check that names
the object it is looking for instead of counting rows (a fresh application usually makes
its own administrator, so `count(*) > 0` passes against a restore of nothing), and a
check that asks for the file beside the database - the uploads directory, the avatar, the
encryption key - because that is where six of the fifteen keep something a person would
call their data.

### The launch, which is entirely a human's

`docs/release-checklist.md` is the ordered list, and every item on it is a stop point in
CLAUDE.md. In summary: six repository settings that no shell can turn on, then labels,
then public, then the tag, then the image and the tap, then the issues. Nothing on it
should be done by a session on its own initiative, and the reason the checklist exists
is so that none of it has to be worked out under time pressure on the day.

### Then

`docs/security.md`, `smoke.yml`, and a `mysql-dump` input kind - `recipe init --compose`
already recognises a MySQL service and tells the contributor, in the file it writes,
that restored cannot restore it yet. That message is a promise to somebody.

And the 38 findings in `docs/review/backlog.md`, which are not for a session to do:
they are what a stranger picks up. `scripts/backlog-issues.sh --apply` files them once
the repository is public, and the four best first issues in the list were left unfixed
on purpose.

`docs/roadmap.md` has the rest, including the two harness gaps worth closing first: a
`wait` step so a recipe can seed something the application processes asynchronously,
and a way to hand a test asset to a service, which is what stands between Paperless-ngx
and a real document round trip.

---

## Blocked

| # | Item | Since | Blocking | Needed from |
|---|---|---|---|---|
| - | Nothing. | - | - | - |

Everything that is *waiting* is waiting on a human, not on a discovery, and every one of
them is a stop point in CLAUDE.md with the work up to it finished:

- the repository is not public (stop point 4), so `scripts/backlog-issues.sh` and
  `scripts/recipes-wanted.sh` cannot file the 38 + 50 issues that are written and ready;
- nothing is tagged (stop point 1), so `install.sh` cannot be tested against a real
  release asset - the only part of it that is unverified;
- nothing is pushed to ghcr or a tap (stop point 6);
- six repository settings have to be turned on by hand, three of which documents in
  this repository already promise. `docs/release-checklist.md` is the ordered list.

## Open questions for a human

1. **The name and the owner, still.** `github.com/spelingbee/restored` is in `go.mod`,
   in both schema `$id`s, in `internal/nudge`, in the CI workflows, in
   `.all-contributorsrc`, and in every document that links to a file on GitHub.
   `docs/name-check.md` recommends `drillback` because it is the only candidate clean on
   every registry and all three TLDs, and `restored` costs discoverability. Changing it
   is still `grep -rl spelingbee/restored | xargs sed -i` plus a `go mod edit`, and it
   is still free until something is published - which is stop points 1, 3, 4 and 6, all
   of which are still closed. It got more expensive this session, because there is more
   of it. See ADR-036.
2. **ADR-023 - is a failed ready probe exit 1 or exit 2?** Unchanged, and now
   implemented as exit 1. Session 3 hit two more instances of the shape the question is
   about, and both were recipe bugs reported as unusable restores: Vaultwarden refusing
   to start over an empty `SMTP_HOST`, and Nextcloud answering 503 because a restored
   tree carried somebody else's uids. Both are fixed in the recipes. What the question
   is really asking is whether "the app did not start" and "the data did not come back"
   deserve different exit codes, and after three sessions the honest answer is that a
   user cannot tell them apart from the exit code today.
3. **ADR-030 - should digest pinning be required rather than encouraged?** Unchanged.
   Five recipes now pin tags, and `recipe-health.yml` runs weekly with `--pull always`
   specifically so that a moved tag is noticed. That weakens the case for requiring
   digests, because the failure is now detected rather than silent.
4. **ADR-033 - are six recipes the right gate for v0.1?** Five exist and all five
   round-trip in under three minutes each. The sixth is a session's work.
5. **Is Nextcloud achievable inside the isolation rules?** **Answered: yes**, and
   without an exception. It needed a `prepare` service in the recipe's own compose file
   that chowns the restored tree and lays a config overlay over `config.php` - which is
   the same procedure Nextcloud's own restore documentation gives a human. No isolation
   rule was bent. The new question it raises is item 7.
6. **`restored recipe test` is not built.** **Answered: it is**, and all five recipes
   pass it. A recipe from a stranger can now be accepted on evidence.
7. **New: should a recipe be able to declare a preparation step?** The Nextcloud recipe
   declares a twenty-line compose service that fixes ownership and writes a config
   overlay before the application starts. It works and it is honest, but every recipe
   for an application that is fussy about ownership will now copy it, and a copied
   twenty-line service is a convention rather than a mechanism - which is the thing
   CLAUDE.md says to prefer against. A `prepare:` block in the recipe format would be a
   mechanism. It would also be a new way for a recipe to run arbitrary code, which is
   exactly what the isolation rules exist to constrain.
8. **The review promise in CONTRIBUTING.md is a promise.** "First response within
   24 hours, merged within 48 when CI is green." It is the right promise - it is most of
   what makes a first contribution feel worth making - and there is one maintainer. It
   should be confirmed or changed before the repository is public, not after somebody
   has relied on it. Session 4 made it cheaper to keep (`recipes / verdict` is now a
   single check to look at rather than a matrix to read) and told contributors that
   their first run waits for approval - but the clock is still a human's.
9. **New: is a `contact address` needed, and whose?** `SECURITY.md` and
   `CODE_OF_CONDUCT.md` both route through GitHub's private advisory form, which is a
   repository setting that has to be turned on (`docs/release-checklist.md` item 1.1).
   Session 4 added the two fallbacks that do not need an address - a public issue
   containing only the request, and GitHub's abuse report for a conduct complaint about
   the maintainer - deliberately without inventing an email address to put in a public
   repository. Whether to publish one is a human's call, and it is cheap now and
   awkward later.
10. **New: `dist/homebrew/` versus `docs/homebrew-tap.md`.** The session brief asked for
    the tap content under `dist/homebrew/`. It is in `docs/homebrew-tap.md` instead,
    because `goreleaser --clean` empties `dist/` on every run: a checked-in file there
    disappears the next time somebody builds a release, and then gets committed as a
    deletion by whoever runs `git add -A` afterwards. If the tap repository should be a
    subtree of this one later, that decision changes this.
