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

**Phase:** reviewed, hardened, green on every recipe, and ready to release - up to the tag, which is a human's
(stop point 1). Five independent reviewers went through the repository before strangers
could; all 9 P0 and all 21 P1 findings are fixed; the remaining 38 are written up as
`help wanted` issues waiting for the repository to be public.
**Version:** **v0.1.0, released 2026-09-03** at `7d93053`:
https://github.com/spelingbee/drillback/releases/tag/v0.1.0 - six archives, six SBOMs,
checksums; `install.sh` verified end to end from a clean Ubuntu container. `0.1.0-dev`
is what a local build reports; the release version comes from the tag through ldflags
and from nowhere else (which is issue #78 for `go install` users). Still open from the
release checklist: the ghcr push (needs a `write:packages` token), the tap's token and
secret, and the announcement.
**Module:** `github.com/spelingbee/drillback`. The name is settled: the human chose
the rename at stop point 7 on 2026-08-30, session 8 executed it, ADR-070 records it.
Pre-rename history in this file says `restored`, deliberately.
**Language of record:** English, everywhere, enforced by `scripts/lint-english.sh`.

What works, and is proved by a command below:

| Capability | State |
|---|---|
| `restored check --recipe <bundled\|dir> --source restic --from <repo>` | works, PASS and RESTORE UNUSABLE both reached against real stacks |
| `restored check --source dir --from <tree>` | works |
| **`restored check --target <name>` / `--all` / `--config`** | **works since session 7; `restored.yaml` with sources, targets, defaults and the precedence chain (ADR-067); `--all` runs in file order and exits with the worst outcome (ADR-068)** |
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

- **`smoke.yml`**, the fresh-clone test of SPEC.md 11.3. The `unit` job proves
  `go test ./...` is green with no docker and no restic, which is most of what it was
  for; session 4's fresh-clone review walked the rest of it by hand.
- **`docs/recipes.md`, `docs/security.md`.** `CHANGELOG.md`, `install.sh`,
  `docs/docker.md`, `docs/homebrew-tap.md`, `docs/release-checklist.md` and
  `docs/recipe-spec.md` all exist now.
- **The 38 P2 and P3 findings** in `docs/review/backlog.md`. Deliberately not fixed:
  they are the contributor entry points. They are filed - issues #4-#46 carry the
  `help wanted` label, the 35 `recipes-wanted` issues run to #76, all on 2026-08-31.
- **No release yet.** No tag, no image pushed, no tap. The repository **is public**
  (found so by session 9 on 2026-09-03; the human's stop-point-4 decision), the
  labels exist, the 76 issues are filed, and the two public-only settings - private
  vulnerability reporting, first-time-contributor approval - are on and verified via
  the API. What remains is stop points 1 and 6 and the announcement;
  `docs/release-checklist.md` is the ordered list.

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

### Session 7 - 2026-08-30 - `internal/config`, and the rest of the CLI surface

**Goal:** ADR-045's debt, twice deferred: `restored.yaml`, `--config`, `--target`,
`--all`, the `--all` report shape, and then the `--help`-versus-SPEC-section-2 diff.

**Done, in order:**

1. **`internal/config`** (ADR-067). Strict decoding - an unknown key refuses, because
   `enabld: false` silently running a disabled target is the config-file version of a
   false PASS. `version: 1` required; a restic source needs `repository`, a dir source
   needs `path`, and a field of the other kind is named in the error. `${NAME}` in a
   source's `env` resolves from the process environment at run time and an unset
   variable refuses loudly. Relative host paths resolve against the config file's
   directory (rooted POSIX paths count as absolute even on Windows). The SPEC.md 2.9
   example is the loader's test fixture, so the two cannot disagree. `internal/nudge`'s
   one-key reader moved here as `config.NudgeSilenced`, behaviour unchanged, tests
   ported; `nudge/config.go` is gone.
2. **`check --config`, `--target` and `--all`** (ADR-068). `--target` is an ordinary
   single run fed from the config; `--all` runs every enabled target in file order,
   renders each report as it finishes, closes with a summary block, and exits with the
   worst outcome. Every target resolves before any target runs. A flag beats the config
   only when the user actually typed it (`Flags().Changed`). The multi-target JSON
   document is SPEC.md 5.2's, each element the single-run document plus a `target`
   field; `report.Multi` has golden renderings in colour, plain and ASCII. The nudge
   never fires under `--all`. A source's password settings and env block travel to
   restic as child-process environment (`runner.Options.SourceEnv`), never logged.
3. **The `--help` diff of ADR-045**, adjudicated as ADR-069: SPEC section 2 is
   normative for the surface (flags, defaults, exit codes), cobra owns the bytes; the
   2.1 exit footer was synced from the build's real one (130 included); the missing
   `Environment:` block in the real `check --help` stays UX-11, a contributor's.
4. **UX-18, found the hard way.** `recipe test --keep` keeps *two* workspaces and two
   compose projects - the harness's and its inner check's - and names only the
   harness's. This session spent twenty minutes treating its own inner-check leftovers
   (`restored-jg6vpitw`, still Up, teardown line absent - which `--keep` makes correct)
   as a possible rogue process, coordinating with the drill session before the
   timestamps closed the case; the drill session then reported the same two-workspace
   surprise from its own n8n `--keep` run hours earlier. Recorded in
   `docs/review/backlog.md` with the directory-shape tell that distinguishes the two.
5. **GitHub, private.** `spelingbee/restored` created private and `origin` set; the
   push is blocked on the gh token's missing `workflow` scope (it may not write
   `.github/workflows/`), waiting on the human: `gh auth refresh -h github.com -s
   workflow`. Creating a *private* repository is not a stop point; making it public
   remains stop point 4, untouched.
6. **An adversarial review pass, and its fixes.** An independent reviewer read the
   three commits, confirmed the hard parts clean where it hunted hardest (the full
   precedence walk, duplicate YAML keys, env-to-log leakage, restic 0.19.1 accepting
   the quoted `RESTIC_PASSWORD_COMMAND` end to end) - and found two real defects plus
   four smaller ones, all fixed in the follow-up commit:
   - an interrupted `--all` could exit 0 claiming a clean sweep: a SIGTERM landing
     during a passing target's teardown (which runs uncancelled, on purpose) completed
     that target and silently skipped the rest. Skipped targets are now counted in
     both documents (`targets_skipped`, and a red "never ran" line under the summary)
     and force exit 2 - what never ran is unproven, and unproven is a statement about
     the drill (ADR-058);
   - the config suite failed on any machine that exports real AWS credentials,
     because a test asserted `${AWS_*}` was unset. It now unsets them itself;
   - the `--all` summary column misaligned for the [10m, 1h) duration band, which a
     30-minute default timeout makes ordinary and which no golden exercised - one
     does now;
   - an explicit `check_timeout: 0s` silently meant "use the flag default" - the
     exact typo-that-does-nothing ADR-067 exists to refuse, so a zero duration now
     refuses like a negative one;
   - `--target` with a typed `--source` but no `--from` built a job pointing a dir
     source at a restic URL, failing downstream with an error that never said why.
     It now refuses, naming `--from`; a full source replacement drops the config
     source's env, while `--from` alone repoints the source and keeps it;
   - the precedence chain had no automated coverage. `TestJobFromConfigPrecedence`
     now walks it flag by flag, and a docker-free test drives `runAll` with a
     cancelled context to hold the interrupted-sweep behaviour down.

**Not done, deliberately:** README still says nothing about `restored.yaml` - the
quick start stays `--recipe`, and the config's user documentation belongs with the
`docs/security.md` pass. `recipes.yml` and `recipe-health.yml` are untouched.

**Evidence.** The new surface against real stacks - a three-target config (restic
source with `password_file`, dir source over a kept staging tree, and a deliberately
broken target), run with `--all`:

```text
$ ./bin/restored check --all --config .../restored.yaml
  target vault-restic  PASS                6.7s
  target vault-dir     PASS                4.8s
  target broken        ERROR              0.79s  · required input "data": no matching fi...

  3 targets: 2 passed, 0 unusable, 1 errored, in 12.2s
$ echo $?
2
$ ./bin/restored check --target vault-dir --config .../restored.yaml --json > target.json
$ echo $?
0
$ python - # target.json
single doc verdict: PASS | has runs key: False
```

The refusals:

```text
$ ./bin/restored check
restored: --recipe is required (or --target <name> / --all, which read restored.yaml)
$ ./bin/restored check --recipe gitea --all
restored: --recipe, --target and --all are mutually exclusive
$ ./bin/restored check --target gitea        # no restored.yaml anywhere
restored: no restored.yaml found (searched: restored.yaml, C:\Users\kadyr\.config\restored\restored.yaml, \etc\restored\restored.yaml)
(exit 2, all three)
```

The suites, race first (in the CI image, since this host has no C toolchain):

```text
$ docker run --rm -v "C:/My/Projects/Work/restored:/src" \
    -v "C:/Users/kadyr/go/pkg/mod:/go/pkg/mod" -w /src golang:1.27 go test ./... -race
ok  	github.com/spelingbee/restored/internal/config	1.163s
... 13 packages, all ok ...
RACE_EXIT=0

$ go test -tags integration ./... -timeout 30m
ok  	github.com/spelingbee/restored/internal/runner	271.191s
... all ok ...
INTEGRATION_EXIT=0

$ gofmt -l .
$ go vet ./... && go vet -tags integration ./...
$ golangci-lint run && golangci-lint run --build-tags integration
0 issues.
0 issues.
$ ./scripts/lint-english.sh
lint-english: ok
$ docker ps -aq --filter "label=com.restored.run" | wc -l
0
```

After the review fixes, the battery was re-run on the final code: 13 packages ok
uncached, both linters at 0 issues on both tag sets, the race detector green again in
the CI image, and the config suite green with real AWS credentials exported - the
exact environment that failed before:

```text
$ AWS_ACCESS_KEY_ID=AKIAREAL AWS_SECRET_ACCESS_KEY=real \
    go test ./internal/config/ -run TestJobMergesDefaultsAndTarget -count=1
ok  	github.com/spelingbee/restored/internal/config	0.736s
```

### Session 8 - 2026-08-31 - The rename: `restored` is `drillback`

**Goal:** execute the human's stop-point-7 decision of 2026-08-30 - the rename to
`drillback` that `docs/name-check.md` recommended - before the first push makes it
expensive. The owner stays `spelingbee`.

**Done:**

1. **The rename, everywhere the present tense lives** (ADR-070): module
   `github.com/spelingbee/drillback` and every import; `cmd/drillback` and the binary;
   `apiVersion: drillback/v1` in the schema, the loader and all twenty-one recipe
   directories; `${DRILLBACK_*}` placeholders and the `drillback` internal network in
   every compose file and the scaffold generator; `com.drillback.run` labels and
   `drillback-<runid>` projects and workspaces; `drillback.yaml` with `/etc/drillback`
   and `$XDG_CONFIG_HOME/drillback` in the search order; SPEC.md, README.md,
   CONTRIBUTING.md, the schemas' `$id`s, goreleaser, Dockerfile, install.sh, the
   workflows, the issue templates, the demo scripts. Historical documents - session
   logs here, the ADRs, the review reports, the drill's captured outputs - keep the
   name the runs were made under, deliberately.
2. **GitHub**: the private repository is renamed `spelingbee/drillback` (the old name
   redirects), `origin` updated. Still private; stop point 4 untouched.
3. **The demo texts re-captured** from real runs of the renamed binary - not edited -
   and spliced into README.md by `scripts/capture-demo.sh`, as the rule requires.
4. **Not done, and flagged**: `docs/demo/demo.gif` still shows the old name - `vhs` is
   not on this host - so re-rendering it is on the pre-launch list. The local working
   directory (`C:\My\Projects\Work\restored`) also keeps its name: a host path outside
   the repository, the human's to move (CLAUDE.md's container commands use the real
   path and stay correct).

