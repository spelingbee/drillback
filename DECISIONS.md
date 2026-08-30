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
| [035](#adr-035-the-github-owner-stays-a-placeholder) | The GitHub owner stays a placeholder | superseded by ADR-036 |
| [036](#adr-036-the-name-is-restored-and-the-owner-is-spelingbee) | The name is `restored` and the owner is `spelingbee` | accepted |
| [037](#adr-037-embedded-assets-live-in-a-root-package) | Embedded assets live in a root package | accepted |
| [038](#adr-038-internalrunner-owns-the-state-machine) | `internal/runner` owns the state machine | accepted |
| [039](#adr-039-the-compose-safety-schema-runs-before-interpolation) | The compose safety schema runs *before* interpolation | accepted |
| [040](#adr-040-sqlite-checks-run-in-process-against-the-workspace-file) | SQLite checks run in-process against the workspace file | accepted |
| [041](#adr-041-services-that-receive-a-dump-start-before-the-rest) | Services that receive a dump start before the rest | accepted |
| [042](#adr-042-the-gitea-recipe-mounts-its-data-input-at-data) | The Gitea recipe mounts its data input at `/data` | accepted |
| [043](#adr-043-the-broken-demo-dumps-the-wrong-database) | The broken demo dumps the wrong database | accepted |
| [044](#adr-044-restored-interpolates-composeyaml-itself) | restored interpolates compose.yaml itself | accepted |
| [045](#adr-045-no-restoredyaml-target-or-all-in-this-build) | No `restored.yaml`, `--target` or `--all` in this build | accepted |
| [046](#adr-046-recipe-test-exists-and-says-it-is-not-implemented) | `recipe test` exists and says it is not implemented | accepted |
| [047](#adr-047-a-jsonpath-subset-in-tree-rather-than-a-dependency) | A JSONPath subset in-tree rather than a dependency | accepted |
| [048](#adr-048-a-checks-expect-got-pair-is-a-hint-subject) | A check's expect/got pair is a hint subject | accepted |
| [049](#adr-049-the-demos-take-their-backup-with-restic-in-a-container) | The demos take their backup with restic in a container | accepted |
| [050](#adr-050-forbidden-compose-keys-are-checked-in-go-as-well-as-in-the-schema) | Forbidden compose keys are checked in Go as well as in the schema | accepted |

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

---

## ADR-036: The name is `restored` and the owner is `spelingbee`

**Status:** accepted. Supersedes [ADR-035](#adr-035-the-github-owner-stays-a-placeholder).

**Context.** ADR-035 left the name and the GitHub owner as the literal `OWNER`, and
CLAUDE.md made deciding them a stop point that needs a human. Session 2's brief is
titled "restored - Session 2: Build the core", refers to `restored check`, `cmd/restored`
and `recipes/gitea` throughout, and says explicitly: do not ask questions, decide,
record, continue. Blocking the whole session on a name the brief itself uses would have
produced nothing.

**Decision.** The module path is `github.com/spelingbee/restored`. The owner is the
GitHub account this machine is authenticated as (`gh api user`), not a guess.
`docs/name-check.md` still recommends `drillback` on discoverability grounds, and that
recommendation stands unanswered.

**Consequences.** Code exists. The rename is still one `grep -rl spelingbee` over
`go.mod`, the two schema `$id`s, `internal/nudge`, and the imports, which is a mechanical
change a human can make in a minute. The cost is that a human has now been presented with
a fait accompli on a decision CLAUDE.md reserved for them, which is recorded here and in
PROGRESS.md rather than buried.

---

## ADR-037: Embedded assets live in a root package

**Status:** accepted

**Context.** `schema/*.json`, `recipes/**` and `docs/hints.yaml` ship inside the binary
(ADR-025, ADR-011). `go:embed` cannot reach outside the directory of the package that
declares it, and all three live above `internal/`.

**Decision.** A package `restored` at the repository root, in `assets.go`, holds the
three `embed.FS` values. `internal/recipe` and `internal/hints` read from it.

**Consequences.** There is exactly one copy of each file: the one a contributor edits,
the one CI validates, and the one the binary carries. The alternative - a build step that
copies them under `internal/` - creates a second copy that drifts the first time somebody
edits the wrong one. The cost is a root package that exists only to hold data.

---

## ADR-038: `internal/runner` owns the state machine

**Status:** accepted

**Context.** SPEC.md section 13 lists the packages, and none of them owns the order of
the eight states, the budgets, or the teardown guarantee. `internal/cli` was the only
candidate, and section 13.1 says cli is only allowed to know about exit codes and
stdout.

**Decision.** `internal/runner` holds the lifecycle. It is the only package that knows
what order things happen in.

**Consequences.** The boundaries in section 13.1 survive: `cli` maps a report to an exit
code and nothing else, `report` stays a pure function, and `check` and `probe` still do
not know what a recipe is. The cost is one package the specification does not mention,
which section 13 should gain when it is next revised.

---

## ADR-039: The compose safety schema runs *before* interpolation

**Status:** accepted

**Context.** SPEC.md section 3.4.1 says `compose-safety.schema.json` validates after
`${...}` interpolation, "because the whole point is to see the real, resolved values".
Its own volume rule contradicts that: the pattern accepts a left-hand side that is a
named volume or a literal `${RESTORED_*}` placeholder, and rejects an absolute host
path. After interpolation every placeholder *is* an absolute host path, so the schema
would reject every valid recipe.

**Decision.** The safety schema validates the file as written, with placeholders intact.
Containment of the resolved paths is a separate Go rule, `CheckResolvedMounts`, which
runs after interpolation and requires every bind source to be inside the run workspace.

**Consequences.** Strictly stronger than what the specification describes: the schema
proves the recipe only asks for mounts restored controls, and the Go rule proves those
mounts resolved where they should have. The two together are checkable by an editor and
by CI's independent validator, which post-interpolation validation never could be.
SPEC.md section 3.4.1 needs correcting.

---

## ADR-040: SQLite checks run in-process against the workspace file

**Status:** accepted

**Context.** A `sql` check with `driver: sqlite` names a `file:`, which is a workspace
path, not a container path. The alternative is to run `sqlite3` inside the application's
container, which requires the application's image to ship a `sqlite3` binary. Uptime
Kuma's does not.

**Decision.** SQLite queries run in the restored process, through `modernc.org/sqlite`,
opening the restored file read-only.

**Consequences.** A recipe for a SQLite application needs no extra service and no
sqlite3 binary anywhere, which is one fewer thing between a contributor and a working
recipe. The database is opened read-only, so a check cannot mutate the thing it is
checking. The dependency was already in the budget (CLAUDE.md), and it is pure Go, so
CGO_ENABLED=0 still holds (ADR-024).

---

## ADR-041: Services that receive a dump start before the rest

**Status:** accepted

**Context.** SPEC.md section 4.1 has one COMPOSE UP that starts everything, followed by
LOAD DUMPS. Against a real Gitea that does not work: compose starts Gitea and PostgreSQL
together, Gitea connects to an empty database and runs its own migrations, and the dump
then fails to load with `relation "IDX_auth_token_expires_unix" already exists`. Every
application with automatic migrations behaves this way, which is most of the target list.

**Decision.** COMPOSE UP starts only the services named by an input's `load.service`.
LOAD DUMPS runs. The remaining services start at the beginning of READY, against a
database that already holds the restore.

**Consequences.** This is what a real restore looks like: put the database back, then
start the application. The stage names in the report are unchanged, and the compose
stage's note says which service went first. The cost is that the READY stage now
includes the application's own startup, which was implicit before and is now where the
time is actually spent.

---

## ADR-042: The Gitea recipe mounts its data input at `/data`

**Status:** accepted

**Context.** SPEC.md section 3.1 mounts the `data` input at `gitea:/data/gitea` and
looks for bare repositories under `/data/gitea/gitea-repositories`. The input's own
description says it is "the host side of /data" on a default docker compose install, and
the official image keeps repositories at `/data/git/repositories`. The example
contradicted itself, and the check would have found nothing.

**Decision.** `mount.into: gitea:/data`, and the `repo-files-on-disk` check looks under
`/data/git/repositories` with the same `*/*.git/HEAD` glob.

**Consequences.** The recipe matches what a Gitea backup actually contains, which is the
only thing that makes the check worth running. `scripts/demo.sh` proves it against a
real instance. SPEC.md section 3.1 is corrected in the same commit.

---

## ADR-043: The broken demo dumps the wrong database

**Status:** accepted

**Context.** The session brief asks `scripts/demo-broken.sh` to take its dump with
`pg_dump --schema=public` and expects the report to show
`relation "repository" does not exist`. Neither half works. Gitea's tables live in the
public schema, so `--schema=public` dumps them; and on PostgreSQL 15 and later that dump
opens with `CREATE SCHEMA public`, which fails against a fresh database before any check
runs. Separately, given ADR-041, Gitea rebuilds its own schema when the dump carries
nothing, so the tables exist and the error can never appear for this application.

**Decision.** The broken demo dumps the wrong database - `pg_dump -d postgres` instead
of `-d gitea`, one character in a cron line - which is the most common real cause of
exactly this class of failure. The observable symptom is tables that are present and
empty, and a new hint rule, `db/tables-empty`, names it.

**Consequences.** The demo produces what the brief asked for in substance: RESTORE
UNUSABLE, exit 1, a visible hint, from a backup that a nightly cron job would have been
producing happily for two years. It does not produce the literal error string, because
that string is unreachable for an application that migrates itself, and saying so is
more useful than engineering a fixture to fake it. `postgres/relation-missing` is still
reachable and is covered by the integration suite, where the fixture stack's PostgreSQL
has no application to rebuild its schema.

---

## ADR-044: restored interpolates compose.yaml itself

**Status:** accepted

**Context.** `${RESTORED_*}` placeholders could be left for `docker compose` to expand
from the environment, or expanded by restored before compose sees the file.

**Decision.** restored expands them, writes the result into the workspace, and runs
`docker compose -f <workspace>/compose.yaml`. A `$$` escape survives as a literal `$`,
and any `$` introduced by a substituted value is re-escaped so compose does not expand
it a second time. The environment is passed through as well, so nothing depends on which
of the two did the work.

**Consequences.** What compose reads is exactly what was validated: the containment
check, the label injection, and `recipe show --compose` all operate on the same bytes
the daemon will act on. An undefined placeholder is an error rather than an empty string,
which is the failure mode where a volume mount silently becomes `/`. The cost is that
the file in the workspace has lost the recipe's comments, being a re-marshalled document.

---

## ADR-045: No `restored.yaml`, `--target` or `--all` in this build

**Status:** accepted

**Context.** SPEC.md section 2.9 specifies a config file with sources, targets and a
precedence chain, and `check` takes `--target` and `--all`. None of it is needed for the
session's definition of done, which is `check --recipe ... --source restic` end to end.

**Decision.** `internal/config` is not written. `--config`, `--target` and `--all` are
not registered as flags.

**Consequences.** The `--help` output does not match SPEC.md section 2, which session
2's original plan in PROGRESS.md called the acceptance criterion for the CLI. Registering
a flag that silently does nothing would have been worse: the flags are absent, so an
invocation using one fails loudly. `internal/config` is the first item in the next
session's list.

---

## ADR-046: `recipe test` exists and says it is not implemented

**Status:** accepted

**Context.** The round-trip harness (SPEC.md section 7) is the load-bearing piece of the
contribution flow and is not built. `restored recipe test` is in the CLI surface.

**Decision.** The command is registered. It prints that it is not implemented in this
build, points at PROGRESS.md, and exits 2.

**Consequences.** `restored recipe --help` still describes the whole surface, and nobody
discovers the gap by having a command silently do nothing or, worse, pass. The
alternative - omitting the command - hides a gap that a contributor will hit the moment
they read CONTRIBUTING. This is a documented hole, not a stub pretending to be finished.

---

## ADR-047: A JSONPath subset in-tree rather than a dependency

**Status:** accepted

**Context.** `json_path` needs to select a value out of a response body. The recipes use
`$.data`, `$.type`. A full JSONPath library brings filters, wildcards and recursive
descent, none of which a closed `expect` vocabulary (ADR-020) should offer.

**Decision.** About sixty lines in `internal/check/jsonpath.go` supporting `$`, dotted
keys, bracketed keys, and array indices. Nothing else parses, and an unsupported
expression is an error naming what it expected.

**Consequences.** The dependency budget stays small, and the vocabulary a reviewer has
to hold in their head does not quietly grow a query language through the back door. If a
recipe ever genuinely needs a filter, that is a conversation, not an upgrade.

---

## ADR-048: A check's expect/got pair is a hint subject

**Status:** accepted

**Context.** SPEC.md section 6.1 matches hints against a failing check's
`observed.error`, its body, its stderr, and the service logs. The most common shape of
an unusable restore produces none of those: the query ran, the application answered, and
the answer was a zero where the recipe wanted a one.

**Decision.** Each failing check also contributes a subject in a stable phrasing -
`expected <expect>, got <got>` - which the catalog may match against.

**Consequences.** `db/tables-empty` can fire, and so can any future rule about a
diagnosis that has no error string. The phrasing is now something the catalog depends on,
which is a coupling; `restore/empty-input` in the shipped catalog already depends on
restored's own wording the same way, so the precedent was set in the specification.

---

## ADR-049: The demos take their backup with restic in a container

**Status:** accepted

**Context.** The demo scripts have to produce a restic repository whose snapshot paths
match the recipe defaults - `/srv/gitea/data`, `/srv/gitea/db.sql`. Running restic on
the host records the host's own paths, which on Windows carry a drive letter and on any
machine carry a temporary directory.

**Decision.** The demos run `restic/restic` in a container with the sample tree mounted
at `/srv`. `restored check` still uses the host's restic to read the repository.

**Consequences.** The snapshot is identical on every host, so the demo needs no
`--input` override and shows the defaults working, which is the thing being
demonstrated. The repository format is portable, so the host's restic reads what the
container's restic wrote. The cost is one more pinned image.

---

## ADR-050: Forbidden compose keys are checked in Go as well as in the schema

**Status:** accepted

**Context.** `compose-safety.schema.json` expresses the forbidden keys as a single
`not: {anyOf: [...]}`. That is correct, and the message a validator produces for it is
`services.app: 'not' failed`, which tells a contributor nothing about which key they
used.

**Decision.** The same list is also checked in Go, first, with a message that names the
service, the key, and why it is not allowed.

**Consequences.** A clearer error message over a cleverer implementation, which is the
tie-breaker CLAUDE.md sets. The cost is one list in two places; they are checked against
each other by the one-case-per-construct table in
`internal/recipe/safety/safety_test.go`, which fails if either drifts.

---

## ADR-051: The harness backs up through a container so the snapshot carries the recipe's paths

**Status:** accepted

**Context.** Stage B ends by running the command a user runs:

```sh
restored check --recipe <dir> --source restic --from <workspace>/repo --snapshot latest
```

No `--input` overrides. For that to find anything, the throwaway snapshot has to record
the paths the recipe declares - `/srv/gitea/data`, `/srv/gitea/db.sql` - and not the
paths the staging tree happens to occupy. `restic backup` has no path-rewriting flag;
it resolves every argument to an absolute path on the machine it runs on, which on this
host means a drive letter and a temporary directory.

**Decision.** The harness runs `restic/restic:0.19.1` in a container, bind-mounting each
staged input at *its own* `default_path` inside the container, and backs those paths up.
The container is on `--network none`, runs as the caller's uid on Linux so the
repository belongs to the caller, and passes the password by environment name only, so
no value ever reaches an argument list or the debug log.

Each input is mounted at its leaf path rather than mounting the staging root at `/`, so
nothing shadows a directory the container itself needs.

**Consequences.** The snapshot is identical on every host, so stage B is genuinely the
user's command rather than a decorated version of it. This is the same trick ADR-049
already uses for the demos, and it reuses the same pinned image. The cost is that
`recipe test` needs docker for the backup as well as for the stack, which it needed
anyway.

---

## ADR-052: `recipe test` exit codes - a recipe that proves nothing is invalid, not failing

**Status:** accepted
**Supersedes:** the exit-code table in SPEC.md 2.7 as first written

**Context.** SPEC.md 2.7 originally mapped both "stage A had no failing check" and
"stage B had a failing check" to exit 1, and reserved exit 2 for tool errors including
"recipe invalid". Session 3's brief asks for exit 2 on stage A. The two do not agree.

**Decision.**

| code | meaning |
|---|---|
| 0 | every recipe passed both stages |
| 1 | stage B failed: the round trip did not restore |
| 2 | tool error, **or** stage A found no data-sensitive check |

A recipe whose checks all pass against an empty stack has not failed a test. It is not
a test. That is the same class of defect as a recipe that does not match the schema,
and the spec already routes "recipe invalid" to exit 2.

**Consequences.** The 1/2 split keeps the meaning it has everywhere else in this tool:
1 is a verdict about data, 2 is "this cannot be run as written". A contributor whose
recipe exits 2 with `recipe has no data-sensitive check` is told to add a check; one
whose recipe exits 1 is told the seed, the export and the check disagree about where
the data lives. SPEC.md 2.7 was corrected to match.

---

## ADR-053: Stage B creates only the inputs the application cannot create for itself

**Status:** accepted

**Context.** SPEC.md 7.3 step 1 says stage B starts from "EMPTY inputs (same as stage
A)". Stage A's empty SQLite input is a zero-length file, which is exactly right there:
it is what a backup that captured nothing looks like, SQLite says "file is not a
database", and the checks fail, which is what stage A needs to see.

It is hopeless as a starting point. Uptime Kuma opens `/app/data/kuma.db`, finds a
zero-length file where a database should be, and crash-loops instead of running its
migrations. Stage B never gets a stack to seed. Measured: the ready probe burned its
full budget, three times.

**Decision.** Stage A keeps the three shapes of SPEC.md 7.2. Stage B creates only

- every `dir` input, as an empty directory, and
- every non-`dir` input that compose bind-mounts, as its empty shape, because a bind
  mount to a path that does not exist makes docker create a directory there;

and creates nothing else. An input declared `within:` another arrives when the
application writes it, and an input nothing mounts is produced by an export step.

**Consequences.** Stage B starts the way a first install starts, which is the only way
an application will initialise itself. The two stages no longer share one
"empty inputs" routine, and the difference is the load-bearing part, so it is stated in
`internal/harness/empty.go` and pinned by
`TestEmptyInputsLeavesTheApplicationSomethingToCreate`. SPEC.md 7.3 was corrected.

---

## ADR-054: Empty inputs are created world-writable

**Status:** accepted

**Context.** An application in a container runs as whatever uid its image chose:
Nextcloud is 33, Gitea and Paperless are 1000, the sqlite3 image is 100. None of them
is the uid running `restored`. On Linux a bind mount carries the host's permissions
straight through, so a 0755 directory owned by the caller is a directory the
application cannot write, and stage B never gets a stack that starts. Windows ignores
the mode, which is why this was invisible on the development machine and would have
been a CI-only failure.

**Decision.** The harness creates empty input directories 0777 and empty input files
0666, with an explicit `chmod` after `MkdirAll` because the process umask would
otherwise turn 0777 into 0755.

To make that safe, `internal/workspace` now creates the workspace root **0700** rather
than 0755. The temporary directory is world-traversable; the workspace is not.

**Consequences.** No other local user can reach the permissive files, because they
cannot get through the directory holding them, and the whole tree is destroyed when
the run ends. The alternative was to guess each image's uid, which is not knowable -
the Nextcloud image's `Config.User` is empty, because apache starts as root and drops
privileges later.

---

## ADR-055: A restored tree is opened up before the application sees it

**Status:** accepted
**Extends:** SPEC.md section 4.3, restore-stage sanitisation

**Context.** A backup records the ownership of the machine it was taken from, and
restic restores it faithfully. That ownership means nothing in a restore drill: the
application is about to start in a fresh container as the uid its own image chose.
Measured: a restored Nextcloud data directory arrives as `drwxrwx--- root root`,
www-data cannot write it, and Nextcloud answers 503 "your data directory is not
writable". Gitea and Paperless have the same shape with different uids.

Reporting that as an unusable restore would be a false alarm, which is precisely the
failure this tool exists to remove. It is a fact about uid mapping, not about the
backup.

**Decision.** After sanitisation, `restored` opens the modes of every restored input:
directories 0777, files 0666. Ownership is left alone, because the uid to chown *to* is
not knowable from outside the container.

**Consequences.** Most applications now start against a restored tree without the
recipe having to do anything. An application that refuses a permissive mode - Nextcloud
rejects a world-readable data directory outright - declares a preparation service in
its own compose file, which is `recipes/nextcloud`'s `prepare`. That is the same
`chown -R www-data:www-data` its documentation asks an operator for, so the recipe is
teaching the real procedure rather than working around the tool.

Safe for the same reason as ADR-054: the workspace root is 0700 and lives for one run.

## ADR-056: Interpolation may change a value, and nothing else

**Status:** accepted
**Extends:** ADR-039, SPEC.md section 9.3
**Found by:** the session 4 security review, `docs/review/security.md` SEC-01

**Context.** ADR-039 validates `compose.yaml` as written, with the `${RESTORED_*}`
placeholders intact, because the schema's volume rule can only recognise a bind mount
restored controls while the placeholder is still there. The consequence nobody wrote
down is that the only structural check on the document happens *before* the values go
in. Nothing re-read the file afterwards: `CheckResolvedMounts` parses
`services.*.volumes` and nothing else, and `LabelCompose` round-trips the YAML.

`vars` values are contributor-supplied strings, and a string containing a line break
pasted into an unquoted scalar position does not fill in a value - it adds lines to
the document. Proven against this repository: a recipe with

```yaml
vars:
  port: "8080\n    privileged: true\n    network_mode: host\n    pid: host"
```

passed `restored recipe validate --strict` with exit 0, and `recipe show --compose`
printed a service running privileged, on the host network, in the host PID namespace.
Every bundled recipe already models the vulnerable shape, because
`POSTGRES_PASSWORD: ${RESTORED_VAR_db_password}` is unquoted in `recipes/gitea`. The
blast radius is root on the machine running the drill, and root on the GitHub-hosted
runner for every fork pull request `recipes.yml` tests.

**Decision.** Interpolation is a value substitution and is now checked as one. A new
`safety.Render` replaces every direct call to `safety.Interpolate`, and it asserts
that the rendered document has the same *shape* as the document it came from: the
same mapping keys at every path, the same sequence lengths, and a scalar wherever
there was a scalar. Scalar values are free to differ; that is the point. Anything
else differing means a substituted value was parsed as YAML structure.

Two cheaper checks sit in front of it, for the message rather than for the coverage:
`Interpolate` rejects a substituted value containing a line break and names the
variable, and `Render` re-runs `checkForbiddenKeys` so a smuggled key is named.

`safety.Validate` now renders the file and throws the result away, so that
`recipe validate` - which is what CI gates a contributed recipe on and what a
maintainer reads before merging - refuses the injection rather than discovering it
minutes into a run.

**Consequences.** The check is on the class, not the instance. A deny-list of
injectable keys would have to grow with the compose specification; "interpolation adds
no keys" does not. A recipe variable can no longer carry a multi-line value - a
certificate, a private key, a config fragment - and must use an input instead, which
is the correct place for anything that large and is what the error says.

The cost is one extra YAML parse per render, on a path that is about to start
containers.

---

## ADR-057: The compose safety schema is an allow-list

**Status:** accepted
**Supersedes the deny-list half of:** ADR-009, ADR-014
**Found by:** the session 4 security review (`docs/review/security.md` SEC-02, SEC-04)
and the maintainer review (`docs/review/maintainer.md` MNT-01)

**Context.** `schema/compose-safety.schema.json` enumerated twelve keys to reject and
accepted everything else, and it constrained only `services.*` and `networks.*`. Three
holes followed from that shape, all confirmed accepted by
`recipe validate --strict` with exit 0 against this repository:

- The top-level `volumes:` block was not validated at all. A "named volume" with
  `driver_opts: {type: none, device: /, o: bind}` is a bind mount of any host path,
  and `CheckResolvedMounts` skipped it because `hostroot` is not a host path.
  `device: /var/run` hands a container the Docker socket, which SPEC.md section 9.3
  says restored never does. `external: true` with a `name:` attaches a volume
  belonging to one of the user's other containers.
- The service body accepted every key not on the deny-list.
  `volumes_from: ["container:<name>"]` attaches the volumes of a container already
  running on the host. `extra_hosts` with `host-gateway`, `uts: host`, `sysctls`,
  `group_add`, `security_opt: label:disable` and `env_file: ../../../etc/passwd` were
  all accepted.
- Top-level `configs:` and `secrets:` both take a `file:` that the daemon reads from
  the host.

The pattern is the same each time: the compose specification grows, and a deny-list
loses to it silently. Every new key is an unreviewed grant that nobody decided to
make.

**Decision.** Invert it. `additionalProperties: false` at the document root, on the
service body, on the network body and on the volume body, with an enumerated
allow-list of what a recipe legitimately needs. The root allows `services`,
`networks`, `volumes` and `name`. A top-level volume may carry `labels` and `name`
and nothing else - not `driver`, not `driver_opts`, not `external`. The long-syntax
service volume is closed the same way.

`forbiddenService` and the new `forbiddenTopLevel` stay, and are now purely about
message quality: the schema rejects `privileged` with a JSON Schema error, and the Go
rule rejects it first with a sentence explaining why. Both `env_file` and `sysctls`
are deliberately outside the allow-list rather than inside it with a constraint.

**Consequences.** A recipe that needs a compose key restored has not thought about now
fails with "unknown key", and the contributor opens an issue instead of silently
getting the grant. That is the trade this project wants: a maintainer merges a recipe
without reading it, so the schema has to be the thing that is trusted, and a schema
that accepts what it has not considered cannot be.

All five bundled recipes validate unchanged under the allow-list - they use seven
service keys between them - which is evidence that the list is not too tight, not
proof. Widening it is a one-line pull request with a reason attached, which is the
review this class of change should get.

`internal/recipe/safety/bypass_test.go` holds one test per bypass above, so a
regression names the finding it re-opens.

## ADR-058: A run that ran out of time has no opinion about the backup

**Status:** accepted
**Extends:** SPEC.md sections 4.2 and 5.1
**Found by:** the session 4 architecture review (`docs/review/architecture.md` ARCH-01
and ARCH-04) and the UX review (`docs/review/ux.md` UX-01)

**Context.** Two findings, one root cause: the runner decided what a failure *meant*
from *where* it happened rather than from *why*.

Every stage from LOAD DUMPS onward routed its error through `fail()`, which set
`RESTORE_UNUSABLE` and exit 1 unconditionally. A cancelled `runCtx` is
indistinguishable from a real failure at that point - the probe records `context
deadline exceeded`, the stage is marked failed, and the report accuses the backup. The
`restored --help` text said the opposite, promising exit 2 for a "timeout before any
check could run".

The defaults made it reachable rather than theoretical: `--timeout 15m` with
`--restore-timeout 10m` and `--ready-timeout 5m` inside it left nothing for COMPOSE
UP, LOAD DUMPS and the checks. A 40 GB restore that took nine minutes - inside its own
budget, so it succeeded - ran the whole run out of time during the ready probes, and a
cron job paged somebody at 03:00 about a backup that was fine.

The mirror image of the same mistake: `attachHint` was called only from the two exit-1
paths, so every catalog rule about a *tool* error was unreachable code. Four of the
seventeen shipped rules could never fire, including the one for an unreachable docker
daemon and the one for a path that is not in the snapshot. `cli/check.go` then
suppressed the TTY report whenever the run returned an error, while `--report` still
wrote the JSON - so a machine consumer saw more than the human did. ADR-011 sells the
hint catalog as "the easiest useful contribution to restored", and a quarter of the
examples a contributor would read from were dead.

**Decision.** Three changes.

1. `fail()` asks `runCtx.Err()` before it sets a verdict. A deadline or a cancellation
   produces `ERROR` and exit 2 with a message naming the stage and the budget, and
   saying plainly that nothing is known about the backup because the drill did not
   finish. Only a real stage failure still produces `RESTORE_UNUSABLE`.

2. The default `--timeout` is 30 minutes, and the stage budgets are clamped to fit
   inside whatever it is: restore to half, ready to a quarter, a check to an eighth.
   Clamping is downward only, so an explicit `--restore-timeout 30s` is left where the
   user put it. SPEC.md 4.2's per-state budgets sum to roughly 27 minutes, which is
   what 30 is drawn from.

3. `attachHint` moves into a deferred finaliser registered before anything can fail,
   so every exit path gets a hint match - including the ones that never reach a
   verdict. It tolerates a nil resolution, because the earliest failures happen before
   the recipe is resolved and those are the ones the catalog has most to say about.
   `cli/check.go` renders the report whenever the run got far enough to have an id,
   and stops repeating the error underneath it.

**Consequences.** Exit 1 now means what the help says it means: the drill finished and
the restore was unusable. That is the number people put in cron jobs and alerting
rules, and it was wrong in the one case where a false alarm is most expensive.

A run can now take twice as long before it gives up, which is the right trade for a
tool whose slowest supported operation is restoring somebody's entire Nextcloud.

Four hint rules that were shipped, tested and unreachable now fire. The most common
first failure - a recipe default path that is not the user's layout - reaches the user
as a report with the stages, the workspace, a message naming `--input`, and a hint
carrying a runnable `restic ls latest` command. It used to be one line of prose with
nothing to do next.

## ADR-059: A repository string is scrubbed before it is shown to anyone

**Status:** accepted
**Implements:** SPEC.md section 9.3, which has promised this since session 1
**Found by:** the session 4 security review (`docs/review/security.md` SEC-03)

**Context.** `repositoryLabel` returned `--from` unchanged, or `RESTIC_REPOSITORY`,
and the result became `report.source.repository`: printed by the TTY report, written
by `--json` and `--report`, and echoed into the debug log as part of the restic argv.

restic's REST backend takes credentials inside the repository string -
`rest:https://user:password@host:8000/` is documented upstream and is a common
configuration - and so do S3, Azure, B2, Swift and rclone. SPEC.md section 9.3 said
the URL "is scrubbed of any `user:password@` userinfo". Nothing scrubbed anything:
`grep -rn -i 'scrub|userinfo|redact' --include='*.go' .` returned nothing.

The issue templates ask a reporter to attach the JSON report. SECURITY.md names this
exact case as in scope.

**Decision.** One function, `restic.SafeRepository`, used by `repositoryLabel` and by
the debug line that prints the restic argv. It parses the part after the backend
prefix as a URL and, when there is a password, keeps the user name and drops the
password. The user name stays because it is often the only way to tell two
repositories apart in a report, and it is not the secret.

Backends whose second half is not a URL are left alone by name, which is what keeps a
local path - and a Windows drive letter, whose colon looks exactly like a backend
prefix - untouched.

**Consequences.** The report can still be attached to a bug report, which is what it
is for. `sftp:user@host:/path` is unchanged, because it carries no password.

This does not make the report safe to publish in general: SEC-08 (200 lines of every
service's container log) and SEC-06 (`expect.glob` as a filesystem oracle) are
separate findings with their own entries in the issue tracker. It closes the one that
was a promise in the specification.

## ADR-060: The recipe tables are regenerated on main, not by the contributor

**Status:** accepted
**Extends:** ADR-030 (generated files are checked in and CI diffs them)
**Found by:** the session 4 maintainer review (`docs/review/maintainer.md` MNT-02)

**Context.** `recipes/README.md` and the table in `README.md` are generated from the
recipe registry, are checked in, and were diffed by CI's `generated` job. Adding
`recipes/<name>/` changes both. So `ci / generated` went red on *every* recipe pull
request, while CONTRIBUTING.md promised, in bold, that a recipe-only pull request
needs nothing but `recipes.yml` to be green. The walkthrough and the pull request
template never mention regenerating anything, because when they were written nobody
had walked the path with a new directory in it.

The contributor this project measures itself by - a stranger adding one recipe - was
therefore guaranteed a red check on their first attempt, with the fix being a
generator command nothing had told them about. That is the most expensive possible
place to put an unwritten step.

**Decision.** Split the generated files by where they come from.

`docs/recipe-spec.md` is generated from `schema/recipe.schema.json`. It still hard
fails in CI, for everyone: changing the schema without regenerating the document
contributors read is a real mistake by whoever changed the schema.

`recipes/README.md` and the `README.md` table are generated from the registry, which
changes because someone added a recipe. CI reports them as a notice and does not gate
on them, and a new workflow, `refresh-registry.yml`, regenerates and commits them on
`main` after the merge.

**Consequences.** The promise in CONTRIBUTING.md is now true, which was the point. A
recipe contributor adds one directory and is done.

The tables can be stale for the length of one merge, on `main` only. That is
acceptable: they are a convenience index, and `restored recipe validate ./recipes/*`
and the registry itself are the sources of truth.

The loop guard on `refresh-registry.yml` is load-bearing and doubled: the trigger
excludes `recipes/README.md`, and the push uses `GITHUB_TOKEN`, which by GitHub's own
rule does not trigger further workflow runs. Remove either and the job re-triggers
itself forever.

## ADR-061: A failed round trip carries the check that failed it

**Status:** accepted
**Extends:** SPEC.md section 7, the round-trip harness
**Found by:** the session 4 maintainer review (`docs/review/maintainer.md` MNT-03) and
the fresh-clone walk (`docs/review/fresh-clone.md` FC-05)

**Context.** `harness.Stage` had `Reason` and `Error` and nothing else. Stage B ran a
real `restored check`, read two counters off the report, and let it go out of scope.

So a contributor whose recipe failed in CI got one sentence - "2 of 5 checks failed
after a real round trip (repos-in-db, api-lists-repos)" - and nothing else. Not the
query, not the expectation, not what came back, not what the application logged. All
of it had been computed. The `--report` JSON that `recipes.yml` uploads as an artifact
had no check array in it at all. The workspace holding the debug log was deleted,
because CI does not pass `--keep`.

The one actionable line that was printed made it worse: `st.Command` named the restic
repository inside the harness workspace that the same function had just removed, so
pasting the reproduction command answered "no such file or directory".

The fresh-clone reviewer hit exactly this from the other side and lost 6.5 of their
13.4 recipe-writing minutes to two blind three-minute runs.

This is the review burden the harness exists to remove, reappearing on every failing
pull request: either the contributor reproduces it locally with Docker, or the
maintainer does.

**Decision.** `harness.Stage` gains `Check *report.Report`, populated whenever a stage
did not pass - a failed round trip, a stage A that found no data-sensitive check, and
a tool error, which since ADR-058 also produces a report worth keeping. It is omitted
for a passing stage, because nobody reads a passing check's observations and this
document is uploaded as a CI artifact.

`writeResult` renders it with `report.Report.WriteTTY`, indented two spaces under the
stage that produced it. That renderer already exists, is pure, and is golden-tested;
the harness grows no second renderer.

Both `st.Command` values are replaced with commands that still work after teardown:
`restored recipe test <ref> --stage a --keep` and `--stage b --keep`.

**Consequences.** A failing recipe pull request now shows the contributor what the
check asked, what it expected, what it got, what the services logged and which hint
matched - in the CI log, without downloading anything, and in the JSON artifact for
anyone who wants to parse it.

The harness JSON is larger, because it now nests up to two check reports per recipe.
That is the point of it. The 200-line-per-service log embedding that makes it large is
its own finding (SEC-08) with its own entry in the backlog.
