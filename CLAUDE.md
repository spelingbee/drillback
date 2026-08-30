# CLAUDE.md

Instructions for any session working on this repository.

**Read [PROGRESS.md](PROGRESS.md) first.** It is the handover: what exists, what is next,
what is blocked. Then read [DECISIONS.md](DECISIONS.md) before proposing anything
architectural — 50 decisions are already made, and re-deciding one costs a session.
[SPEC.md](SPEC.md) is the contract for v0.1.

---

## Commands

These run, with one exception noted under *On this machine*: there is no `make` here,
so the `make` targets are what CI uses and the bare commands are what to type locally.
Anything below that stops being true is a bug in this file.

```sh
# build
make build                  # go build -ldflags ... -o bin/drillback ./cmd/drillback
go build ./...

# test
make test                   # go test ./... -race
go test ./...               # green WITHOUT docker or restic installed
make test-integration       # go test -tags integration ./... -timeout 30m  (needs docker)
go test ./internal/recipe/ -run TestSchemaRejects -v
go test ./internal/report/ -update     # rewrite the golden renderings

# lint
make lint                   # gofmt -l, go vet (twice), golangci-lint run
gofmt -l .                  # must print nothing
go vet -tags integration ./...   # build-tagged files compile nowhere else
./scripts/lint-english.sh   # fails on non-ASCII outside the allowlist

# recipes
./bin/drillback recipe validate ./recipes/*/ --strict
./bin/drillback recipe show gitea --inputs-only
./bin/drillback recipe init myapp --compose ~/docker/myapp/docker-compose.yml
./bin/drillback recipe test ./recipes/gitea      # the round trip, both stages
make recipe-test                                # all five, in sequence

# generated files - never edit these by hand
make docs                   # docs/recipe-spec.md, from schema/recipe.schema.json
make recipes-index          # recipes/README.md and the table in README.md
make check-generated        # what CI diffs

# demo output - never write this by hand
make capture-demo           # scripts/capture-demo.sh > docs/demo/*.txt, and splices
                            # them into README.md between the markers
./scripts/demo.sh           # PASS, exit 0
./scripts/demo-broken.sh    # RESTORE UNUSABLE, exit 1  (make demo-broken negates it)
./scripts/demo-kuma.sh      # PASS, exit 0
```

### On this machine

The toolchain is not on the default PATH. Every command above assumes:

```sh
export PATH="/c/My/Projects/Work/gotool/go/bin:/c/Users/kadyr/go/bin:$PATH"
```

**There is no `make` on this host.** The Makefile is correct and is what CI runs; here,
run the command a target wraps:

```sh
go build -o bin/drillback ./cmd/drillback     # make build
go test ./...                               # make test, minus -race, see below
go test -tags integration ./... -timeout 30m  # make test-integration
./scripts/capture-demo.sh                   # make capture-demo
go run ./tools/gen recipe-spec > docs/recipe-spec.md    # make docs
go run ./tools/gen recipes-index > recipes/README.md    # make recipes-index,
go run ./tools/gen readme-table                         #   which runs both
```

Docker commands need `MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'` exported first, or
Git Bash rewrites `/src` into `C:/Program Files/Git/src` before the daemon sees it.

There is no C compiler here, so `-race` cannot run on the host. Run it in the same
image CI uses:

```sh
docker run --rm -v "C:/My/Projects/Work/restored:/src" \
  -v "C:/Users/kadyr/go/pkg/mod:/go/pkg/mod" -w /src golang:1.27 go test ./... -race
```

Windows will not create symlinks for an unprivileged user, so
`TestSanitiseNeutralisesEscapingSymlinks` skips here. It is the security-critical path,
so do not leave it unrun:

```sh
CGO_ENABLED=0 GOOS=linux go test -c -o /tmp/workspace.test ./internal/workspace
docker run --rm -v /tmp:/t alpine:3.20 /t/workspace.test -test.v
```

PROGRESS.md records how restic and golangci-lint were installed.

## Conventions

### Commit after every green step

A commit is the unit of recoverable work. Commit when something builds and its tests
pass, not when a feature is finished. Small commits with real messages; the body says
*why*, since the diff already says what.

Never commit red. If a step cannot be made green, commit the work-in-progress on a
branch and write down in PROGRESS.md exactly what is broken and what was tried.

### English only

Every file in this repository — code, comments, docs, commit messages, test fixture
names — is in English. No exceptions. CI enforces it (ADR-012).

### Never hand-write demo output

Any terminal block that claims to show what the tool prints must be captured from a real
run by `scripts/capture-demo.sh` into `docs/demo/*.txt`, and included from there.

SPEC.md § 5.1 contains hand-written mocks. They are labelled as design mocks and they
are **never** to be copied into README.md, the website, an issue, or a release note. If
you need real output, run the tool.

### Isolation is not negotiable

Never `--privileged`. Never `network_mode: host`, `pid: host`, `ipc: host`. Never a bind
mount to an absolute host path. Never a published port. Never write, read, or delete
anything outside the run workspace.