#### A mistake, recorded

The mechanical rename ran `sed -i` over the whole live file set - **including five
binary test assets**. GNU sed dropped one byte from each (a 70-byte PNG became
69 bytes of `data`; the drill-canary MP3 lost its last byte), and the damage was
invisible to every text-level check: build, unit suite, race, linters and
`recipe validate` all green. The full round-trip sweep caught it: Gogs silently
rejected the corrupt avatar upload and failed `avatar-file-present`; listmonk
answered 500 to the corrupt media upload and errored its seed. Both reproduced
identically in a second, uncontended sweep - and ConvertX, Gotify and Navidrome
*passed* with corrupt assets, because nothing in their checks decodes the file. The
five binaries were restored from git (their bytes contain no name), and all five
affected recipes re-ran green.

The integration suite then caught a second rename miss the same way: the fixture
recipe at `testdata/recipes/fixture` still expected `body_matches: "restored
fixture"` against a page the renamed test now writes as "drillback fixture page" -
green under every static check, red the moment a container actually served the page.
Fixed, with the fixture's network name aligned in the same commit.

Two lessons, both this project's own rules restated: the checks that run the real
thing catch what every static check misses - twice in one session - and a blanket
`sed -i` belongs on a file list that excludes what `grep -I` calls binary, the same
mistake `lint-english.sh` already made and fixed in session 4.

