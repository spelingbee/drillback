# Decisions

Architecture Decision Records for `restored`. One entry per decision, newest ADR number
last. Format: **Context** (what forced a choice), **Decision** (what we chose),
**Consequences** (what it costs, including the bad parts).

Statuses: `accepted` · `superseded by ADR-nnn` · `proposed`.

ADR-001 through ADR-012 are the fixed decisions handed down with the brief; they are
recorded here for traceability and are not to be relitigated without a superseding ADR.
ADR-013 onward were made while writing [SPEC.md](SPEC.md).

| # | Title | Status |
|---|---|---|
| [001](#adr-001-go-with-cobra-single-static-binary) | Go with cobra, single static binary | accepted |
| [002](#adr-002-apache-20) | Apache-2.0 | accepted |
| [003](#adr-003-docker-engine--compose-v2-as-the-only-runtime) | Docker Engine + compose v2 as the only runtime | accepted |
| [004](#adr-004-restic-and-dir-as-the-only-v01-sources) | restic and dir as the only v0.1 sources | accepted |
| [005](#adr-005-three-input-kinds-dir-sqlite-postgres-dump) | Three input kinds: dir, sqlite, postgres-dump | accepted |
| [006](#adr-006-recipes-declare-logical-inputs-not-user-paths) | Recipes declare logical inputs, not user paths | accepted |
| [007](#adr-007-round-trip-self-validation-of-recipes) | Round-trip self-validation of recipes | accepted |
| [008](#adr-008-zero-friction-contribution-nudge) | Zero-friction contribution nudge | accepted |
| [009](#adr-009-hard-isolation-rules) | Hard isolation rules | accepted |
| [010](#adr-010-three-exit-codes-and-a-stable-json-report) | Three exit codes and a stable JSON report | accepted |
| [011](#adr-011-a-community-extensible-hint-catalog) | A community-extensible hint catalog | accepted |
| [012](#adr-012-english-only-and-no-hand-written-demo-output) | English only, and no hand-written demo output | accepted |
| [013](#adr-013-apiversion-restoredv1-not-a-domain-qualified-group) | apiVersion `restored/v1`, not a domain-qualified group | accepted |
| [014](#adr-014-compose-safety-is-an-allowlist-enforced-before-anything-starts) | Compose safety is an allowlist, enforced before anything starts | accepted |
| [015](#adr-015-every-recipe-network-must-be-internal-true) | Every recipe network must be `internal: true` | accepted |
| [016](#adr-016-http-checks-run-from-a-helper-container-not-from-the-host) | HTTP checks run from a helper container, not from the host | accepted |
| [017](#adr-017-inputs-are-always-copied-into-the-workspace-never-mounted-from-source) | Inputs are always copied into the workspace, never mounted from source | accepted |
| [018](#adr-018-escaping-symlinks-are-neutralised-not-rejected) | Escaping symlinks are neutralised, not rejected | accepted |
| [019](#adr-019-the-test-section-is-inert-during-restored-check) | The `test:` section is inert during `restored check` | accepted |
| [020](#adr-020-a-closed-expect-vocabulary-no-expression-language) | A closed `expect` vocabulary, no expression language | accepted |
| [021](#adr-021-postgres-dump-format-detected-from-magic-bytes) | Postgres dump format detected from magic bytes | accepted |
| [022](#adr-022-run-ids-are-8-random-base32-characters-and-label-everything) | Run ids are 8 random base32 characters, and label everything | accepted |
| [023](#adr-023-a-failed-ready-probe-is-exit-1-not-exit-2) | **A failed ready probe is exit 1, not exit 2** | accepted |
| [024](#adr-024-pure-go-sqlite-driver-cgo_enabled0-everywhere) | Pure-Go SQLite driver, CGO_ENABLED=0 everywhere | accepted |
| [025](#adr-025-bundled-recipes-are-embedded-in-the-binary) | Bundled recipes are embedded in the binary | accepted |
| [026](#adr-026-checks-never-stop-early-and-are-never-retried) | Checks never stop early, and are never retried | accepted |
| [027](#adr-027-at-most-one-hint-per-run) | At most one hint per run | accepted |
| [028](#adr-028---json-writes-to-stdout-humans-to-stderr) | `--json` writes to stdout, humans to stderr | accepted |
| [029](#adr-029-the-nudge-requires-a-strict-valid-recipe-and-a-tty) | The nudge requires a strict-valid recipe and a TTY | accepted |
| [030](#adr-030-image-tags-pinned-latest-rejected-digests-optional) | Image tags pinned, `:latest` rejected, digests optional | accepted |
| [031](#adr-031-no-mocked-docker-api) | No mocked Docker API | accepted |
| [032](#adr-032-stage-a-treats-startup-refusal-as-a-pass) | Stage A treats startup refusal as a pass | accepted |
| [033](#adr-033-six-recipes-gate-v01) | Six recipes gate v0.1 | accepted |
| [034](#adr-034-no-standalone-docshintsyaml-file-in-session-1) | No standalone `docs/hints.yaml` file in session 1 | accepted |
| [035](#adr-035-the-github-owner-stays-a-placeholder) | The GitHub owner stays a placeholder | accepted |

---

## ADR-001: Go with cobra, single static binary

**Status:** accepted (fixed decision 1)

**Context.** The tool must run on a self-hoster's machine with no runtime to install,
must cross-compile to Linux, macOS and Windows on both amd64 and arm64, and must shell
out to `docker` and `restic` reliably.

**Decision.** Go (latest stable), CLI built on cobra, distributed as a single static
binary, released with goreleaser.

**Consequences.** Installation is one file, which removes the largest barrier for the
target audience. Cross-compilation is free, provided `CGO_ENABLED=0` holds — which
constrains the SQLite driver (ADR-024). cobra brings a large dependency tree and a
help-output style we do not fully control; accepted, because writing flag parsing by
hand is not where this project's effort belongs.

## ADR-002: Apache-2.0

**Status:** accepted (fixed decision 2)

**Context.** The project wants external contributors and possible downstream packaging
by distributions and vendors.

**Decision.** Apache-2.0, with the copyright line filled as "The restored Authors".

**Consequences.** Permissive enough for anyone to package or embed; the explicit patent
grant is friendlier to corporate contributors than MIT. Slightly longer than MIT and
requires the NOTICE convention if we ever add one. No CLA — contributions are under the
project licence by the DCO-free default of GitHub's terms plus the licence itself.

## ADR-003: Docker Engine + compose v2 as the only runtime

**Status:** accepted (fixed decision 3)

**Context.** The target applications are distributed as compose stacks. Supporting a
second runtime means every recipe has to work twice.

**Decision.** Require Docker Engine and `docker compose` v2 (the plugin, invoked as
`docker compose`, never the deprecated `docker-compose` binary). Must work with Docker
Desktop on the WSL2 backend, rootless Docker, and plain Linux Docker. No non-Docker
fallback.

**Consequences.** One runtime contract, so a recipe that works for the author works for
everyone. Podman users are unsupported even though its Docker-compatible socket will
often work — saying "unsupported" is more honest than shipping untested support.
Rootless Docker introduces UID-shifting permission failures that are common enough to
have their own hint (`permissions/eacces`).

## ADR-004: restic and dir as the only v0.1 sources

**Status:** accepted (fixed decision 4)

**Context.** borg, kopia, rclone and plain tar all have users. Supporting four sources
shallowly would mean none of them is good.

**Decision.** v0.1 supports `restic` (repository and credentials through the standard
`RESTIC_REPOSITORY` / `RESTIC_PASSWORD` / `RESTIC_PASSWORD_FILE` environment; snapshot
selection `latest` by default, or `--snapshot <id>`, `--tag`, `--host`) and `dir` (a
tree that is already restored or exported). Nothing else.

**Consequences.** restic covers the largest share of the audience and has a clean
`--json` interface, so snapshot selection is testable against recorded output rather
than a live repository. `dir` costs almost nothing and makes the tool usable with any
backup system a user can export from, which quietly widens the audience without widening
the surface. borg and kopia land in v0.2 behind the same `source` interface, which is
why that interface exists in v0.1 with only two implementations.

## ADR-005: Three input kinds: dir, sqlite, postgres-dump

**Status:** accepted (fixed decision 5)

**Context.** Self-hosted apps store state as a directory, an SQLite file, or a
PostgreSQL database — usually two of the three.

**Decision.** v0.1 supports `dir`, `sqlite`, and `postgres-dump` (plain SQL through
`psql`, custom format through `pg_restore`, auto-detected). `mysql-dump` is v0.2.

**Consequences.** Covers Gitea, Nextcloud, Vaultwarden, Paperless-ngx, Immich, Miniflux,
Home Assistant and Uptime Kuma without a fourth kind. MySQL/MariaDB users — a real
minority in this audience, but not nobody — cannot write a recipe for their app until
v0.2, and will say so. Accepted: MySQL dumps have enough charset and
`--single-transaction` subtlety to deserve their own work rather than a rushed
appendix.

## ADR-006: Recipes declare logical inputs, not user paths

**Status:** accepted (fixed decision 6)

**Context.** The recipe author knows the application; only the user knows where their
backup keeps it. A recipe hard-coded to the author's paths is useful to exactly one
person.

**Decision.** A recipe declares named logical inputs with a `kind`, a `title`, a
`default_path` (the most common layout, as a guess) and a mount or load target. Users
override with `--input name=path` or `restored.yaml`. Compose reads them as
`${RESTORED_INPUT_<name>}`.

**Consequences.** A recipe is portable across every user of that application, which is
what makes it worth contributing. The cost is one more concept for the author, and a
class of confusing failure when a default path is wrong for a given user — mitigated by
the `restore/path-not-in-snapshot` hint and by `recipe show --inputs-only`, which exists
precisely to answer "what does this recipe want from my backup?".

## ADR-007: Round-trip self-validation of recipes

**Status:** accepted (fixed decision 7)

**Context.** The success metric is merged external contributions. That only scales if
accepting a recipe requires no maintainer judgement about whether its checks are
meaningful. A recipe whose checks pass against an empty database is worse than no
recipe: it manufactures the exact false confidence the tool exists to destroy.

**Decision.** `restored recipe test <dir>` runs two stages. Stage A starts the stack
with empty inputs and requires **at least one check to fail**, otherwise the recipe is
rejected with "recipe has no data-sensitive check". Stage B seeds, exports, backs the
tree into a throwaway restic repository, tears down, and then runs the ordinary
`restored check` against it, requiring **all checks to pass**. CI runs this for every PR
touching `recipes/**`, and the author runs the identical command locally.

**Consequences.** Recipe quality is enforced mechanically, so review is about taste and
correctness rather than about whether the checks mean anything. It is the single most
expensive feature in v0.1 — it needs a seed/export vocabulary, a temporary restic
repository, and roughly a 20-minute CI budget per recipe. It also constrains which
applications can have recipes at all: anything that cannot be seeded and round-tripped
in 20 minutes is out of scope, and that is stated rather than discovered. Accepted
because without it the contribution flow produces volume without value.

## ADR-008: Zero-friction contribution nudge

**Status:** accepted (fixed decision 8)

**Context.** The moment a user's own recipe just proved their backup works is the moment
they are most willing to share it, and the moment they are least willing to read
CONTRIBUTING.md.

**Decision.** After a PASS with a non-bundled recipe, print a prefilled GitHub
`/new/main?filename=…&value=…` URL and a one-line invitation. Fall back to printed
instructions above a 6,000-character encoded URL. Silenceable with `--no-nudge` and with
`nudge: false` in config.

**Consequences.** Turns the highest-intent moment into a link. The risks are real and
bounded: it must never fire in cron or in `--json` (ADR-029 adds a TTY condition), it
must never fire for a recipe CI would reject (ADR-029 adds a strict-validation
condition), and it must be silenceable in the same sentence that shows it. Only
`recipe.yaml` can be prefilled — `compose.yaml` needs a second file the `/new/` route
cannot carry — so the link is a hook into a normal PR flow, not the whole flow.

## ADR-009: Hard isolation rules

**Status:** accepted (fixed decision 9)

**Context.** The tool runs somebody else's compose file over somebody's backup on a
machine with a Docker socket, and it must never damage the running system it is testing
a backup of.

**Decision.** Every run gets its own compose project `restored-<runid>` and its own
internal network. No published ports; HTTP checks run from a helper container on the
run's network. The workspace is a fresh temporary directory and nothing outside it is
ever bind-mounted. `privileged`, `network_mode: host`, `pid: host`, and bind mounts to
absolute host paths are rejected by the validator. Teardown always runs — including on
SIGINT and on panic — unless `--keep`.

**Consequences.** A drill cannot collide with the production stack it is drilling,
cannot expose a half-restored app on a port, and cannot leave debris. Recipes must
address services by compose service name rather than `localhost`, which is a real
adjustment for authors copying an upstream compose file, and is why the schema rejects
loopback URLs with an explanatory message. Some applications genuinely need host
networking and simply cannot have a recipe; that is the correct outcome.

## ADR-010: Three exit codes and a stable JSON report

**Status:** accepted (fixed decision 10)

**Context.** The output has two consumers: a human reading a terminal, and a cron job
deciding whether to wake someone up.

**Decision.** `0` = all checks passed. `1` = restore unusable. `2` = tool or runtime
error. `--json` emits a stable report whose `schema_version`, `verdict`, `exit_code`,
`summary.*` and `checks[].status` fields are frozen for v0.x.

**Consequences.** A cron job can distinguish "your backup is broken" from "your drill is
broken", which is the difference between an alert that gets acted on and one that gets
muted. It puts a real constraint on future changes: the report is an API. See ADR-023
for where the 1/2 line is drawn, which is the contentious part.

## ADR-011: A community-extensible hint catalog

**Status:** accepted (fixed decision 11)

**Context.** The failure modes of a bad restore are finite, well known to people who
have hit them, and completely opaque to people who have not. `relation "repository" does
not exist` is a five-second diagnosis for one person and a lost evening for another.

**Decision.** `docs/hints.yaml` maps RE2 patterns over check output and service logs to
human explanations, with optional diagnostic commands that are printed but never
executed. Shipped embedded; extensible with `--hints FILE`.

**Consequences.** The report teaches instead of only reporting. Adding a hint is the
smallest possible useful contribution — twenty lines of YAML, no Go, no Docker — which
makes it the natural first-timer entry point and a deliberate recruitment channel. Risk:
a confidently wrong hint is worse than none, so every rule needs a matching fixture and
a near-miss fixture in tests, and hints can never affect the verdict.

## ADR-012: English only, and no hand-written demo output

**Status:** accepted (fixed decision 12)

**Context.** An international contributor base needs one language. And demo output that
was written rather than captured is a lie that ages badly — every project with a
beautiful README terminal block has one that no longer matches the tool.

**Decision.** All repository content is in English. No demo output in any document is
hand-written; it is captured from real runs by `scripts/capture-demo.sh` into
`docs/demo/*.txt` and included from there. Design mocks in SPEC.md are permitted but
must be explicitly labelled as mocks that are never to be copied.

**Consequences.** Enforced by a `english-only` CI lint that fails on non-ASCII outside an
allowlist, and by a demo-drift check comparing README blocks against `docs/demo/`. The
capture script becomes a real dependency: the demo cannot be updated without a working
tool and a working recipe, which is the point.

## ADR-013: apiVersion `restored/v1`, not a domain-qualified group

**Status:** accepted

**Context.** Kubernetes-style `apiVersion` values are domain-qualified
(`restored.dev/v1`). `restored.dev` is registered by someone else (see
[docs/name-check.md](docs/name-check.md)), and the project name is not final.

**Decision.** `apiVersion: restored/v1` — no domain.

**Consequences.** No implicit claim to a domain we do not own, and no rewrite of every
recipe if the name or domain changes. Slightly less conventional for readers who expect
the Kubernetes form. If a domain is ever acquired, moving to it is a schema change with
a migration, so this is worth getting right once rather than twice.

## ADR-014: Compose safety is an allowlist, enforced before anything starts

**Status:** accepted

**Context.** ADR-009 names four forbidden constructs. A denylist of four is a denylist
that misses `devices`, `cgroup_parent`, `userns_mode`, `build:`, `extends:`,
`security_opt: seccomp=unconfined`, and long-syntax `type: bind`.

**Decision.** `schema/compose-safety.schema.json` rejects a named set of keys outright,
restricts `cap_add` to a five-entry allowlist, constrains volume left-hand sides to a
named volume or a `${RESTORED_*}` placeholder by regular expression, and forbids
long-syntax bind mounts entirely. Three further rules are enforced in Go because JSON
Schema cannot express them: no YAML tags or anchor cycles, no unresolved `${}`
placeholder after interpolation, and no reference to a service that does not exist.
Validation runs in the RESOLVE state, before any container is created.

**Consequences.** The safety story is a reviewable file rather than scattered `if`
statements, and a contributor can read exactly what is forbidden. It will reject some
legitimate recipes — an app that genuinely needs `NET_ADMIN` cannot have one — and each
such rejection is a deliberate scope decision, not a bug. The allowlist has to be
maintained as compose gains keys; a new key is unlisted and therefore allowed, so the
*keys* list is a denylist even though the *values* lists are allowlists. That asymmetry
is a known weakness and is why the `not/anyOf` list must be reviewed on each compose
major version.

## ADR-015: Every recipe network must be `internal: true`

**Status:** accepted

**Context.** ADR-009 requires an internal network. Nothing stops a recipe from declaring
a second, non-internal one alongside it.

**Decision.** The schema requires `internal: true` on *every* entry under `networks:`,
and forbids `external: true`.

**Consequences.** A running recipe container has no egress at all: it cannot exfiltrate
the backup it was handed, and it cannot phone home. Image pulls still work, because they
happen on the host before containers attach. The visible cost is that applications which
block on outbound requests at startup will hang their ready probe — Gitea needs
`OFFLINE_MODE`, and other recipes will need their equivalent. That is annoying, and it
is a much better failure than silent egress.

## ADR-016: HTTP checks run from a helper container, not from the host

**Status:** accepted

**Context.** With no published ports (ADR-009), the host cannot reach the application.
The alternatives were: publish a port on `127.0.0.1`, or run the HTTP client inside the
run's network.

**Decision.** HTTP probes and checks execute from a pinned helper container attached to
the run's internal network (`curlimages/curl`, pinned by digest and bundled in the
image list). Recipes address services by compose service name.

**Consequences.** Nothing is ever exposed, not even on loopback, so two concurrent runs
cannot collide and a half-restored app is never reachable from the host. Costs a
container start per run and one more image to keep pinned, and means an HTTP check
cannot use Go's HTTP client directly — response inspection happens over the helper's
output, which constrains the `expect` vocabulary to what can be observed that way. The
schema rejects `localhost` URLs with an explicit message, because it is the single most
likely mistake an author copying an upstream compose file will make.

## ADR-017: Inputs are always copied into the workspace, never mounted from source

**Status:** accepted

**Context.** For `--source dir`, bind-mounting the user's existing tree directly would
be faster and would save disk.

**Decision.** Inputs are always materialised inside `<workspace>/inputs/` — restored
there by restic, or copied there for `dir`. No mode mounts a user's real data into a
container.

**Consequences.** `restored` structurally cannot damage the data it is verifying, even
if a recipe or an image is hostile, and even if the user points `--from` at a live
directory by mistake. The cost is real: a 200 GB backup needs 200 GB of free workspace,
and the default workspace is the OS temp directory, which is often a small tmpfs in RAM.
Mitigated with `--workspace`, the `defaults.workspace` config key, and the
`workspace/no-space` hint, which names this exact trap. A future `--link` fast path for
`dir` would have to argue against this ADR explicitly.

## ADR-018: Escaping symlinks are neutralised, not rejected

**Status:** accepted

**Context.** A restored tree can contain a symlink pointing outside the workspace —
`data/log -> /var/log`, or deliberately `data/config -> /etc`. Mounting it hands a
container a view of the host. Rejecting the whole run instead makes many legitimate
backups undrillable, since stray absolute symlinks are extremely common in real trees.

**Decision.** After restore and before anything is mounted, walk the tree, resolve every
symlink, replace any that escapes the workspace with a zero-byte regular file, and record
each one in `report.warnings` as `symlink_escaped_workspace`.

**Consequences.** The run proceeds and the host is protected. If the application actually
needed that link, a check fails and the warning in the report explains why — which is
more informative than either silently following the link or refusing to start. The
warning list must be surfaced in the TTY report on failure, not only in JSON, or this
becomes a mysterious failure; that is a requirement on the renderer.

## ADR-019: The `test:` section is inert during `restored check`

**Status:** accepted

**Context.** `test.seed` writes data. If a seed step ever ran during a real check, the
tool would be fabricating the evidence it is supposed to be gathering.

**Decision.** `seed` and `export` steps are executed only by `restored recipe test`. The
`test` compose profile, the `${RESTORED_TEST_ASSETS}` mount, and the
`${RESTORED_EXPORT}` mount exist only in the harness. During `check`, they are not
defined, and a compose file referring to them fails validation for the ordinary
"unresolved placeholder" reason.

**Consequences.** A check can never be contaminated by seed data, and the failure mode
if someone gets it wrong is a loud validation error rather than a quiet false PASS. It
means services that exist only for testing must be declared under
`profiles: [test]`, which is one more thing for a recipe author to learn; the
`recipe init` scaffold generates it so most authors never have to work it out.

## ADR-020: A closed `expect` vocabulary, no expression language

**Status:** accepted

**Context.** Checks need to express "at least one row", "status 200", "body contains".
The tempting general solution is an expression language (CEL, starlark, jq).

**Decision.** A fixed table of about twenty `expect` keys (SPEC.md § 3.3), each with one
meaning. No expressions, no scripting, no arithmetic. Regular expressions are Go's RE2,
so they are linear-time and cannot backtrack catastrophically.

**Consequences.** A recipe stays *data*, which is what makes review meaningful, the JSON
Schema complete, and the security story defensible — a reviewer can read a recipe and
know what it will do. Some checks become awkward or impossible to express, and the
answer is to add a vocabulary key with a test rather than to open an escape hatch. The
pressure to add an escape hatch will be constant, and this ADR is the thing to point at.

## ADR-021: Postgres dump format detected from magic bytes

**Status:** accepted

**Context.** `pg_dump` produces plain SQL, custom, directory, or tar format, and users
name the files anything. `backup.sql` being a custom-format dump is common enough to be
the default assumption in a support thread.

**Decision.** Read the first five bytes. `PGDMP` means custom/directory/tar and is loaded
with `pg_restore --clean --if-exists --no-owner --no-acl`; anything else is treated as
plain SQL and loaded with `psql --set ON_ERROR_STOP=1 -f`. The detected format is
recorded in the report and printed in the header.

**Consequences.** Users do not have to declare something the file already knows, and the
detection is visible so a surprise is diagnosable rather than mysterious. `--no-owner
--no-acl` on `pg_restore` deliberately swallows the role-ownership noise that is
irrelevant to a drill; the plain-SQL path cannot do the same, which is exactly why the
`postgres/role-missing` hint exists. `ON_ERROR_STOP=1` means a plain dump fails loudly
on the first error, which is the correct behaviour for a tool whose job is to notice
problems.

## ADR-022: Run ids are 8 random base32 characters, and label everything

**Status:** accepted

**Context.** The compose project name must be unique across concurrent runs, valid as a
compose project name, and short enough to read in a terminal. Orphans from a killed
process must be findable.

**Decision.** 8 lowercase characters from `crypto/rand` over Crockford base32 (no `i`,
`l`, `o`, `u`), giving the project name `restored-<runid>` and the workspace directory
`restored-<runid>`. Every container, network and volume carries the label
`com.restored.run=<runid>`.

**Consequences.** Collision is negligible, the id is short enough to type, and it is not
mistakable for a word. `docker ps -aq --filter label=com.restored.run` finds every
orphan from any run, which makes the "what if it crashes" answer a one-liner rather than
a paragraph. Timestamps were rejected as ids because two runs starting in the same
second is not hypothetical under `--all`.

## ADR-023: A failed ready probe is exit 1, not exit 2

**Status:** accepted — **flagged for human review**

**Context.** The exit-code contract (ADR-010) splits "your backup is broken" (1) from
"your drill is broken" (2). Some stages are ambiguous. If the application never becomes
ready, is that a bad backup or a bad environment? If a Postgres dump fails to load, is
that the dump's fault or the recipe's?

**Decision.** The boundary is drawn at LOAD DUMPS. Everything before it — resolve,
prepare, restore, compose up — is exit 2. Everything from LOAD DUMPS onward — dump
loading, ready probes, checks — is exit **1**, restore unusable.

**Consequences.** An application that will not start from its restored data is reported
as an unusable restore, which is the reading a user wants at 3 a.m.: they do not care
whether the data or the config was missing, only that this backup would not have saved
them. The cost is real false alarms — a recipe pinned to an image tag that upstream
deleted, or a host too slow for the ready budget, will page someone with "RESTORE
UNUSABLE" when nothing is wrong with the backup. The mitigations are the weekly
`recipe-health` run (which catches upstream drift before a user does) and generous
default budgets.

**This is the call in the specification most worth a second opinion.** The alternative —
ready-probe failure as exit 2 — trades false alarms for false silence, and false silence
is the failure mode this entire project exists to eliminate. That is why it was decided
this way, but reasonable people will disagree, and the decision is cheap to reverse
before v0.1 ships and expensive after.

## ADR-024: Pure-Go SQLite driver, CGO_ENABLED=0 everywhere

**Status:** accepted

**Context.** SQLite checks need to query a database file. `mattn/go-sqlite3` is the
standard driver and requires cgo, which breaks the six-target cross-compilation matrix
and static linking (ADR-001).

**Decision.** Use `modernc.org/sqlite`, a pure-Go translation, and build everything with
`CGO_ENABLED=0`.

**Consequences.** Cross-compilation stays trivial and the binaries stay static, which is
the whole distribution story. `modernc.org/sqlite` is slower and larger than the cgo
driver — irrelevant here, since queries are `count(*)` over a restored database, run
once. It occasionally lags upstream SQLite versions; if a recipe ever needs a very new
SQLite feature, the answer is a `sqlite` check executed inside a container with the real
`sqlite3` binary, which the `exec` check kind already allows.

## ADR-025: Bundled recipes are embedded in the binary

**Status:** accepted

**Context.** `restored check --recipe gitea` must work immediately after installing one
file, with no network and no separate recipe download.

**Decision.** `recipes/` is embedded with `go:embed`. `--recipe <arg>` resolves a bundled
name first, then a directory path, then a file path. `docs/hints.yaml` is embedded the
same way. No remote registry in v0.1.

**Consequences.** Zero-setup first run, offline operation, and a recipe set that is
versioned and reviewed with the code that runs it. The cost is that a new recipe reaches
users only at the next release, which makes release cadence part of the contribution
experience — an argument for frequent small releases. It also means the binary grows
with the recipe count; at roughly 3 KB per recipe this is not a constraint for hundreds
of recipes. A registry is deferred to v0.3 and only if the bundled set outgrows the
release cycle, with the trust question answered first.

## ADR-026: Checks never stop early, and are never retried

**Status:** accepted

**Context.** Two independent questions: should a failing check abort the run, and should
a failing check be retried?

**Decision.** All checks run, in declaration order, even after one fails. No check is
ever retried; only ready probes retry.

**Consequences.** The report shows the *shape* of the failure — "the web UI works but
every database check fails" is a completely different diagnosis from "nothing works",
and stopping at the first failure destroys that distinction. Not retrying keeps checks
deterministic and keeps the boundary clean: readiness is a timing question and belongs
to probes, correctness is not and belongs to checks. A flaky check is therefore a recipe
bug, to be fixed by strengthening the ready probe rather than by adding retries — which
is the correct place to fix it.

## ADR-027: At most one hint per run

**Status:** accepted

**Context.** Several hint rules can match a single failure, especially when the same
root cause produces errors in a check, in a body, and in service logs.

**Decision.** Rules are ordered in the file, most specific first, matched against check
errors, then bodies, then the last 200 log lines per service. First match wins. Exactly
one hint, or none, is ever shown.

**Consequences.** The report gives one thing to try, which is what a person under
pressure can act on. A list of five candidate causes is a list of five things to ignore.
The cost is that rule ordering becomes load-bearing: adding a broad rule above a specific
one silently shadows it, so the test suite asserts ordering, and every rule ships with a
near-miss fixture proving it does not over-match.

## ADR-028: `--json` writes to stdout, humans to stderr

**Status:** accepted

**Context.** `restored check --json | jq` must work, and so must seeing progress while
it runs.

**Decision.** With `--json`, the report document is the only thing on stdout; all human
output, progress and logging goes to stderr. Without `--json`, human output goes to
stdout as usual.

**Consequences.** Piping works with no flags to remember, and a cron job can capture the
report and the human log to two different places. It is a small surprise that stderr
carries non-error output under `--json`, which is why it is documented on the flag
itself. `--report FILE` writes the same document regardless, so users who want both a
readable terminal and a machine-readable artifact are not forced to choose.

## ADR-029: The nudge requires a strict-valid recipe and a TTY

**Status:** accepted

**Context.** ADR-008 sets the nudge's conditions. Two failure modes were not covered:
nudging in a non-interactive context, and nudging toward a PR that CI will reject.

**Decision.** Add two conditions. The nudge fires only when stdout or stderr is a TTY —
never under `--json`, never in cron — and only when the recipe passes `recipe validate
--strict`.

**Consequences.** Cron logs stay clean, and nobody is invited to open a PR that fails
within a minute. The second condition means a recipe that works but lacks a description
or a maintainer gets no invitation; that is the right trade, because the alternative
spends a first-time contributor's goodwill on a red CI run, and goodwill is the scarcest
resource this project has.

## ADR-030: Image tags pinned, `:latest` rejected, digests optional

**Status:** accepted

**Context.** A recipe pinned by digest is reproducible and safe from tag mutation. A
recipe pinned by digest is also stale within weeks, and updating it is a chore nobody
volunteers for.

**Decision.** The schema requires a tag and rejects `:latest`. Digest pinning
(`image:tag@sha256:…`) is supported and documented as the hardened option, but is not
required.

**Consequences.** Recipes are reproducible enough to test and loose enough to maintain.
`:latest` is rejected because a recipe that passed yesterday and fails today with no diff
is the worst possible experience for both a contributor and the weekly health run. The
residual risk — a mutated tag — is accepted, mitigated by `--pull never` for users who
want a frozen image cache, and stated in the threat model rather than papered over. This
is the trade that keeps the contribution flow alive, and it is a deliberate security
concession.

## ADR-031: No mocked Docker API

**Status:** accepted

**Context.** Testing Docker interaction against a fake would make the suite fast and
hermetic.

**Decision.** Docker interaction is either tested against a real daemon behind the
`integration` build tag, or not tested. No mock Docker API, no fake compose.

**Consequences.** `go test ./...` stays hermetic and fast for a fresh clone because the
Docker-dependent tests are tagged out, and the tests that do run against Docker are
telling the truth. The cost is that Docker-path coverage only exists where a daemon is
available, so those tests skip with an explanatory message rather than failing on a
contributor's machine without Docker. Mocking the one thing whose real behaviour is the
entire risk surface would buy a green suite and a broken tool.

## ADR-032: Stage A treats startup refusal as a pass

**Status:** accepted

**Context.** Stage A starts the stack with empty inputs and requires a check to fail.
Some applications will not start at all with an empty database — they exit, or they loop
on a migration. The checks then fail for the wrong reason, or the stage times out.

**Decision.** If the ready probes do not succeed within a reduced 90-second budget in
stage A, the stage passes with the explicit result `PASS-BY-STARTUP-REFUSAL`, and the
report says so.

**Consequences.** An application that refuses to boot without its data has demonstrated
exactly the data-sensitivity stage A is testing for, so passing is correct. The result is
named distinctly rather than folded into a plain PASS, so a maintainer reviewing a recipe
can see that the checks themselves were never exercised negatively — which is weaker
evidence, and should look weaker. Without this, a large class of applications could not
have recipes at all.

## ADR-033: Six recipes gate v0.1

**Status:** accepted

**Context.** The recipe format will be wrong in ways only writing recipes reveals, and
freezing `apiVersion: restored/v1` after two recipes guarantees a v2 within a month.

**Decision.** v0.1 does not ship until six bundled recipes pass round trips: Gitea,
Uptime Kuma, Vaultwarden, Paperless-ngx, Miniflux, Nextcloud.

**Consequences.** The format is stressed by Postgres and SQLite, by simple and complex
stacks, and by at least one application (Nextcloud) that is genuinely awkward — which is
the point, because the awkward one finds the format's holes. It delays v0.1, and the six
are chosen to span the shapes rather than to be easy. If one proves impossible inside the
isolation rules, that is important information about the rules and belongs in a new ADR
rather than in a quiet exception.

## ADR-034: No standalone `docs/hints.yaml` file in session 1

**Status:** accepted

**Context.** The brief specifies the initial hint catalog as a section of SPEC.md, and
also constrains session 1 to `git init`, `LICENSE`, `.gitignore`, `.editorconfig` and the
named documents.

**Decision.** The full catalog lives in SPEC.md § 6.2 for now. `docs/hints.yaml` is
created in the implementation session, from that section, verbatim.

**Consequences.** Session 1 stays within its constraint and there is exactly one copy of
the catalog. The implementation session must extract it rather than rewrite it, which is
noted in [PROGRESS.md](PROGRESS.md) § Next steps so the content does not drift.

## ADR-035: The GitHub owner stays a placeholder

**Status:** accepted

**Context.** The Go module path, the nudge URL, the schema `$id`s, and the ghcr image
reference all embed a GitHub owner. The name is not decided (see
[docs/name-check.md](docs/name-check.md)) and the owner depends on it.

**Decision.** Documents use the literal `OWNER`. No `go.mod` is created in session 1.
The rename checklist lives at the end of `docs/name-check.md`.

**Consequences.** Nothing has to be un-decided later, and the substitution is one
`grep -rl OWNER`. The cost is that no code can be written until a human answers, which is
recorded as the single blocking item in PROGRESS.md — and since session 1 writes no code
anyway, it blocks nothing that was going to happen.