These are ADR-009 and ADR-014 and they are enforced by a schema, not by discipline. If a
recipe or a test seems to need one of them, that is a finding to record — not a case to
make an exception for.

### Evidence, not assertion

Every claim in PROGRESS.md that something works must include the **exact command** and
the **tail of its real output**. Like this:

```text
$ go test ./internal/recipe/... -race
ok      github.com/spelingbee/drillback/internal/recipe          0.412s
ok      github.com/spelingbee/drillback/internal/recipe/safety   0.198s
```

Not: "tests pass", "should work now", "verified the schema". If the command was not run,
say it was not run. An unverified claim in the handover is worse than an admitted gap,
because the next session builds on it.

The same applies to anything reported to a human. Do not describe work as complete
without having run the thing that proves it.

### No fake completion

The following are blockers to report, never evidence of progress:

- a `TODO`, `FIXME`, or "will implement later" left in code being called done;
- `t.Skip()`, `test.skip`, or `.only` in a test suite claimed as passing;
- a stub that returns a zero value where the real implementation belongs;
- an unimplemented branch that silently falls through.

Before saying a step is finished, read the diff for these. If one is there, either
implement it or report it explicitly as incomplete.

### Do not ask questions — decide, record, continue

Where the spec leaves a choice, make it, write an ADR in DECISIONS.md with context,
decision and consequences, and keep going. A session that stalls waiting for an answer
has produced nothing; a session that decides and records has produced a decision that a
human can cheaply reverse.

The exception is the stop-points below, which are not questions but checkpoints.

### If blocked for more than 30 minutes, move on

Write what was tried and why it failed in PROGRESS.md § Blocked, and pick up the next
item. Do not spend a session on one wall. A documented blocker is a contribution; an
undocumented hour is not.

### Style

- Standard Go. `gofmt`, `go vet`, `golangci-lint`. No custom style.
- Comments explain *why*. The code says what.
- Errors are wrapped with context (`fmt.Errorf("restoring input %q: %w", name, err)`) and
  never discarded.
- No new dependency without a line in DECISIONS.md justifying it. The dependency budget
  is small on purpose: cobra, a YAML parser, a JSON Schema validator,
  `modernc.org/sqlite`, and as little else as possible.
- Respect the package boundaries in SPEC.md § 13.1. In particular: `internal/recipe`
  imports nothing else from `internal/`, `internal/report` does no I/O beyond its writer,
  and only `internal/compose` and `internal/source/restic` shell out.

---

## Stop points

At each of these, **stop and get explicit human sign-off**. Do not proceed, do not do it
"provisionally", do not do it and then ask.

1. **Before tagging any release.** Tags are public and permanent. Work through the
   release checklist in SPEC.md § 12.6, present it, and wait.
2. **Before filing anything upstream.** An issue, a PR, or a bug report on another
   project — Docker, restic, an application whose image a recipe uses — is this project
   speaking in public to people who did not ask. Draft it, show it, wait.
3. **Before posting anywhere.** Reddit, Hacker News, a forum, a mailing list, a social
   account, a Discord. This includes replying to something about this project. The
   launch is a one-shot resource; it is not a session's decision to spend it.
4. **Before making the repository public**, and before adding a collaborator, a webhook,
   a deploy key, or an Actions secret.
5. **Before spending money.** A domain, a registry plan, a hosted runner, a paid service.
6. **Before publishing anything to a package registry.** ghcr, a Homebrew tap, npm, a
   GitHub Release asset. Building the artifacts is fine; publishing them is not.
7. **The name.** Settled, twice, and closed. Session 2 decided `restored` under the
   owner `spelingbee`, under its brief's instruction to decide rather than ask
   (ADR-036). On 2026-08-30, before anything was published, the human chose the rename
   `docs/name-check.md` had recommended, and session 8 executed it: the project is
   **`drillback`** - module, binary, recipe format, placeholders, repository - with the
   execution and its boundaries (historical documents keep the old name) recorded as
   ADR-070. Do not rename again without a human's explicit instruction.

Everything else — writing code, writing tests, running tests, refactoring, adding
recipes, adding ADRs, committing to a local branch — proceeds without asking.

---

## What this project is optimising for

The success metric is **the number of distinct external contributors with merged PRs**,
not stars, not features, not lines of code.

When a design choice is ambiguous, prefer the one that makes a recipe cheaper to
contribute:

- a clearer error message over a cleverer implementation;
- a mechanical check that lets a maintainer merge without judgement, over a convention
  that requires review;
- a scaffold that generates the right thing, over documentation explaining the right
  thing;
- a smaller `expect` vocabulary that a reviewer can hold in their head, over a general
  expression language.

The round-trip harness (SPEC.md § 7) is the load-bearing piece of this. It exists so a
stranger's recipe can be trusted without a maintainer understanding their application.
Anything that weakens it — a skip flag, an exception for a hard recipe, an "advisory"
mode — is removing the thing that makes the contribution flow work at all.