#### What the first CI run ever caught

The push was the first time any workflow actually executed, and the first run failed
three jobs - none of them rename bugs, all of them debts only a real runner could
collect:

1. **`unit (windows-latest)`: a real security hole in the symlink sanitiser.**
   GitHub's Windows runners may create symlinks, so `TestSanitiseNeutralisesEscapingSymlinks`
   ran on Windows for the first time in the project's life - every host before it
   either skipped (no privilege, like this machine) or was POSIX - and it failed:
   on Windows, a rooted-but-driveless symlink target (`\etc\shadow`) is not
   `filepath.IsAbs`, so `Sanitise` glued it under the link's own directory,
   containment passed, and the escaping link survived to be mounted. Fixed by
   resolving rooted targets against the workspace's volume, the way the OS does;
   the POSIX suite re-ran green in the CI image container.
2. **`lint`: exit 126.** Every shell script was committed from Windows as mode
   100644; `./scripts/lint-english.sh` was not executable on the runner.
   `git update-index --chmod=+x` on all of `scripts/` and `install.sh`.
3. **`integration`: `permission denied` on `repo/keys`.** The demo scripts run
   restic in a container as the image's root, so on a real Linux host the
   repository belongs to root and the `drillback check` that follows cannot read
   it - invisible on Windows, where the daemon maps no ownership. The harness had
   already solved this exact problem (`containerUser`, ADR-051); `scripts/lib.sh`
   now does the same (`-u "$(id -u):$(id -g)"` on Linux, `--no-cache`), and
   `scripts/demo.sh` re-ran green locally after the change.

A rename-review agent also swept the live tree afterwards and found thirteen
stragglers the collocation seds missed - among them the Ctrl-C message printing both
names, `recipes-wanted.sh` introducing the tool under the old name in every issue it
would file, and a double-rename that left the release checklist recommending
"`drillback` over `drillback`" - plus three stale claims that predate the rename
(README said five recipes ship; there are twenty). All fixed.

Runs three to five peeled the deepest debt of all, one layer per run - **a teardown
bug every Linux user would have hit on their first gitea-shaped run**:

4. The application container writes as its own uid (Gitea leaves `queues/` at 0700
   under uid 1000), and the caller's `RemoveAll` cannot delete what it cannot even
   stat - so teardown itself died with `permission denied` and the run exited 2 with
   a workspace left behind. Not a CI quirk: a product bug invisible on Windows,
   where the daemon maps no ownership. The fix is `compose.Client.Scrub`: when
   removal fails on a non-Windows host, run the pinned curl helper with the
   workspace mounted, `chmod -R a+rwX`, and remove again; the runner's teardown and
   the harness's stage B both use it.
5. The first scrub changed nothing, and the fourth run said so: the helper image
   declares its own unprivileged `USER`, so chmod ran as uid 100 against uid 1000's
   files and exited nonzero - and Scrub read only the exec error, not the exit code.
   `--user 0:0`, and a nonzero chmod is now the failure it always was.

The fifth run is the first fully green CI in the project's life:

```text
$ gh run view <run 5> --json jobs
lint:success  generated:success  unit (ubuntu-latest):success
unit (macos-latest):success  unit (windows-latest):success  integration:success
```

#### Evidence

The registry, both stages, all twenty recipes under the new name. The first sweep ran
concurrently with the demo capture and the race container; the second ran alone and
reproduced the same two asset failures, which is what proved they were not flakes:

```text
$ ./bin/drillback recipe test ./recipes/... (all twenty) --timeout 20m
  20 recipes: 18 passed, 1 failed, 1 errored, in 26m39s   # corrupt assets: gogs, listmonk
$ git checkout -- <the five binary assets>
$ ./bin/drillback recipe test ./recipes/gogs ./recipes/listmonk ./recipes/convertx \
    ./recipes/gotify ./recipes/navidrome
  5 recipes: 5 passed, 0 failed, 0 errored, in 5m25s
```

Together: twenty of twenty round-trip green on this commit. The rest of the battery:

```text
$ go build ./... && go test ./... -count=1
(13 packages ok)
$ docker run --rm ... golang:1.27 go test ./... -race
13 ok, RACE_EXIT=0
$ gofmt -l . ; go vet ./... && go vet -tags integration ./...
(nothing)
$ golangci-lint run && golangci-lint run --build-tags integration
0 issues.  0 issues.
$ ./scripts/lint-english.sh
lint-english: ok
$ ./bin/drillback recipe validate ./recipes/*/ --strict; echo $?
... 21 lines, all ok ... 0
$ ./bin/drillback version | head -1
drillback 0.1.0-dev
$ goreleaser check
1 configuration file(s) validated
$ ./scripts/capture-demo.sh
wrote: pass.txt 26 lines, fail.txt 52 lines, kuma.txt 27 lines
$ go test -tags integration ./... -timeout 30m
ok  	github.com/spelingbee/drillback/internal/runner	232.480s
... all ok ... INTEGRATION_EXIT=0
$ docker ps -aq --filter "label=com.drillback.run" | wc -l
0
```

---

### Session 8, addendum - launch preparation on the human's "do everything"

Done after the rename landed, all on 2026-08-31:

1. **Repository settings** (checklist step 1): Discussions on, issue chooser verified,
   branch protection on `main` with the six real check names pinned to the GitHub
   Actions app, workflow permissions read+write. Two settings GitHub refuses on a
   private repository (private vulnerability reporting, first-time-contributor
   approval) moved to the go-public moment, recorded in the checklist.
2. **Labels applied** (`scripts/labels.sh --apply`) - which paid for itself within the
   hour, below.
3. **The demo GIF re-rendered** under the new name via `render-demo.sh`'s documented
   container route, verified frame by frame: act one ends PASS 5/5 exit=0, act two
   RESTORE UNUSABLE exit=1. Open question 8 (the 24h/48h review promise) answered by
   the human: it stands.
4. **The first `recipe-health` run ever** (workflow_dispatch): **17 of 20 recipes
   green on Linux runners**, and the pipeline worked exactly as designed - the three
   failures each auto-filed a `recipe-broken` + `help wanted` issue with the labels
   from step 2. The three failures are one class, diagnosed and reproduced locally
   under a Linux uid (the repro rig is in issue #1): **an application image that
   re-owns its mounted data directory on startup (FreshRSS chowns to 33:33/0770)
   defeats every host-side read that follows** - the sqlite loader's `stat`, the
   `file`-kind checks - invisible on Windows where the daemon maps no ownership.
   freshrss and trilium are exactly this; nextcloud is almost certainly the same
   class through its prepare service. Diagnoses are on issues #1-#3.

**The three red recipes block the tag** (release checklist: all recipes green on the
latest health run) **and nothing else**: going public is not blocked.

---

### Session 9 - 2026-09-03 - The workspace is read through the daemon

The brief, from the human's "what is next" after the launch-day settings: clear the
one thing blocking the tag. Found on arrival, and now recorded above: the repository
was already public, the labels and all 76 issues already existed, and the two
public-only settings had just been switched on by the human; verified via the API
(`approval_policy: first_time_contributors`, `dependabot_security_updates: enabled`,
private vulnerability reporting showing *Disable* in the UI). `scripts/labels.sh
--apply` re-run was a no-op, as designed.

**Done:**

1. **The class behind issues #1-#3, reproduced first.** Docker Desktop was not
   running; started it, built the Linux binary, and ran issue #1's rig (`docker:cli`
   plus `apk add restic`, uid 1001, `TMPDIR=/var/lib/drillback-health`). The failure
   is exactly CI's:

   ```text
   $ ./bin/drillback-linux recipe test ./recipes/freshrss --stage b   (as uid 1001)
       load db    FAILED     0.00s
                  input "db": opening db.sqlite: stat
                  /var/lib/drillback-health/drillback-v6ebhqgm/inputs/data/users/drilladmin/db.sqlite:
                  permission denied
       RESTORE UNUSABLE  0/5 checks  ·  total 4.1s  ·  teardown ok
   ```

2. **`compose.Reader`**, the one place that reads the workspace after `compose up`:
   root in the curl helper, inputs bound read-only at `/inputs`. `List` (`find
   -maxdepth N -exec stat`) backs `file` checks, judged by `observeTree` the way the
   host used to judge the tree, with `path.Match` on `filepath.Glob`'s own pattern
   syntax. `Fetch` copies a SQLite file plus `-wal`/`-shm` into the workspace's new
   `reads/` and the in-process opener reads the copy, so ADR-040 stands. A missing
   path is helper exit 3 and comes back as `exists: false` / `fs.ErrNotExist`, which
   stage A's refusal verdict depends on. No host fast path, no platform branch:
   Windows takes the same route. ADR-071 records it, and records the rejected
   alternative - `chmod -R a+rX` on the live tree - which Nextcloud forbids outright.

3. **Verified under the rig, all three formerly red recipes:**

   ```text
   $ ./bin/drillback-linux recipe test ./recipes/freshrss --timeout 10m   (as uid 1001)
     stage A  negative: the checks must fail against an empty sta… PASS      13.4s
              3 of 5 checks failed against an empty stack: feeds-present,
              entries-present, user-config-present
     stage B  round trip: seed, export, back up, restore, check    PASS      29.8s
              the round trip restored and all 5 checks passed
     PASS   freshrss in 43.4s

   $ ./bin/drillback-linux recipe test ./recipes/trilium ./recipes/nextcloud --timeout 15m
     PASS   trilium in 1m15s        (stage A: 2 of 4 checks failed against an empty stack)
     PASS   nextcloud in 2m01s      (stage A: 3 of 6 checks failed against an empty stack)
     2 recipes: 2 passed, 0 failed, 0 errored, in 3m17s
   ```

   Stage A got stronger as a side effect: the loader now reads the application's
   fresh database instead of dying on its permissions, so freshrss goes from
   PASS-BY-STARTUP-REFUSAL to three checks genuinely failing.

4. **Verified on Windows**, the same binary path:

   ```text
   $ ./bin/drillback recipe test ./recipes/uptime-kuma --timeout 10m
     PASS   uptime-kuma in 1m50s    (stage B: all 6 checks passed)
   ```

5. **The usual gates, locally:**

   ```text
   $ gofmt -l .                       (nothing)
   $ go vet ./... && go vet -tags integration ./...
   VET_OK
   $ go test ./...
   ok      github.com/spelingbee/drillback/internal/check          2.264s
   ok      github.com/spelingbee/drillback/internal/compose        0.993s
   ok      github.com/spelingbee/drillback/internal/loader         2.240s
   ok      github.com/spelingbee/drillback/internal/runner         1.613s
   ok      github.com/spelingbee/drillback/internal/workspace      0.993s
   (every other package ok or without tests; nothing skipped)
   $ golangci-lint run ./...
   0 issues.
   $ ./scripts/lint-english.sh
   lint-english: ok
   ```

   `-race` was not run on this host (no C compiler, as always); CI's `unit` job runs
   it.

6. **Pull request #77** from `session-9-daemon-reads`, "Fixes #1, #2, #3". Because
   the change touches `internal/`, `recipes.yml` on the PR ran the whole twenty-recipe
   matrix on Linux runners. Every check passed:

   ```text
   $ gh pr checks 77 --watch
   integration            pass  3m3s     (run 33712501300)
   lint                   pass  49s
   unit (ubuntu-latest)   pass  1m10s    <- the -race run this host cannot do
   unit (macos-latest)    pass  36s
   unit (windows-latest)  pass  1m21s
   test (freshrss)        pass  1m12s    (run 33712501221, 20 recipes)
   test (trilium)         pass  1m46s
   test (nextcloud)       pass  2m17s
   test (beszel) ... test (vaultwarden): pass, all 17 others
   verdict                pass  4s
   ```

   And `recipes.yml` dispatched on the branch for exactly the three
   (run 33712510323): `test (freshrss)`, `test (trilium)`, `test (nextcloud)` all
   `completed success`, `verdict` success.

7. **Merged (rebase, two commits, `8f683e8` and `ca12a66`), issues #1-#3 closed by
   the merge, and `recipe-health` dispatched on the new `main`:**

   ```text
   $ gh run view 33713094057 --json status,conclusion,jobs
   completed success
   list: success
   health (freshrss): success
   health (trilium): success
   health (nextcloud): success
   health (beszel) ... health (vaultwarden): success, all 17 others
   ```

   **20 of 20.** Release checklist item 2 ("all recipes green on the latest
   `recipe-health` run") is met for the first time. One mistake on the way, recorded
   so it is not repeated: the first merge attempt failed because the docs commit had
   just re-triggered CI, and the command chain behind it still checked out `main`
   and dispatched `recipe-health` against the old code. That run (33712835979) was
   cancelled within two minutes, before any health job reached the step that
   comments on an issue; #1-#3 carry no stray comment. Chain merge-then-dispatch
   with `&&`, never `;`.

**Then the human said "do it", and the release followed, all on 2026-09-03:**

8. **Pre-tag items of SPEC.md 12.6.** CHANGELOG.md's Unreleased folded into a dated
   0.1.0 with a highlights block and a Known gaps list that stopped claiming config
   is unimplemented; README's pre-release note replaced. `scripts/capture-demo.sh` on
   this host rewrote `docs/demo/*.txt` and the README blocks from real runs; the GIF
   re-rendered through the vhs container route with `DRILLBACK_BIN=./bin/drillback-linux`
   (800,541 bytes), and two frames pulled with ffmpeg and looked at: act one ends
   `PASS 5/5 checks`, `exit=0`; act two `RESTORE UNUSABLE 2/5 checks`, `exit=1`.
   Commit `614ad89`; `ci.yml` run 33716474505 green on it.

9. **The tag, twice.** `v0.1.0` at `614ad89` failed in `release.yml`:

   ```text
   x release failed after 1m39s  error=exec: "syft": executable file not found in $PATH
   ```

   The snapshot job ran `build --snapshot`, which stops before SBOMs, so the dry run
   had never proved the step that failed. Fixed in `7d93053`: `anchore/sbom-action/
   download-syft` pinned by commit in both jobs, snapshot runs `release --snapshot`.
   Dispatched the snapshot (run 33716984493, green), then deleted and re-created the
   tag on `7d93053` - the tag was ten minutes old, no release existed, nothing had
   been downloaded. Run 33717166107 green; draft with 13 assets.

10. **Checklist 12.6.4, against the draft's own assets** (drafts have no public URL,
    so `gh release download`, then):

    ```text
    $ sha256sum -c checksums.txt        all 12: OK
    $ docker run --rm -v .../linux:/rel:ro -v .../recipes:/recipes:ro alpine:3.20 \
        sh -c '/rel/drillback version && /rel/drillback recipe validate /recipes/*/ --strict'
    drillback 0.1.0
      commit:    7d9305399d173b135a64fb377b4ab5b46804b2ce
      recipes:   20 bundled
    ok       /recipes/vaultwarden/      (and the other nineteen)
    ```

    goreleaser's generated notes were a fifty-line commit list; replaced with the
    CHANGELOG section plus the footer, and `.goreleaser.yaml` now `changelog:
    disable: true`. Published 05:06 UTC (stop point 6, on the same "do it").

11. **The install path users will take, from a clean container:**

    ```text
    $ docker run --rm ubuntu:24.04 sh -c '... su tester -c "curl -fsSL \
        https://raw.githubusercontent.com/spelingbee/drillback/main/install.sh | sh \
        && ~/.local/bin/drillback version"'
    drillback 0.1.0
      commit:    7d9305399d173b135a64fb377b4ab5b46804b2ce
      platform:  linux/amd64
      recipes:   20 bundled
    ```

    `go install ...@latest` also builds and runs, but reports `0.1.0-dev` with an
    unknown commit: ldflags do not reach it. Filed as #78, `good first issue`.

12. **The container image, built and pushed.** `docker build` with the checklist's
    exact command; `version` inside it says `drillback 0.1.0`, restic 0.18.0. The
    `gh` token had no `write:packages`; the human added it, and then:

    ```text
    $ gh auth token | docker login ghcr.io -u spelingbee --password-stdin
    Login Succeeded
    $ docker push ghcr.io/spelingbee/drillback:0.1.0
    0.1.0: digest: sha256:539cac8391ca2daac28de508ba36e77672f444d0041eee8e30e90228ee68888d size: 856
    $ docker push ghcr.io/spelingbee/drillback:latest
    latest: digest: sha256:539cac8391ca2daac28de508ba36e77672f444d0041eee8e30e90228ee68888d size: 856
    $ gh api user/packages/container/drillback --jq .visibility
    private
    ```

    The package is private and not linked to the repository, and GitHub exposes no
    API for either: Package settings > Change visibility > Public, and connect the
    repository, are the human's, before `docs/docker.md`'s `docker run` line works
    for anyone else.

13. **The tap.** `spelingbee/homebrew-tap` created public; `Casks/drillback.rb` is
    goreleaser's own rendering (snapshot `release` in the `goreleaser/goreleaser`
    container, which carries syft; the stale `dist/` from session 4 had to be removed
    first) with `version "0.1.0"` and the release's four tar.gz checksums substituted
    and cross-checked against `checksums.txt`. Pushed with LF endings (verified: zero
    CR bytes in the blob). **Not verified with `brew install`**: no macOS here.
    `skip_upload` stays `true` until `HOMEBREW_TAP_GITHUB_TOKEN` exists.

**Not done, deliberately:** the announcement (stop point 3). Where and with what
words is a conversation, not a session's decision.

---

## Next steps

In order. Each is sized to be finishable and committable on its own.

### Session 9 - host-side reads of container-owned trees - done

**Answered by session 9.** `compose.Reader` reads the workspace through the daemon
once the application is up; `file` checks, the sqlite integrity check and
`driver: sqlite` queries all go through it; ADR-071 records it. freshrss, trilium and
nextcloud pass both stages under the uid-1001 rig from issue #1. See the session 9
log entry for the evidence and for the CI runs.

### The tag - a human's

The three red recipes were the only thing in the way of stop point 1. Once
`recipe-health` on `main` is 20 of 20 with the session 9 change merged, the release
checklist's step 4 is the next thing, and it is not a session's to do.

### `internal/config` - done

**Answered by session 7.** `restored.yaml`, sources, targets, the precedence chain,
`--config`, `--target`, `--all` and the `--all` report shape all exist and are proven
against real stacks; `nudge/config.go` is absorbed into `config.NudgeSilenced` with
its behaviour unchanged; the `--help`-versus-SPEC-section-2 diff is adjudicated as
ADR-069. See the session 7 log entry.

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

Re-render `docs/demo/demo.gif` with `vhs` (it still shows the old name; the tape file
is already renamed), then `docs/security.md`, `smoke.yml`, and a `mysql-dump` input kind - `recipe init --compose`
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

1. **The name and the owner.** **Answered by the human, 2026-08-30**: `drillback`,
   under `spelingbee`, exactly what `docs/name-check.md` recommended. Session 8
   executed the rename before anything was published; ADR-070 records the execution
   and its boundary (history keeps the old name). Closed.
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
8. **Answered by the human, 2026-08-31.** The promise was put to them with the option
   to soften before going public; the instruction was to proceed with everything as
   written, so "first response within 24 hours, merged within 48 when CI is green"
   stands. The weekly maintainer session is not a 24-hour cadence, so the clock is
   the human's - with the mitigations sessions 4 and 6 already built (a single
   `verdict` check to read, and contributors warned their first run waits).
   The original question, kept for the record:
   **The review promise in CONTRIBUTING.md is a promise.** "First response within
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
