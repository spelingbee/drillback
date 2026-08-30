# drillback — Specification v0.1

> **Your backup is a lie until it boots.**

Status: **partly implemented.** This document is the contract for v0.1. Any change to
it must be accompanied by an entry in [DECISIONS.md](DECISIONS.md), and session 2
changed five things in it where reality disagreed: see ADR-039, ADR-041, ADR-042,
ADR-043 and the `PGOPTIONS` note in section 3.1. [PROGRESS.md](PROGRESS.md) records what
is built and what is not.

Working name: `drillback`, owner `spelingbee` (ADR-036). See
[docs/name-check.md](docs/name-check.md) — the name is still not final, and the rename
is still a single grep.

---

## Table of contents

1. [Problem, audience, non-goals](#1-problem-audience-non-goals)
2. [CLI surface](#2-cli-surface)
3. [Recipe format](#3-recipe-format)
4. [Run lifecycle](#4-run-lifecycle)
5. [Report format](#5-report-format)
6. [Hints](#6-hints)
7. [Round-trip harness](#7-round-trip-harness)
8. [Contribution nudge](#8-contribution-nudge)
9. [Threat model](#9-threat-model)
10. [Testing pyramid](#10-testing-pyramid)
11. [CI plan](#11-ci-plan)
12. [Release process](#12-release-process)
13. [Repository layout](#13-repository-layout)
14. [Roadmap](#14-roadmap)

---

## 1. Problem, audience, non-goals

### 1.1 The problem

A self-hoster sets up restic, points it at `/srv`, adds a cron entry, and watches it
turn green every night for two years. The backup is running. Nobody has ever restored
it. Then the disk dies, and they discover one of the following, for the first time,
under pressure:

- The Postgres dump was taken with `pg_dump --schema-only`, or from the wrong database,
  or by a user without permission on half the tables, and the cron job never checked
  the exit code.
- The SQLite file was copied while the app was running, so it is a torn page or a
  `-wal` file was left behind and the `.db` alone is stale.
- The app directory was backed up, but the bind-mounted named volume that actually
  held the data was not — the backup contains an empty tree with correct permissions.
- Everything restores, but the app will not start, because the config references a
  secret, a UID, or a path that only existed on the old host.

Every one of these is silent until the restore. A green backup job is not evidence.
The only evidence is a restore that boots.

Doing that by hand is expensive: you need a spare machine or at least a safe corner of
the current one, you need to remember how the app was wired, you need to not clobber
the running instance, and you need to know what "it worked" even looks like for that
app. So nobody does it. Not monthly, not ever.

### 1.2 What `drillback` does

`drillback` is a single-binary CLI. Given a backup repository and a *recipe*, it:

1. picks a snapshot (default: the latest),
2. restores the paths the recipe needs into a fresh temporary workspace,
3. starts the application with `docker compose` in an isolated, unpublished,
   internal-network compose project,
4. loads any database dumps declared by the recipe,
5. waits for the recipe's *ready probes*,
6. runs the recipe's *checks* — "the app answers", "the rows are there", "the files are
   there",
7. prints `PASS` or `RESTORE UNUSABLE` with the evidence and, on failure, a hint about
   the likely cause,
8. destroys everything it created.

It answers exactly one question — *would this backup actually come back?* — and it is
designed so the answer costs one command and about a minute.

### 1.3 Who it is for

Self-hosters and small-team sysadmins running applications with `docker compose`:
Nextcloud, Gitea, Vaultwarden, Immich, Paperless-ngx, Home Assistant, Miniflux,
Jellyfin, Forgejo, Uptime Kuma. People who already have backups and already have
Docker, and who are one cron line away from having evidence instead of hope.

Explicitly *not* the target for v0.1: Kubernetes operators, enterprise backup suites,
bare-metal or VM-image restore, and anyone whose application is not expressible as a
compose stack.

### 1.4 Success metric

Not stars. **The number of distinct external contributors with merged PRs.**

The unit of contribution is a recipe: roughly sixty lines of YAML that somebody who
runs Paperless-ngx already knows how to write. Every design choice in this document
exists to make that contribution *cheap to create* (`recipe init` scaffolds it),
*automatic to validate* (`recipe test` proves it locally and in CI, with no maintainer
in the loop), and *a short path to submit* (the contribution nudge, section 8).

### 1.5 Non-goals for v0.1

These are deliberate omissions, not oversights. Each is a "no" for v0.1 specifically.

| Not in v0.1 | Why | When |
|---|---|---|
| borg, kopia, rclone, tar archives as sources | One source done properly beats four done shallowly. restic covers the largest share of the target audience and has a clean CLI contract. | v0.2 (borg, kopia) |
| MySQL / MariaDB dumps | `mysql-dump` needs its own format detection, charset handling, and `--single-transaction` caveats. Postgres and SQLite cover the two most common self-hosted shapes. | v0.2 |
| Notifications (ntfy, Gotify, healthchecks.io, email, webhooks) | Exit codes and `--json` are already enough to wire any notifier in one cron line. Building notifiers in v0.1 spends effort on plumbing instead of on recipes. | v0.2 |
| Web UI, dashboard, HTML report | The output is a terminal and a JSON document. Anything else is a second product. | HTML report in v0.3 |
| Built-in scheduling / daemon mode | `drillback` is a one-shot command. The system already has cron and systemd timers, and they are better at it. | cron mode with history in v0.2 |
| A remote recipe registry | Recipes ship embedded in the binary from `recipes/` in this repository. A registry is an availability and trust problem we have not earned yet. | undecided |
| Restoring *to* production | `drillback` never writes outside its workspace and has no "and now put it back" mode. It verifies; a human restores. | never |
| Non-Docker runtimes (Podman without the Docker socket, LXC, systemd-nspawn) | One runtime contract. Podman's docker-compatible socket happens to work today, but it is not tested or supported. | undecided |
| Incremental / partial verification, deduplication awareness, backup *performance* measurement | Different question. | never |

---

## 2. CLI surface

The blocks in this section are normative for the *surface*: which flags exist, their
defaults, the exit codes, and the documented sections of each help text. They are not
byte-exact renderings - cobra owns the parts a hand-written block cannot promise
(alphabetical flag order, line wrapping, canonical duration forms like `1m0s`, the
generated `completion` command, and the `Global Flags` section) - and chasing those
bytes would mean hand-maintaining a mock of a generator's output, which is the exact
mistake section 5.1's warning exists for. A flag present here and absent in the build,
a differing default, or a missing documented section (see UX-11 in
`docs/review/backlog.md` for the known one) is a bug in one of the two. ADR-069
records the adjudication.

### 2.1 `drillback --help`

```text
drillback — your backup is a lie until it boots.

Restores the latest snapshot of a backup into a throwaway, isolated environment,
starts the application with docker compose, and verifies that it actually works.

Usage:
  drillback [command]

Available Commands:
  check       Restore a backup and verify that the application boots
  recipe      Work with recipes: validate, show, init, test
  version     Print version, commit, and build information
  help        Help about any command

Flags:
      --config string      Path to drillback.yaml. Default search order:
                           ./drillback.yaml, $XDG_CONFIG_HOME/drillback/drillback.yaml,
                           /etc/drillback/drillback.yaml
      --json               Emit the machine-readable report on stdout; human output
                           goes to stderr, so `drillback check --json | jq` works
      --log-level string   trace|debug|info|warn|error (default "info")
      --no-color           Disable ANSI colour (NO_COLOR is also honoured)
      --no-nudge           Never print the "contribute this recipe" invitation
  -h, --help               help for drillback
  -v, --version            version for drillback

Exit codes:
  0    all checks passed
  1    restore unusable - the drill finished and one or more checks failed, or the
       application never became ready
  2    tool or runtime error - docker missing, restic failed, recipe invalid, or the
       run exceeded --timeout before it could reach a verdict
  130  interrupted - the workspace and the compose project may still exist

Docs: https://github.com/spelingbee/drillback
```

> The owner is `spelingbee` (ADR-036). A human has not confirmed the name; the rename
> is one grep and it is listed under *Open questions* in [PROGRESS.md](PROGRESS.md).

### 2.2 `drillback check --help`

```text
Restore a backup into a throwaway environment and verify that the application boots.

Every run gets its own workspace directory, its own compose project named
drillback-<runid>, and its own internal network. No ports are published; HTTP checks
run from a helper container attached to the run's network. The workspace and the
compose project are always destroyed on exit — including on Ctrl-C and on panic —
unless --keep or --keep-on-fail says otherwise.

Usage:
  drillback check [flags]

Examples:
  # restic repository from the environment, bundled recipe, latest snapshot
  export RESTIC_REPOSITORY=/mnt/backups/restic
  export RESTIC_PASSWORD_FILE=/etc/restic/pass
  drillback check --recipe gitea

  # a specific snapshot, with one input remapped to a non-default path
  drillback check --recipe gitea --snapshot 4a7f1c2e \
      --input db=/var/backups/gitea/gitea.dump

  # a local recipe directory, against a tree that is already restored on disk
  drillback check --recipe ./recipes/uptime-kuma --source dir --from /mnt/export/uk

  # only snapshots this host wrote, tagged "gitea"
  drillback check --recipe gitea --host hypervisor --tag gitea

  # every target in drillback.yaml, for cron
  drillback check --all --json --report /var/log/drillback/run.json

Flags:
      --recipe string           Recipe to run: a bundled name (e.g. "gitea"), a path
                                to a directory containing recipe.yaml, or a path to a
                                recipe.yaml file. Required unless --target or --all.
      --target string           Run one named target from drillback.yaml
      --all                     Run every target in drillback.yaml, sequentially
      --source string           Backup source: restic|dir (default "restic")
      --from string             Source location. For restic: the repository, overriding
                                RESTIC_REPOSITORY. For dir: the already-restored tree.
      --snapshot string         restic snapshot id, or "latest" (default "latest")
      --tag strings             Only consider snapshots carrying this tag (repeatable)
      --host string             Only consider snapshots written by this host
      --input name=path         Override a recipe input's path inside the backup
                                (repeatable). See `drillback recipe show <name>` for
                                the input names a recipe declares.
      --set key=value           Override a recipe variable (repeatable)
      --timeout duration        Wall-clock budget for the whole run (default 30m)
      --restore-timeout duration  Budget for the restore stage (default 10m)
      --ready-timeout duration  Budget for all ready probes together (default 5m)
      --check-timeout duration  Per-check timeout (default 60s)
      --pull string             Image pull policy: always|missing|never (default "missing")
      --workspace string        Parent directory for the run workspace
                                (default: os.TempDir())
      --keep                    Do not tear down. Prints the workspace path and the
                                compose project name, and the exact commands to clean
                                up by hand.
      --keep-on-fail            Tear down on PASS, keep everything on failure
      --report string           Also write the JSON report to this file
      --hints string            Load additional hint rules, matched before the
                                built-in ones (section 6.1)
      --no-nudge                Never print the "contribute this recipe" invitation
  -h, --help                    help for check

Environment:
  RESTIC_REPOSITORY, RESTIC_PASSWORD, RESTIC_PASSWORD_FILE, RESTIC_PASSWORD_COMMAND
  and the backend variables restic itself reads (AWS_*, B2_*, AZURE_*, ...) are passed
  through to restic unchanged. drillback never parses or logs their values.
```

### 2.3 `drillback recipe --help`

```text
Work with recipes.

A recipe is a directory containing recipe.yaml, compose.yaml, and optionally
test assets. It declares the *logical inputs* an application needs — not your paths —
plus the probes that say the app is up and the checks that say the data survived.

Usage:
  drillback recipe [command]

Available Commands:
  init        Scaffold a new recipe directory
  show        Print a resolved recipe: defaults applied, variables expanded
  test        Run the round-trip harness against a recipe
  validate    Validate a recipe against the schema and the safety rules

Flags:
  -h, --help   help for recipe

Use "drillback recipe [command] --help" for more information about a command.
```

### 2.4 `drillback recipe validate --help`

```text
Validate one or more recipes against the JSON Schema and the safety rules.

The safety rules are hard failures, never warnings. A recipe is rejected if its
compose.yaml uses privileged containers, host networking, the host PID or IPC
namespace, published ports, a non-internal network, a bind mount to a path outside
the run workspace, or a `!!` YAML tag. See SPEC.md section 9, Threat model.

Usage:
  drillback recipe validate <dir|file>... [flags]

Examples:
  drillback recipe validate ./recipes/gitea
  drillback recipe validate ./recipes/*/
  drillback recipe validate ./recipes/gitea --json
  drillback recipe validate ./recipes/*/ --strict     # what CI runs

Flags:
      --strict   Also fail on warnings: missing description, missing maintainer,
                 an image reference without a tag, a check without a title, a recipe
                 with fewer than two checks.
      --json     Machine-readable findings on stdout
  -h, --help     help for validate

Exit codes:
  0  every recipe is valid
  2  at least one recipe is invalid  (recipe problems are tool errors, not verdicts)
```

### 2.5 `drillback recipe show --help`

```text
Print a recipe with defaults applied, variables expanded, and inputs resolved to the
paths this invocation would actually use.

Use this to see exactly what `drillback check` would do, without running anything.

Usage:
  drillback recipe show <name|dir|file> [flags]

Examples:
  drillback recipe show gitea
  drillback recipe show gitea --input db=/backup/gitea.dump --format json
  drillback recipe show ./recipes/uptime-kuma --compose

Flags:
      --format string     yaml|json (default "yaml")
      --input name=path   Override an input path (repeatable)
      --set key=value     Override a variable (repeatable)
      --compose           Also print the rendered compose.yaml
      --inputs-only       Print only the resolved input table — the fastest way to
                          answer "which paths does this recipe want from my backup?"
  -h, --help              help for show
```

### 2.6 `drillback recipe init --help`

```text
Scaffold a new recipe directory: recipe.yaml, compose.yaml, and README.md, prefilled
with comments and a working skeleton.

The generated recipe deliberately does NOT pass `drillback recipe test` yet. It has no
data-sensitive check, and stage A will tell you so. Writing that check is the first
and most important thing you do, because it is the only thing that makes the recipe
worth anything.

Usage:
  drillback recipe init <name> [flags]

Examples:
  drillback recipe init paperless
  drillback recipe init immich --db postgres-dump --with-dir data --with-dir thumbs
  drillback recipe init miniflux --dir ./my-recipes --image miniflux/miniflux:2.2.0

Flags:
      --dir string         Parent directory (default "./recipes")
      --db string          Database input to scaffold: none|sqlite|postgres-dump
                           (default "none")
      --with-dir strings   Scaffold a dir input with this name (repeatable,
                           default [data])
      --image string       Application image to write into compose.yaml
      --force              Overwrite an existing directory
  -h, --help               help for init
```

### 2.7 `drillback recipe test --help`

```text
Run the round-trip harness against a recipe. This is what CI runs for every PR that
touches recipes/**, and it runs identically on your laptop.

Stage A, "negative": start the stack with EMPTY inputs, run the checks, and require
that AT LEAST ONE check FAILS. A recipe whose checks all pass against an empty stack
proves nothing about a restore and is rejected with "recipe has no data-sensitive
check".

Stage B, "positive": start a fresh stack, run test.seed, run test.export, back the
resulting input tree up into a throwaway restic repository, tear everything down, then
run a normal `drillback check` against that repository and require that ALL checks PASS.

Usage:
  drillback recipe test <name|dir>... [flags]

Examples:
  drillback recipe test ./recipes/gitea
  drillback recipe test ./recipes/gitea --stage a --keep
  drillback recipe test ./recipes/* --json --report ./recipe-test.json

Flags:
      --stage string      a|b|both (default "both")
      --timeout duration  Wall-clock budget per recipe for the whole harness
                          (default 20m)
      --keep              Keep workspaces and compose projects for inspection
      --report string     Write the JSON report to this file
      --json              Machine-readable report on stdout
  -h, --help              help for test

Exit codes:
  0  every recipe passed both stages
  1  stage B failed: the round trip did not restore
  2  tool error (docker missing, restic missing, recipe invalid, timeout), or stage A
     found no data-sensitive check, which makes the recipe invalid rather than failing
```

The 1/2 split is the one the rest of the tool uses: 1 is a verdict about data, 2 is
"this cannot be run as written". A recipe whose checks all pass against an empty stack
has not failed a test - it is not a test. See DECISIONS.md ADR-052.

### 2.8 `drillback version --help` and its output

```text
Usage:
  drillback version [flags]

Flags:
      --json   Machine-readable output
  -h, --help   help for version
```

```text
$ drillback version
drillback 0.1.0
  commit:    3f9a1c4e8b2d7605a1f39c0e5d84b7a2c6e1f093
  built:     2026-09-14T08:41:22Z
  go:        go1.25.1
  platform:  linux/amd64
  docker:    27.3.1 (compose v2.29.7)
  restic:    0.17.3
  recipes:   14 bundled
```

The `docker`, `compose` and `restic` lines are probed at runtime and print
`not found` when the dependency is missing. `drillback version` never exits non-zero
for a missing dependency — that is `check`'s job — so it stays usable as a bug-report
command.

### 2.9 `drillback.yaml`

Discovered at `./drillback.yaml`, then `$XDG_CONFIG_HOME/drillback/drillback.yaml`, then
`/etc/drillback/drillback.yaml`. First match wins; `--config` overrides the search.

```yaml
# /etc/drillback/drillback.yaml
version: 1

defaults:
  source: home-nas          # which entry of `sources` a target uses if it says nothing
  timeout: 15m
  restore_timeout: 10m
  ready_timeout: 5m
  check_timeout: 60s
  pull: missing             # always|missing|never
  nudge: true               # false is the config equivalent of --no-nudge
  workspace: /var/tmp       # parent dir for run workspaces; needs room for the data

sources:
  home-nas:
    kind: restic
    repository: sftp:backup@nas.lan:/srv/restic
    password_file: /etc/drillback/nas.pass
    host: hypervisor        # default snapshot filter for every target on this source

  offsite:
    kind: restic
    repository: s3:s3.eu-central-1.amazonaws.com/acme-backups
    password_command: ["pass", "show", "restic/offsite"]
    env:
      # Passed to restic verbatim. Values are read from the environment of the
      # drillback process; drillback does not read secrets out of this file.
      AWS_ACCESS_KEY_ID: ${AWS_ACCESS_KEY_ID}
      AWS_SECRET_ACCESS_KEY: ${AWS_SECRET_ACCESS_KEY}

  staging-export:
    kind: dir
    path: /mnt/exports/nightly

targets:
  gitea:
    recipe: gitea
    source: home-nas
    tags: [gitea]
    inputs:
      data: /srv/gitea/data
      db:   /srv/gitea/dumps/gitea.sql

  vaultwarden:
    recipe: vaultwarden
    source: home-nas
    tags: [vaultwarden]
    inputs:
      data: /srv/vaultwarden

  uptime-kuma:
    recipe: uptime-kuma
    source: offsite
    inputs:
      data: /srv/uptime-kuma/data
      db:   /srv/uptime-kuma/data/kuma.db
    check_timeout: 90s      # this target is slow over the offsite repository

  paperless:
    recipe: ./recipes-local/paperless   # a recipe that is not bundled yet
    source: home-nas
    inputs:
      data:  /srv/paperless/media
      db:    /srv/paperless/dumps/paperless.dump
    set:
      pg_version: "16"
    enabled: false          # skipped by --all; still runnable with --target paperless
```

Precedence, lowest to highest: recipe defaults, `defaults:`, the target block, then
command-line flags. A flag beats the config only when the user actually typed it: a
flag left at its default is not an opinion (ADR-068).

Two rules the file's author can rely on (ADR-067). Relative host-filesystem paths - a
target's `recipe` directory, a `password_file`, a dir source's `path`, a `workspace` -
resolve against the directory of the config file itself, never against the working
directory, because the file is found by a search order and read from cron; a restic
`repository` is left exactly as written, since it is a backend reference and not
necessarily a path. And `--all` runs the enabled targets sequentially in the order the
file declares them; a target with `enabled: false` is skipped by `--all` and still
runnable with `--target`.

A cron line that verifies everything nightly and lets the exit code do the alerting:

```cron
# m  h  dom mon dow  command
  17 4  *   *   *    /usr/local/bin/drillback check --all --json \
                       --report /var/log/drillback/$(date +\%F).json \
                       >>/var/log/drillback/run.log 2>&1 \
                     || curl -fsS -X POST https://ntfy.sh/my-backup-alerts \
                          -d "restore drill FAILED, see /var/log/drillback"
```

With `--all`, the process exit code is the worst outcome across targets: `2` if any
target hit a tool error, otherwise `1` if any target was unusable, otherwise `0`.

---

## 3. Recipe format

A recipe is a directory:

```text
recipes/gitea/
├── recipe.yaml      # the contract: inputs, ready probes, checks, test harness
├── compose.yaml     # how to start the app, using ${DRILLBACK_*} placeholders
├── README.md        # what this recipe assumes about your backup
└── test/            # optional assets used only by `drillback recipe test`
    └── seed.sql
```

The central idea: **a recipe declares logical inputs, not user paths.** The recipe
author knows that Gitea needs "a data directory" and "a database dump". Only the user
knows those live at `/srv/gitea/data` and `/var/backups/gitea.sql`. The recipe supplies
defaults; the user overrides them with `--input` or `drillback.yaml`.

### 3.1 Example: Gitea + PostgreSQL

`recipes/gitea/recipe.yaml`:

```yaml
apiVersion: drillback/v1
kind: Recipe

metadata:
  name: gitea
  title: Gitea + PostgreSQL
  description: >
    Verifies that a Gitea backup restores: the web UI renders, the repository and user
    rows are in the database, and at least one bare repository is present on disk.
  maintainers: ["@example-handle"]
  upstream: https://github.com/go-gitea/gitea
  tags: [git, forge, postgres]

vars:
  db_name: gitea
  db_user: gitea
  # Not a secret. This database exists for about ninety seconds on an internal
  # network with no published ports, and is destroyed with `compose down -v`.
  db_password: drillback-throwaway
  gitea_port: 3000

inputs:
  data:
    kind: dir
    title: Gitea data directory
    description: >
      Contains gitea-repositories/, attachments, avatars, and the SQLite-free app
      state. On a default docker-compose install this is the host side of /data.
    default_path: /srv/gitea/data
    required: true
    mount:
      env: DRILLBACK_INPUT_data      # compose.yaml refers to ${DRILLBACK_INPUT_data}
      into: gitea:/data             # service:path, for `file` checks and exports

  db:
    kind: postgres-dump
    title: Gitea database dump
    description: >
      Plain SQL from `pg_dump`, or a custom-format dump from `pg_dump -Fc`. The format
      is detected from the file's magic bytes, not from its extension.
    default_path: /srv/gitea/db.sql
    required: true
    load:
      service: db
      database: "{{ .vars.db_name }}"
      user: "{{ .vars.db_user }}"
      timeout: 5m

ready:
  - name: postgres accepts connections
    kind: exec
    service: db
    command: ["pg_isready", "-U", "{{ .vars.db_user }}", "-d", "{{ .vars.db_name }}"]
    timeout: 90s
    interval: 2s

  - name: gitea answers on the internal network
    kind: http
    url: http://gitea:{{ .vars.gitea_port }}/api/healthz
    expect_status: 200
    timeout: 180s
    interval: 3s

checks:
  - id: web-ui-renders
    title: The web UI renders the instance home page
    kind: http
    url: http://gitea:{{ .vars.gitea_port }}/
    expect:
      status: 200
      body_matches: "(?i)<title>[^<]*gitea"

  - id: repos-in-db
    title: The database contains at least one repository row
    kind: sql
    driver: postgres
    service: db
    database: "{{ .vars.db_name }}"
    user: "{{ .vars.db_user }}"
    query: "SELECT count(*) FROM repository;"
    expect:
      scalar_int_min: 1

  - id: users-in-db
    title: The database contains at least one real user account
    kind: sql
    driver: postgres
    service: db
    database: "{{ .vars.db_name }}"
    user: "{{ .vars.db_user }}"
    query: 'SELECT count(*) FROM "user" WHERE lower_name <> ''ghost'';'
    expect:
      scalar_int_min: 1

  - id: repo-files-on-disk
    title: At least one bare repository exists on disk
    kind: file
    service: gitea
    # The official image keeps bare repositories under /data/git/repositories, one
    # directory per owner. The input is the host side of /data, so this is the path a
    # default docker compose install actually produces.
    path: /data/git/repositories
    expect:
      exists: true
      glob: "*/*.git/HEAD"
      glob_min_count: 1

  - id: api-lists-repos
    title: The API lists repositories, so the database and the disk agree
    kind: http
    url: http://gitea:{{ .vars.gitea_port }}/api/v1/repos/search?limit=1
    expect:
      status: 200
      json_path: "$.data"
      json_path_len_min: 1

test:
  seed:
    - name: create an admin user
      kind: exec
      service: gitea
      user: git
      command:
        - gitea
        - admin
        - user
        - create
        - --username=drilluser
        - --password=drill-pass-123
        - --email=drill@example.invalid
        - --admin
        - --must-change-password=false
      timeout: 120s

    - name: create a repository through the API
      kind: http
      method: POST
      url: http://gitea:{{ .vars.gitea_port }}/api/v1/user/repos
      basic_auth: ["drilluser", "drill-pass-123"]
      json_body: '{"name":"drill-repo","auto_init":true,"private":false}'
      expect_status: 201
      timeout: 120s

  export:
    # `data` is a dir input: the harness exports it automatically by copying
    # gitea:/data back out into the staging tree. Only non-dir inputs need an
    # explicit export step.
    - name: dump the database into the staging tree
      kind: exec
      service: db
      # $DRILLBACK_EXPORT is mounted into every service by the harness, and only by
      # the harness. It does not exist during `drillback check`.
      command:
        - sh
        - -c
        - 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" > "$DRILLBACK_EXPORT/db.sql"'
      timeout: 5m
      produces: db      # the file lands at the `db` input's path in the staging tree
```

`recipes/gitea/compose.yaml`:

```yaml
# Rendered by drillback before `docker compose up`. The ${DRILLBACK_INPUT_*} values are
# absolute paths inside the run workspace, and ${DRILLBACK_VAR_*} come from `vars:`
# plus any --set overrides.
#
# Do not add `ports:`, `privileged:`, `network_mode:`, `pid:`, `ipc:`, or a bind mount
# to any path outside the workspace. `drillback recipe validate` rejects all of them.
services:
  db:
    image: postgres:16.4-alpine
    environment:
      POSTGRES_DB: ${DRILLBACK_VAR_db_name}
      POSTGRES_USER: ${DRILLBACK_VAR_db_user}
      POSTGRES_PASSWORD: ${DRILLBACK_VAR_db_password}
    # Speed only. This database is thrown away; durability is not a goal. These are
    # postmaster settings, so they go on the server's own command line: putting them
    # in PGOPTIONS breaks the image's own initialisation, which connects as a client.
    command:
      - postgres
      - -c
      - fsync=off
      - -c
      - full_page_writes=off
      - -c
      - synchronous_commit=off
    volumes:
      - db-data:/var/lib/postgresql/data
    networks: [drillback]

  gitea:
    image: gitea/gitea:1.22.6
    depends_on:
      db:
        condition: service_started
    environment:
      USER_UID: "1000"
      USER_GID: "1000"
      GITEA__database__DB_TYPE: postgres
      GITEA__database__HOST: db:5432
      GITEA__database__NAME: ${DRILLBACK_VAR_db_name}
      GITEA__database__USER: ${DRILLBACK_VAR_db_user}
      GITEA__database__PASSWD: ${DRILLBACK_VAR_db_password}
      GITEA__server__ROOT_URL: http://gitea:${DRILLBACK_VAR_gitea_port}/
      # The network is internal; without offline mode Gitea blocks on fetching
      # avatars and assets from the internet and the ready probe times out.
      GITEA__server__OFFLINE_MODE: "true"
      GITEA__security__INSTALL_LOCK: "true"
      GITEA__cron__ENABLED: "false"
    volumes:
      - ${DRILLBACK_INPUT_data}:/data
    networks: [drillback]

volumes:
  db-data:

networks:
  drillback:
    internal: true
```

### 3.2 Example: Uptime Kuma (SQLite)

`recipes/uptime-kuma/recipe.yaml`:

```yaml
apiVersion: drillback/v1
kind: Recipe

metadata:
  name: uptime-kuma
  title: Uptime Kuma (SQLite)
  description: >
    Verifies that an Uptime Kuma backup restores: the dashboard is served, the SQLite
    database passes an integrity check, and the configured monitors and users are
    still in it.
  maintainers: ["@example-handle"]
  upstream: https://github.com/louislam/uptime-kuma
  tags: [monitoring, sqlite]

vars:
  kuma_port: 3001

inputs:
  data:
    kind: dir
    title: Uptime Kuma data directory
    description: The whole /app/data directory, including kuma.db and upload/.
    default_path: /srv/uptime-kuma/data
    required: true
    mount:
      env: DRILLBACK_INPUT_data
      into: kuma:/app/data

  db:
    kind: sqlite
    title: The Uptime Kuma SQLite database
    description: >
      kuma.db, which lives inside the data directory. It is declared separately so the
      SQL checks have something to point at, and so that a zero-byte database, or one
      whose -wal companion was not backed up, is reported as a restore failure instead
      of as a mysteriously empty dashboard.
    default_path: /srv/uptime-kuma/data/kuma.db
    within: data           # this file is inside the `data` input; not restored twice
    required: true
    load:
      integrity_check: true   # sqlite needs no loading; drillback only verifies it

ready:
  # Uptime Kuma answers / with a 302 to /dashboard, so the readiness question is asked
  # of the API entry point instead: it is the cheapest endpoint that proves the server
  # is serving rather than merely listening.
  - name: kuma serves HTTP
    kind: http
    url: http://kuma:{{ .vars.kuma_port }}/api/entry-page
    expect_status: 200
    timeout: 180s
    interval: 3s

checks:
  - id: dashboard-renders
    title: The dashboard entrypoint renders
    kind: http
    url: http://kuma:{{ .vars.kuma_port }}/dashboard
    expect:
      status: 200
      body_matches: "(?i)uptime[ -]?kuma"

  - id: db-integrity
    title: The SQLite database passes PRAGMA integrity_check
    kind: sql
    driver: sqlite
    file: "{{ .inputs.db.path }}"
    query: "PRAGMA integrity_check;"
    expect:
      scalar_equals: "ok"

  - id: monitors-present
    title: At least one monitor survived the restore
    kind: sql
    driver: sqlite
    file: "{{ .inputs.db.path }}"
    query: "SELECT count(*) FROM monitor;"
    expect:
      scalar_int_min: 1

  - id: users-present
    title: At least one user account survived the restore
    kind: sql
    driver: sqlite
    file: "{{ .inputs.db.path }}"
    query: "SELECT count(*) FROM user;"
    expect:
      scalar_int_min: 1

  - id: heartbeats-present
    title: Heartbeat history survived the restore, so this is not a fresh database
    kind: sql
    driver: sqlite
    file: "{{ .inputs.db.path }}"
    query: "SELECT count(*) FROM heartbeat;"
    expect:
      scalar_int_min: 1

  - id: api-entry-page
    title: The API entry point answers
    kind: http
    url: http://kuma:{{ .vars.kuma_port }}/api/entry-page
    expect:
      status: 200
      json_path: "$.type"
      json_path_equals: "entryPage"

test:
  seed:
    # Uptime Kuma's first-run setup is a socket.io wizard, which is expensive to drive
    # from a shell. The checks above read the database, so seeding rows directly is
    # sufficient to make stage A fail and stage B pass. Documented tradeoff: this
    # exercises the restore path, not Kuma's own migration path. See SPEC.md § 7.4.
    - name: seed a user, a monitor, and a heartbeat
      kind: exec
      service: seeder
      # As root: Uptime Kuma runs as root and its -wal companion is root-owned, so
      # the sqlite3 image's own unprivileged user cannot write through it.
      user: "0"
      command: ["sh", "-c", "sqlite3 /app/data/kuma.db < /seed/seed.sql"]
      timeout: 120s
  export: []   # everything lives inside the `data` dir input, exported automatically
```

`recipes/uptime-kuma/compose.yaml`:

```yaml
services:
  kuma:
    image: louislam/uptime-kuma:1.23.16-alpine
    environment:
      UPTIME_KUMA_PORT: "${DRILLBACK_VAR_kuma_port}"
    volumes:
      - ${DRILLBACK_INPUT_data}:/app/data
    networks: [drillback]

  # Started only by `drillback recipe test`, via the "test" compose profile.
  seeder:
    image: keinos/sqlite3:3.46.0
    profiles: [test]
    volumes:
      - ${DRILLBACK_INPUT_data}:/app/data
      - ${DRILLBACK_TEST_ASSETS}:/seed:ro
    command: ["sleep", "infinity"]
    networks: [drillback]

networks:
  drillback:
    internal: true
```

`recipes/uptime-kuma/test/seed.sql`:

```sql
-- Minimal rows to make the data-sensitive checks meaningful. Column lists are
-- explicit so a schema change upstream fails loudly instead of silently seeding
-- nothing.
INSERT INTO user (username, password, active)
VALUES ('drilluser', '$2b$10$notarealhashnotarealhashnotarealhashnotarealhash', 1);

INSERT INTO monitor (name, type, url, interval, active, user_id)
VALUES ('drill monitor', 'http', 'http://kuma:3001/', 60, 0, 1);

INSERT INTO heartbeat (monitor_id, status, msg, time, ping, important, duration)
VALUES (1, 1, 'seeded by drillback recipe test', datetime('now'), 12, 1, 60);
```

### 3.3 Field reference

#### Input kinds

| kind | meaning | how it is materialised | how it is checked |
|---|---|---|---|
| `dir` | a directory tree | restored into the workspace, bind-mounted read-write at `mount.into` | `file` checks |
| `sqlite` | a single database file | restored into the workspace; if `within` is set it is *not* restored again, only located inside the parent input | `sql` checks with `driver: sqlite`; `load.integrity_check` runs `PRAGMA integrity_check` before any check |
| `postgres-dump` | a plain-SQL or custom-format dump | restored into the workspace, then loaded into `load.service` during the *load dumps* state | `sql` checks with `driver: postgres` |

Postgres dump format is detected from the first five bytes: `PGDMP` means custom,
directory, or tar format and is loaded with
`pg_restore --clean --if-exists --no-owner --no-acl`; anything else is treated as plain
SQL and loaded with `psql --set ON_ERROR_STOP=1 -f`. The detected format appears in the
report so a surprise is visible rather than mysterious.

#### Probe and check kinds

`ready` probes take `http`, `tcp`, `exec`. They are *retried* until they succeed or
their `timeout` expires, on `interval`. A ready probe that never succeeds ends the run
as **RESTORE UNUSABLE** (exit 1), not as a tool error — "the app did not come up" is a
verdict about the backup, not about `drillback`. See [DECISIONS.md](DECISIONS.md) ADR-023.

`checks` take `http`, `exec`, `sql`, `file`. They are run **once**, never retried, in
declaration order, and every check runs even after an earlier one fails — a report that
stops at the first failure hides the shape of the problem.

#### The `expect` vocabulary

Closed and small on purpose: a recipe is data, not a program. There is no expression
language, no scripting, and no way for a check to compute anything.

| key | applies to | meaning |
|---|---|---|
| `status` | http | exact HTTP status code |
| `status_in` | http | list of acceptable status codes |
| `body_matches` | http, exec | RE2 regular expression against the body / stdout |
| `body_not_matches` | http, exec | RE2 regular expression that must *not* match |
| `json_path` | http | a JSONPath selecting the value the next key tests |
| `json_path_equals` | http | selected value equals this string |
| `json_path_int_min` | http | selected value is an integer at least this large |
| `json_path_len_min` | http | selected array/string has at least this length |
| `exit_code` | exec | exact exit code (default `0`) |
| `stdout_matches` / `stderr_matches` | exec | RE2 against the stream |
| `scalar_equals` | sql | first column of first row equals this string |
| `scalar_int_min` / `scalar_int_max` | sql | first column of first row as an integer |
| `rows_min` / `rows_max` | sql | number of rows returned |
| `exists` | file | the path exists |
| `is_dir` | file | the path is a directory |
| `size_min` | file | file size in bytes, at least |
| `glob` + `glob_min_count` | file | at least N entries matching the glob below `path` |
| `not_empty` | file | a directory with at least one entry |

Regular expressions are Go's RE2: linear time, no backtracking, so a hostile recipe
cannot hang a runner with a catastrophic pattern.

#### Templating

Recipe strings are Go `text/template` with a restricted context: `.vars`,
`.inputs.<name>.path`, `.inputs.<name>.kind`, `.run.id`. No functions beyond
`printf`, `quote`, `default`. `compose.yaml` uses `${DRILLBACK_*}` shell-style
interpolation instead, because that is what `docker compose` itself does; the two
syntaxes are deliberately different so it is always obvious which engine expands a
placeholder.

### 3.4 JSON Schema

`schema/recipe.schema.json`, JSON Schema draft 2020-12. Abbreviated only where a
pattern repeats; every constraint that matters is present.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/spelingbee/drillback/schema/recipe.schema.json",
  "title": "drillback recipe",
  "type": "object",
  "additionalProperties": false,
  "required": ["apiVersion", "kind", "metadata", "inputs", "checks"],
  "properties": {
    "apiVersion": { "const": "drillback/v1" },
    "kind": { "const": "Recipe" },

    "metadata": {
      "type": "object",
      "additionalProperties": false,
      "required": ["name", "title", "description"],
      "properties": {
        "name": { "type": "string", "pattern": "^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$" },
        "title": { "type": "string", "minLength": 3, "maxLength": 80 },
        "description": { "type": "string", "minLength": 20 },
        "maintainers": {
          "type": "array",
          "items": { "type": "string", "pattern": "^@[A-Za-z0-9-]+$" }
        },
        "upstream": { "type": "string", "format": "uri" },
        "tags": { "type": "array", "items": { "type": "string" }, "uniqueItems": true }
      }
    },

    "vars": {
      "type": "object",
      "propertyNames": { "pattern": "^[a-z][a-z0-9_]*$" },
      "additionalProperties": { "type": ["string", "number", "boolean"] }
    },

    "inputs": {
      "type": "object",
      "minProperties": 1,
      "propertyNames": { "pattern": "^[a-z][a-z0-9_]*$" },
      "additionalProperties": {
        "type": "object",
        "additionalProperties": false,
        "required": ["kind", "title", "default_path"],
        "properties": {
          "kind": { "enum": ["dir", "sqlite", "postgres-dump"] },
          "title": { "type": "string", "minLength": 3 },
          "description": { "type": "string" },
          "default_path": { "$ref": "#/$defs/absolutePath" },
          "required": { "type": "boolean", "default": true },
          "within": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
          "mount": {
            "type": "object",
            "additionalProperties": false,
            "required": ["env", "into"],
            "properties": {
              "env": { "type": "string", "pattern": "^DRILLBACK_INPUT_[a-z][a-z0-9_]*$" },
              "into": { "type": "string", "pattern": "^[a-z0-9][a-z0-9_-]*:/[^\\s:]*$" }
            }
          },
          "load": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "service": { "$ref": "#/$defs/serviceName" },
              "database": { "type": "string" },
              "user": { "type": "string" },
              "timeout": { "$ref": "#/$defs/duration" },
              "integrity_check": { "type": "boolean" }
            }
          }
        },
        "allOf": [
          {
            "comment": "postgres-dump must say where to load it",
            "if": { "properties": { "kind": { "const": "postgres-dump" } } },
            "then": {
              "required": ["load"],
              "properties": {
                "load": { "required": ["service", "database", "user"] }
              }
            }
          },
          {
            "comment": "dir inputs must declare a mount",
            "if": { "properties": { "kind": { "const": "dir" } } },
            "then": { "required": ["mount"] }
          }
        ]
      }
    },

    "ready": {
      "type": "array",
      "items": { "$ref": "#/$defs/probe" }
    },

    "checks": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#/$defs/check" }
    },

    "test": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "seed":   { "type": "array", "items": { "$ref": "#/$defs/step" } },
        "export": { "type": "array", "items": { "$ref": "#/$defs/step" } }
      }
    }
  },

  "$defs": {
    "duration": { "type": "string", "pattern": "^[0-9]+(ns|us|ms|s|m|h)$" },

    "serviceName": { "type": "string", "pattern": "^[a-z0-9][a-z0-9_-]*$" },

    "absolutePath": {
      "type": "string",
      "pattern": "^/(?:[^/\\x00]+/)*[^/\\x00]*$",
      "not": { "pattern": "(^|/)\\.\\.(/|$)" },
      "description": "Absolute POSIX path as it appears inside the backup. No `..`."
    },

    "internalURL": {
      "type": "string",
      "pattern": "^https?://[a-z0-9][a-z0-9_.-]*(:([0-9]{1,5}|\\{\\{[^{}]*\\}\\}))?(/([^\\s{}]|\\{\\{[^{}]*\\}\\})*)?$",
      "not": {
        "pattern": "^https?://(localhost|127\\.|0\\.0\\.0\\.0|\\[::1\\]|host\\.docker\\.internal|169\\.254\\.)"
      },
      "description": "Must address a compose service by name on the run's internal network. Loopback, the host gateway, and link-local are rejected — a check that reaches the host is not checking the restore. `{{ ... }}` template placeholders are permitted in the port and path, because this schema validates the recipe file as written; see § 3.4.1."
    },

    "regex": { "type": "string", "maxLength": 512 },

    "probe": {
      "type": "object",
      "required": ["name", "kind"],
      "properties": {
        "name":     { "type": "string" },
        "kind":     { "enum": ["http", "tcp", "exec"] },
        "timeout":  { "$ref": "#/$defs/duration" },
        "interval": { "$ref": "#/$defs/duration" }
      },
      "allOf": [
        {
          "if": { "properties": { "kind": { "const": "http" } } },
          "then": {
            "required": ["url"],
            "properties": {
              "url": { "$ref": "#/$defs/internalURL" },
              "expect_status": { "type": "integer", "minimum": 100, "maximum": 599 }
            }
          }
        },
        {
          "if": { "properties": { "kind": { "const": "tcp" } } },
          "then": {
            "required": ["service", "port"],
            "properties": {
              "service": { "$ref": "#/$defs/serviceName" },
              "port": { "type": "integer", "minimum": 1, "maximum": 65535 }
            }
          }
        },
        {
          "if": { "properties": { "kind": { "const": "exec" } } },
          "then": {
            "required": ["service", "command"],
            "properties": {
              "service": { "$ref": "#/$defs/serviceName" },
              "user": { "type": "string" },
              "command": {
                "type": "array",
                "minItems": 1,
                "items": { "type": "string" },
                "description": "argv, never a shell string. Use [\"sh\",\"-c\",\"...\"] explicitly if you need a shell, so it is visible in review."
              }
            }
          }
        }
      ],
      "unevaluatedProperties": false
    },

    "check": {
      "type": "object",
      "required": ["id", "title", "kind", "expect"],
      "properties": {
        "id":    { "type": "string", "pattern": "^[a-z0-9][a-z0-9-]{1,48}$" },
        "title": { "type": "string", "minLength": 5 },
        "kind":  { "enum": ["http", "exec", "sql", "file"] },
        "timeout": { "$ref": "#/$defs/duration" },
        "expect": {
          "type": "object",
          "minProperties": 1,
          "additionalProperties": false,
          "properties": {
            "status":            { "type": "integer", "minimum": 100, "maximum": 599 },
            "status_in":         { "type": "array", "items": { "type": "integer" } },
            "body_matches":      { "$ref": "#/$defs/regex" },
            "body_not_matches":  { "$ref": "#/$defs/regex" },
            "json_path":         { "type": "string" },
            "json_path_equals":  { "type": "string" },
            "json_path_int_min": { "type": "integer" },
            "json_path_len_min": { "type": "integer", "minimum": 0 },
            "exit_code":         { "type": "integer" },
            "stdout_matches":    { "$ref": "#/$defs/regex" },
            "stderr_matches":    { "$ref": "#/$defs/regex" },
            "scalar_equals":     { "type": "string" },
            "scalar_int_min":    { "type": "integer" },
            "scalar_int_max":    { "type": "integer" },
            "rows_min":          { "type": "integer", "minimum": 0 },
            "rows_max":          { "type": "integer", "minimum": 0 },
            "exists":            { "type": "boolean" },
            "is_dir":            { "type": "boolean" },
            "not_empty":         { "type": "boolean" },
            "size_min":          { "type": "integer", "minimum": 0 },
            "glob":              { "type": "string" },
            "glob_min_count":    { "type": "integer", "minimum": 1 }
          },
          "dependentRequired": {
            "json_path_equals":  ["json_path"],
            "json_path_int_min": ["json_path"],
            "json_path_len_min": ["json_path"],
            "glob_min_count":    ["glob"]
          }
        }
      },
      "allOf": [
        {
          "if": { "properties": { "kind": { "const": "http" } } },
          "then": {
            "required": ["url"],
            "properties": {
              "url": { "$ref": "#/$defs/internalURL" },
              "method": { "enum": ["GET", "HEAD", "POST"] },
              "basic_auth": {
                "type": "array", "minItems": 2, "maxItems": 2,
                "items": { "type": "string" }
              },
              "json_body": { "type": "string" }
            }
          }
        },
        {
          "if": { "properties": { "kind": { "const": "sql" } } },
          "then": {
            "required": ["driver", "query"],
            "properties": {
              "driver":   { "enum": ["postgres", "sqlite"] },
              "query":    { "type": "string", "minLength": 5 },
              "service":  { "$ref": "#/$defs/serviceName" },
              "database": { "type": "string" },
              "user":     { "type": "string" },
              "file":     { "type": "string" }
            },
            "oneOf": [
              { "required": ["service"], "properties": { "driver": { "const": "postgres" } } },
              { "required": ["file"],    "properties": { "driver": { "const": "sqlite" } } }
            ]
          }
        },
        {
          "if": { "properties": { "kind": { "const": "file" } } },
          "then": {
            "required": ["service", "path"],
            "properties": {
              "service": { "$ref": "#/$defs/serviceName" },
              "path": { "type": "string", "pattern": "^/", "not": { "pattern": "(^|/)\\.\\.(/|$)" } }
            }
          }
        },
        {
          "if": { "properties": { "kind": { "const": "exec" } } },
          "then": {
            "required": ["service", "command"],
            "properties": {
              "service": { "$ref": "#/$defs/serviceName" },
              "user": { "type": "string" },
              "command": { "type": "array", "minItems": 1, "items": { "type": "string" } }
            }
          }
        }
      ],
      "unevaluatedProperties": false
    },

    "step": {
      "type": "object",
      "required": ["name", "kind"],
      "properties": {
        "name":    { "type": "string" },
        "kind":    { "enum": ["http", "exec"] },
        "timeout": { "$ref": "#/$defs/duration" },
        "produces": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
        "service": { "$ref": "#/$defs/serviceName" },
        "user":    { "type": "string" },
        "command": { "type": "array", "items": { "type": "string" } },
        "url":     { "$ref": "#/$defs/internalURL" },
        "method":  { "enum": ["GET", "POST", "PUT", "DELETE"] },
        "basic_auth": { "type": "array", "minItems": 2, "maxItems": 2, "items": { "type": "string" } },
        "json_body": { "type": "string" },
        "expect_status": { "type": "integer" }
      },
      "unevaluatedProperties": false
    }
  }
}
```

#### 3.4.1 When each schema runs

The two schemas validate at different moments, and the difference is load-bearing.

`recipe.schema.json` validates **the file as written**, with `{{ ... }}` placeholders
still in it. That is what makes it usable by an editor, by a contributor's
`check-jsonschema` invocation, and by the independent-validator CI job (§ 11.1) — none
of which can expand a template. Every pattern in the schema that could contain a
placeholder therefore permits one; `internalURL` is the case that matters, since a
templated port is the normal way to write a recipe.

`compose-safety.schema.json` validates **after** `${...}` interpolation, because the
whole point is to see the real, resolved values: `${DRILLBACK_INPUT_data}` must be
checked as the path it becomes, not as the literal placeholder.

The order in RESOLVE is: parse recipe → validate against `recipe.schema.json` → expand
templates → resolve inputs → interpolate `compose.yaml` → validate against
`compose-safety.schema.json` → run the three Go-only rules below.

### 3.5 The compose safety schema

`compose.yaml` is validated separately, after `${...}` interpolation, by
`schema/compose-safety.schema.json`. This is the machine-readable form of decision 9;
section 9 explains why each rule exists.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/spelingbee/drillback/schema/compose-safety.schema.json",
  "title": "drillback compose safety rules",
  "type": "object",
  "required": ["services", "networks"],
  "properties": {
    "services": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "required": ["image", "networks"],
        "not": {
          "anyOf": [
            { "required": ["ports"] },
            { "required": ["privileged"] },
            { "required": ["network_mode"] },
            { "required": ["pid"] },
            { "required": ["ipc"] },
            { "required": ["userns_mode"] },
            { "required": ["devices"] },
            { "required": ["device_cgroup_rules"] },
            { "required": ["cgroup_parent"] },
            { "required": ["build"] },
            { "required": ["extends"] },
            { "required": ["external_links"] }
          ]
        },
        "properties": {
          "image": {
            "type": "string",
            "pattern": "^[a-z0-9][a-z0-9._/-]*(:[A-Za-z0-9._-]+)?(@sha256:[a-f0-9]{64})?$",
            "not": { "pattern": ":latest$" },
            "description": "Pinned tag required. `latest` makes a recipe untestable over time."
          },
          "cap_add": {
            "type": "array",
            "items": { "enum": ["CHOWN", "DAC_OVERRIDE", "FOWNER", "SETGID", "SETUID"] },
            "description": "Allowlist. Anything outside it, and NET_ADMIN / SYS_ADMIN in particular, is rejected."
          },
          "volumes": {
            "type": "array",
            "items": {
              "oneOf": [
                {
                  "type": "string",
                  "pattern": "^(\\$\\{DRILLBACK_(INPUT|TEST_ASSETS|EXPORT)[A-Za-z0-9_]*\\}|[a-z][a-z0-9_-]*):/[^:]+(:(ro|rw|z|Z|ro,z|rw,z))?$",
                  "description": "Left side must be a named volume declared in this file, or a ${DRILLBACK_*} placeholder that drillback resolves inside the workspace. An absolute host path is rejected."
                },
                {
                  "type": "object",
                  "required": ["type", "target"],
                  "properties": {
                    "type": { "enum": ["volume", "tmpfs"] },
                    "target": { "type": "string", "pattern": "^/" }
                  },
                  "description": "Long syntax may not use type: bind. Bind mounts are only expressible through the ${DRILLBACK_*} short form, which drillback controls."
                }
              ]
            }
          },
          "security_opt": {
            "type": "array",
            "items": { "not": { "pattern": "(?i)(seccomp[:=]unconfined|apparmor[:=]unconfined|systempaths=unconfined)" } }
          }
        }
      }
    },
    "networks": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "required": ["internal"],
        "properties": {
          "internal": { "const": true },
          "external": { "const": false }
        }
      }
    }
  },
  "not": { "required": ["include"] }
}
```

Three rules cannot be expressed in JSON Schema and are enforced in Go, with the same
severity:

1. **No YAML tags.** The recipe loader rejects any `!!` tag or anchor cycle before
   parsing, so a recipe cannot exploit the YAML layer.
2. **Every `${...}` placeholder must be one drillback defines.** After collecting the
   recipe's `vars` and `inputs`, any remaining `${FOO}` in compose.yaml is an error —
   an unset variable silently expanding to the empty string is how a volume mount
   becomes `/`.
3. **`mount.into` service names must exist** in compose.yaml, and every service
   referenced by a probe, check, or step must exist too. A typo in a service name is
   caught at validate time, not sixty seconds into a run.

---

## 4. Run lifecycle

### 4.1 State machine

```text
                         ┌──────────────┐
                         │   RESOLVE    │  parse config + recipe, validate schema,
                         │   (no I/O    │  validate compose safety, resolve inputs,
                         │   budget)    │  check docker/compose/restic are present
                         └──────┬───────┘
                                │ ok                       any failure ⇒ exit 2
                                ▼
                         ┌──────────────┐
                         │   PREPARE    │  mkdir workspace, install signal handlers,
                         │    5s        │  allocate runid, register teardown
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │   RESTORE    │  restic restore --target <ws>/inputs
                         │   10m        │  (or copy/link for source=dir),
                         │              │  then sanitise the tree
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │  COMPOSE UP  │  docker compose -p drillback-<runid> up -d
                         │   5m         │  (pull policy applies here)
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │  LOAD DUMPS  │  psql / pg_restore per postgres-dump input;
                         │   5m each    │  PRAGMA integrity_check per sqlite input
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │    READY     │  retry each probe on its interval
                         │   5m total   │
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │    CHECKS    │  run every check once, in order,
                         │   60s each   │  never stop early
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │    REPORT    │  collect logs, match hints, render TTY/JSON,
                         │    10s       │  write --report, print the nudge
                         └──────┬───────┘
                                ▼
                         ┌──────────────┐
                         │   TEARDOWN   │  compose down -v --remove-orphans,
                         │    2m        │  rm -rf workspace
                         └──────────────┘
                              always
```

### 4.2 Per-state contract

| State | Budget (flag) | Failure means | Exit |
|---|---|---|---|
| RESOLVE | none | config unreadable, recipe invalid, unknown input name in `--input`, docker or compose or restic missing, docker daemon unreachable | 2 |
| PREPARE | 5s | cannot create the workspace, no space | 2 |
| RESTORE | 10m (`--restore-timeout`) | restic exits non-zero, snapshot not found, wrong password, a **required** input path absent from the snapshot | 2 |
| COMPOSE UP | 5m (part of `--timeout`) | image pull fails, compose rejects the file, a container exits immediately | 2 |
| LOAD DUMPS | 5m per input (`inputs.*.load.timeout`) | `psql`/`pg_restore` non-zero, sqlite integrity check fails | **1** |
| READY | 5m total (`--ready-timeout`) | a probe never succeeds | **1** |
| CHECKS | 60s each (`--check-timeout`) | any check fails its `expect` | **1** |
| REPORT | 10s | cannot write `--report` | 2 |
| TEARDOWN | 2m | best-effort; a failure is logged and downgrades the exit code to 2 only if the workspace could not be removed | 0/1/2 |

The distinction that matters: **states before LOAD DUMPS produce exit 2, states from
LOAD DUMPS onward produce exit 1.** The line falls where `drillback` stops being able to
fail on its own and starts observing the backup. A dump that will not load, an app that
never becomes ready, and a check that fails are all *verdicts about the backup*. See
ADR-023 — this is the single call in this specification most worth a second opinion.

### 4.3 Restore-stage sanitisation

Between RESTORE and COMPOSE UP, `drillback` walks the restored tree once and:

- resolves every symlink; any link whose target escapes the workspace is replaced with
  a zero-byte file, and the event is recorded in the report under `warnings` (a backup
  containing `/etc/shadow` symlinks is a real thing and must not become a read of the
  host's `/etc/shadow` from inside a container);
- rejects any path component equal to `..` (restic will not produce one, but the `dir`
  source is user-supplied);
- refuses to proceed if a required input resolved to a path outside
  `<workspace>/inputs/`;
- records the byte size and file count of each input, so an empty restore is visible in
  the report even when every check somehow passes.

### 4.4 Teardown guarantees

Teardown runs from a single deferred function, and is registered *before* the first
resource is created. It is triggered by: normal completion, any error, `SIGINT`,
`SIGTERM`, and a recovered panic. It is idempotent and time-boxed at 2 minutes.

- `docker compose -p drillback-<runid> down -v --remove-orphans --timeout 20`
- `os.RemoveAll(workspace)`

A second `SIGINT` during teardown skips it and exits 130, after printing the workspace
path and the compose project name so nothing is silently orphaned.

With `--keep`, teardown is skipped and the tool prints:

```text
Kept for inspection:
  workspace:        /var/tmp/drillback-k7m2q9xf
  compose project:  drillback-k7m2q9xf

Clean up with:
  docker compose -p drillback-k7m2q9xf down -v --remove-orphans
  rm -rf /var/tmp/drillback-k7m2q9xf
```

Orphans from a crashed process are recoverable: every object `drillback` creates carries
the label `com.drillback.run=<runid>`, so `docker ps -aq --filter label=com.drillback.run`
finds them all.

---

## 5. Report format

### 5.1 TTY output

> **DESIGN MOCKS.** Everything in section 5.1 is hand-written to specify the intended
> shape of the output. It is **never** to be copied into README.md, the website, or any
> other document. Real demo output is captured from real runs by
> `scripts/capture-demo.sh` into `docs/demo/*.txt` and included from there. See
> [CLAUDE.md](CLAUDE.md).

PASS:

```text
drillback 0.1.0 · recipe gitea · run k7m2q9xf

  source     restic  sftp:backup@nas.lan:/srv/restic
  snapshot   4a7f1c2e  2026-09-13 02:14:07  host=hypervisor  tags=[gitea]
  inputs     data  /srv/gitea/data     1.8 GiB   14,203 files
             db    /srv/gitea/db.sql   42.1 MiB  plain SQL

  restore    ok      18.4s
  compose    ok       6.1s   2 services
  load db    ok      11.7s   psql, 0 errors
  ready      ok      22.9s   postgres accepts connections, gitea answers

  CHECKS
  ✔  web-ui-renders       The web UI renders the instance home page        0.21s
  ✔  repos-in-db          The database contains at least one repository    0.04s
                          row  →  7
  ✔  users-in-db          The database contains at least one real user     0.03s
                          account  →  3
  ✔  repo-files-on-disk   At least one bare repository exists on disk      0.12s
                          →  7 matches for */*.git/HEAD
  ✔  api-lists-repos      The API lists repositories, so the database      0.19s
                          and the disk agree  →  1 item

  PASS  5/5 checks  ·  total 1m 02s  ·  teardown ok

This backup boots.
```

RESTORE UNUSABLE:

```text
drillback 0.1.0 · recipe gitea · run q4x8b1na

  source     restic  sftp:backup@nas.lan:/srv/restic
  snapshot   9c1e77b0  2026-09-13 02:14:07  host=hypervisor  tags=[gitea]
  inputs     data  /srv/gitea/data     1.8 GiB   14,203 files
             db    /srv/gitea/db.sql   88.3 KiB  plain SQL

  restore    ok      17.9s
  compose    ok       5.8s   2 services
  load db    ok       0.9s   psql, 0 errors
  ready      ok      21.4s   postgres accepts connections, gitea answers

  CHECKS
  ✔  web-ui-renders       The web UI renders the instance home page        0.20s
  ✘  repos-in-db          The database contains at least one repository    0.05s
                          row
                            query   SELECT count(*) FROM repository;
                            expect  scalar_int_min: 1
                            got     ERROR:  relation "repository" does not exist
                                    LINE 1: SELECT count(*) FROM repository;
  ✘  users-in-db          The database contains at least one real user     0.04s
                          account
                            expect  scalar_int_min: 1
                            got     ERROR:  relation "user" does not exist
  ✔  repo-files-on-disk   At least one bare repository exists on disk      0.11s
                          →  7 matches for */*.git/HEAD
  ✘  api-lists-repos      The API lists repositories, so the database      0.24s
                          and the disk agree
                            expect  status: 200
                            got     status: 500
                            body    {"message":"database error","url":"..."}

  RESTORE UNUSABLE  2/5 checks passed  ·  total 58s  ·  teardown ok

  LIKELY CAUSE
    The dump loaded without error but the application's tables are missing.
    A dump taken with `pg_dump --schema-only`, or from the wrong database, or
    restricted with `--table`, produces exactly this: psql succeeds, the schema
    is not there.

    Check what the dump actually contains:
      head -50 /srv/gitea/db.sql | grep -i 'CREATE TABLE'
      pg_restore --list /srv/gitea/db.sql | head

    The dump is 88.3 KiB for a 1.8 GiB data directory with 7 repositories on
    disk. That ratio is itself suspicious.
                                              (hint: postgres/relation-missing)

  Service logs from the failure window are in the JSON report (--report).
  Re-run with --keep to keep the stack up and poke at it yourself.
```

Design rules the mocks encode:

- The verdict is a **word**, on its own line, in the same column every time. Colour is
  an enhancement, never the only signal: `PASS` / `RESTORE UNUSABLE` and `✔` / `✘` read
  identically through `NO_COLOR`, `| cat`, and a screenshot pasted into an issue.
- Every failing check prints **expect and got side by side**. A report that says "check
  failed" and nothing else forces the user into `--keep`, which is exactly the friction
  the tool exists to remove.
- Passing checks print their observed value too (`→ 7`), because "passed with 0 rows"
  and "passed with 7 rows" are different levels of confidence and the user should see
  which they got.
- The header always states which snapshot and which paths, so a report pasted into an
  issue is self-contained.
- **At most one** hint is printed. A list of five possible causes is a list of five
  things to ignore.

### 5.2 JSON report

`drillback check --json` writes this document to stdout; human output goes to stderr.
`--report FILE` writes the same document to a file regardless of `--json`.

```json
{
  "schema_version": 1,
  "tool": { "name": "drillback", "version": "0.1.0", "commit": "3f9a1c4e" },
  "run": {
    "id": "q4x8b1na",
    "compose_project": "drillback-q4x8b1na",
    "started_at": "2026-09-14T02:31:08Z",
    "finished_at": "2026-09-14T02:32:06Z",
    "duration_ms": 58412,
    "workspace_removed": true
  },
  "verdict": "RESTORE_UNUSABLE",
  "exit_code": 1,
  "recipe": {
    "name": "gitea",
    "title": "Gitea + PostgreSQL",
    "source": "bundled",
    "digest": "sha256:0f6d…c3a1"
  },
  "source": {
    "kind": "restic",
    "repository": "sftp:backup@nas.lan:/srv/restic",
    "snapshot": {
      "id": "9c1e77b0",
      "time": "2026-09-13T02:14:07Z",
      "hostname": "hypervisor",
      "tags": ["gitea"],
      "selected_by": "latest"
    }
  },
  "inputs": [
    {
      "name": "data", "kind": "dir",
      "backup_path": "/srv/gitea/data",
      "bytes": 1932735283, "files": 14203,
      "source": "recipe_default"
    },
    {
      "name": "db", "kind": "postgres-dump",
      "backup_path": "/srv/gitea/db.sql",
      "bytes": 90419, "files": 1,
      "detected_format": "plain",
      "source": "recipe_default"
    }
  ],
  "stages": [
    { "name": "restore",   "status": "ok", "duration_ms": 17903 },
    { "name": "compose_up","status": "ok", "duration_ms": 5811, "services": ["db", "gitea"] },
    { "name": "load_dumps","status": "ok", "duration_ms": 912,
      "detail": { "db": { "loader": "psql", "stderr_lines": 0 } } },
    { "name": "ready",     "status": "ok", "duration_ms": 21447,
      "probes": [
        { "name": "postgres accepts connections", "status": "ok", "attempts": 4, "duration_ms": 7102 },
        { "name": "gitea answers on the internal network", "status": "ok", "attempts": 6, "duration_ms": 14345 }
      ] }
  ],
  "checks": [
    {
      "id": "web-ui-renders",
      "title": "The web UI renders the instance home page",
      "kind": "http",
      "status": "pass",
      "duration_ms": 204,
      "expect": { "status": 200, "body_matches": "(?i)<title>[^<]*gitea" },
      "observed": { "status": 200, "body_bytes": 18422, "matched": true }
    },
    {
      "id": "repos-in-db",
      "title": "The database contains at least one repository row",
      "kind": "sql",
      "status": "fail",
      "duration_ms": 51,
      "expect": { "scalar_int_min": 1 },
      "observed": {
        "error": "ERROR:  relation \"repository\" does not exist\nLINE 1: SELECT count(*) FROM repository;"
      },
      "query": "SELECT count(*) FROM repository;"
    }
  ],
  "summary": { "checks_total": 5, "checks_passed": 2, "checks_failed": 3, "checks_skipped": 0 },
  "hint": {
    "id": "postgres/relation-missing",
    "matched_on": "checks[1].observed.error",
    "title": "The dump loaded but the application's tables are missing",
    "text": "A dump taken with `pg_dump --schema-only`, or from the wrong database, or restricted with `--table`, produces exactly this…",
    "commands": [
      "head -50 /srv/gitea/db.sql | grep -i 'CREATE TABLE'",
      "pg_restore --list /srv/gitea/db.sql | head"
    ]
  },
  "warnings": [
    { "code": "symlink_escaped_workspace", "detail": "inputs/data/log/current -> /var/log/gitea (neutralised)" }
  ],
  "logs": {
    "db":    ["2026-09-14T02:31:20Z LOG:  database system is ready to accept connections"],
    "gitea": ["2026-09-14T02:31:41Z ...level=ERROR ...pq: relation \"repository\" does not exist"]
  },
  "nudge": { "shown": false, "reason": "recipe is bundled" }
}
```

Stability contract: `schema_version` is bumped on any breaking change. Within a major
version, fields are only added, never removed or retyped. `verdict`, `exit_code`,
`summary.*`, and `checks[].status` are the fields automation is expected to depend on,
and they are frozen for v0.x.

With `--all`, the document is instead `{"schema_version":1, "runs":[ … ]}` where each
element is exactly the document above plus a `target` field naming the target that
produced it - additive, which is what the stability contract permits, and without it
two targets sharing a recipe are indistinguishable. The top-level `summary` counts
targets rather than checks (`targets_total`, `targets_passed`, `targets_unusable`,
`targets_errored`, `duration_ms`), and the top-level `exit_code` is the worst outcome
across targets: 2 if any target hit a tool error, otherwise 1 if any restore was
unusable, otherwise 0. See ADR-068.

### 5.3 Exit codes

| Code | Name | Meaning | Typical cron reaction |
|---|---|---|---|
| 0 | PASS | Every check passed. | Nothing. |
| 1 | RESTORE UNUSABLE | The stack came up or failed to, and the data is not there. **This backup will not save you.** | Page someone. |
| 2 | ERROR | `drillback` could not perform the test: docker missing, restic failed, recipe invalid, snapshot not found, out of disk. **The backup is unproven, not proven bad.** | Fix the runner, then re-run. Do not treat as PASS. |
| 130 | interrupted | Second `SIGINT` during teardown. | — |

The 1/2 split exists so a cron job can tell "your backup is broken" from "your drill is
broken". Conflating them is how a monitoring system trains its owner to ignore it.

Exit 2 is a statement about `drillback`, never about the backup. A run that exceeds
`--timeout` is exit 2 for this reason: the drill did not finish, so nothing is known
(ADR-058).

### 5.4 The harness JSON report

`drillback recipe test --json` and `--report FILE` write a *different* document from
`drillback check`. It is one document per invocation, however many recipes it covered,
and it nests a check report per stage.

```json
{
  "schema_version": 1,
  "tool": { "name": "drillback", "version": "0.1.0" },
  "started_at": "2026-08-30T02:38:11Z",
  "finished_at": "2026-08-30T02:39:04Z",
  "duration_ms": 53000,
  "summary": { "total": 1, "passed": 1, "failed": 0, "errored": 0 },
  "recipes": [
    {
      "recipe": "gitea",
      "status": "pass",
      "stages": [
        {
          "name": "A",
          "status": "pass",
          "reason": "3 of 5 checks failed against an empty stack: repos-in-db, ...",
          "duration_ms": 21000,
          "phases": [ { "name": "up", "status": "pass", "duration_ms": 4200 } ],
          "command": "drillback recipe test ./recipes/gitea --stage a --keep"
        }
      ]
    }
  ]
}
```

Two things about it are contracts.

**`schema_version` is the integer `1`, the same field name and the same type as the
check report's.** They are separate contracts with separate version numbers, but a
consumer doing `jq -e '.schema_version == 1'` must not have to know which document it
is holding. (It was a string here and an integer there until session 4; see
`docs/review/ux.md` UX-06.)

**`stages[].check` is the whole check report for a stage that did not pass**, in the
shape of section 5.2 - the per-check query, expectation and observation, the service
logs, and the hint. It is omitted for a stage that passed, because nobody reads a
passing check's observations and this document is uploaded as a CI artifact. Without
it, a contributor whose recipe fails in CI is told which checks failed and nothing
else, while everything they need has already been computed and discarded (ADR-061).

`stages[].command` is a command that still works after the run: the harness deletes its
workspaces unless `--keep` was given, so it names `drillback recipe test <ref> --stage
<a|b> --keep` rather than a path that no longer exists.

---

## 6. Hints

### 6.1 Mechanism

After the checks run and before the report renders, `drillback` matches a catalog of
regular expressions against, in order: each failing check's `observed.error`, its
`observed.body`, its stderr, and then the last 200 log lines of each service. The first
rule that matches wins; at most one hint is ever shown.

Rules are ordered in the file, most specific first. Each rule may be scoped with
`when:` so a Postgres rule cannot fire on an SQLite failure.

The catalog ships embedded (`docs/hints.yaml`, `go:embed`). `--hints FILE` loads an
additional file whose rules are matched **before** the built-in ones, so a user or a
downstream distribution can add rules without a rebuild. A hint is presentation only:
it can never change the verdict or the exit code.

Adding a hint is the smallest possible contribution to this project — one YAML block,
no Go, no Docker — and is deliberately advertised as the first-timer entry point.

### 6.2 `docs/hints.yaml`

```yaml
# docs/hints.yaml — error pattern catalog.
#
# Matched in order; first match wins; at most one hint is shown per run.
# `match` is an RE2 regular expression (Go's regexp), applied to check errors,
# response bodies, and service logs.
#
# Adding a rule here is the easiest useful contribution to drillback. If you hit a
# confusing restore failure, the fix is often just twenty lines in this file.
version: 1

rules:
  - id: postgres/relation-missing
    when: { driver: postgres }
    match: 'relation "([^"]+)" does not exist'
    title: The dump loaded but the application's tables are missing
    text: >
      psql reported no error, yet the table the application needs is not there. A dump
      taken with `pg_dump --schema-only`, from the wrong database, or narrowed with
      `--table`/`--schema`, produces exactly this. It can also happen when the dump
      restored into a different schema than the one on the application's search_path.
    commands:
      - "grep -ci 'CREATE TABLE' {{ .input.db.path }}"
      - "pg_restore --list {{ .input.db.path }} | head -30"

  - id: postgres/empty-dump
    when: { driver: postgres }
    match: '(?i)(input file is too short|archive is empty|unexpected end of file|pg_restore: error: did not find magic string)'
    title: The dump file is truncated or empty
    text: >
      The dump is not a complete file. The usual cause is a backup script that pipes
      `pg_dump` into a file without checking the exit code, so a failed or interrupted
      dump is backed up as a truncated file every night. Add `set -o pipefail` and check
      the exit status before the backup runs.
    commands:
      - "ls -l {{ .input.db.path }}"
      - "tail -c 200 {{ .input.db.path }}"

  - id: db/tables-empty
    match: 'expected scalar_int_min: [1-9][0-9]*, got 0'
    title: The application's tables are there, but they are empty
    text: >
      Every table the checks read exists and holds nothing. Two causes produce exactly
      this. Either the dump was taken from the wrong database, or with
      `pg_dump --schema-only`, or narrowed with `--table`, so it carried a schema and
      none of the rows. Or the dump carried nothing at all and the application rebuilt
      an empty schema for itself on start, which is what an application with automatic
      migrations does the moment it meets an empty database. Compare the size of the
      dump with the size of the data directory in the report above: a forge with
      repositories on disk and a half-kilobyte dump is not a backup.
    commands:
      - "grep -c 'INSERT INTO' {{ .input.db.path }}"
      - "ls -l {{ .input.db.path }}"

  - id: postgres/role-missing
    when: { driver: postgres }
    match: 'role "([^"]+)" does not exist'
    title: The dump references a database role that does not exist here
    text: >
      The dump carries ownership and GRANT statements for roles that live in the source
      cluster's globals, which `pg_dump` of a single database does not include. For a
      restore drill this is usually harmless noise — drillback loads custom-format dumps
      with `--no-owner --no-acl` for exactly this reason — but with a plain-SQL dump the
      statements run and fail. Re-dump with `--no-owner --no-acl`, or back up globals
      too with `pg_dumpall --globals-only`.

  - id: postgres/version-mismatch
    when: { driver: postgres }
    match: '(?i)(unsupported version .* in file header|server version mismatch|pg_restore: error: unsupported version)'
    title: The dump was made by a newer PostgreSQL than the recipe's image
    text: >
      A dump from PostgreSQL N cannot be restored into N-1. Set the image tag in the
      recipe's compose.yaml, or the `pg_version` variable if the recipe has one, to
      match the server the dump came from.
    commands:
      - "head -c 512 {{ .input.db.path }} | strings | head -5"

  - id: postgres/auth-failed
    when: { driver: postgres }
    match: '(?i)(password authentication failed|no pg_hba.conf entry|FATAL:.*role .* does not exist)'
    title: The recipe's database credentials do not match its compose file
    text: >
      This is a recipe bug, not a backup problem. The `db_user`/`db_name` variables in
      recipe.yaml and the POSTGRES_* environment in compose.yaml have drifted apart.
      `drillback recipe show <name>` prints the resolved values.

  - id: sqlite/not-a-database
    when: { driver: sqlite }
    match: '(?i)(file is not a database|file is encrypted or is not a database)'
    title: The SQLite file is not a database
    text: >
      Either the file was replaced by something else (an HTML error page and a
      zero-length file are both common when a download or an rclone copy failed), or it
      is encrypted, or only part of it was backed up. Check the first 16 bytes: a real
      SQLite file starts with `SQLite format 3`.
    commands:
      - "head -c 16 {{ .input.db.path }} | xxd"
      - "ls -l {{ .input.db.path }}"

  - id: sqlite/wal-missing
    when: { driver: sqlite }
    match: '(?i)(database disk image is malformed|integrity_check.*(row [0-9]+ missing|page [0-9]+)|database is locked)'
    title: The database was copied while the application was writing to it
    text: >
      A live SQLite database is a `.db` plus a `-wal` and a `-shm` file, and copying only
      the `.db` gives you a torn image — sometimes valid but stale, sometimes malformed.
      Back up with `sqlite3 db '.backup out.db'`, or stop the container for the copy, or
      include the `-wal` and `-shm` files.
    commands:
      - "ls -l {{ .input.db.path }}*"

  - id: sqlite/empty-schema
    when: { driver: sqlite }
    match: '(?i)no such table: ([A-Za-z0-9_]+)'
    title: The database restored, but it has none of the application's tables
    text: >
      The file is a valid SQLite database with a different, or empty, schema. The usual
      cause is backing up the path where the application *would* keep its database
      rather than where it does — a named Docker volume is not the bind-mounted
      directory next to it.
    commands:
      - "sqlite3 {{ .input.db.path }} '.tables'"

  - id: compose/image-pull-failed
    match: '(?i)(manifest unknown|pull access denied|repository does not exist|toomanyrequests|failed to resolve reference)'
    title: The recipe's image could not be pulled
    text: >
      Not a backup problem. Either the tag pinned in the recipe has been deleted
      upstream, or this host has hit an anonymous Docker Hub rate limit, or there is no
      network. `docker login` raises the rate limit; `--pull never` uses whatever is
      already in the local image cache.

  - id: compose/port-conflict
    match: '(?i)(bind: address already in use|port is already allocated)'
    title: A recipe is publishing a port, which drillback does not allow
    text: >
      drillback never publishes ports; every check runs on the run's internal network.
      If you see this, a recipe has a `ports:` entry that slipped past validation.
      Please open an issue with the recipe name — `drillback recipe validate` should have
      caught it.

  - id: permissions/eacces
    match: '(?i)(permission denied|EACCES|operation not permitted|chown.*Operation not permitted)'
    title: The restored files are owned by a UID the container cannot use
    text: >
      The application inside the container runs as a fixed UID (Gitea 1000, Nextcloud
      33, Paperless 1000). The backup preserved the ownership from the original host,
      and it does not match. On rootless Docker the mismatch is systematic, because the
      user namespace shifts every UID. Set USER_UID/USER_GID or PUID/PGID in the
      recipe's compose.yaml to the UID the files actually have — `drillback check --keep`
      then `ls -ln` in the workspace shows you which.

  - id: restore/empty-input
    match: '(?i)restored input "([a-z0-9_]+)" is empty'
    title: The path exists in the snapshot but contains nothing
    text: >
      The backup ran, the directory was included, and it was empty when it ran. The
      classic cause is backing up a bind-mount path while the application actually
      stores its data in a *named volume* mounted over the top of it, so the host
      directory the backup reads is a hollow shell. `docker inspect <container>
      --format '{{ .Mounts }}'` on the live system shows where the data really is.

  - id: restore/path-not-in-snapshot
    match: '(?i)(no matching files found for|path .* not found in snapshot|ls: .*: no such file or directory)'
    title: The recipe's default path is not where your backup keeps that data
    text: >
      Recipe defaults are a guess at the most common layout. Point the input at your
      path with `--input <name>=/your/path`, or record it in drillback.yaml. To see what
      the snapshot actually contains, use `restic ls latest | head -50`.
    commands:
      - "restic ls {{ .snapshot.id }} | head -50"

  - id: app/still-in-setup
    match: '(?i)(install|setup) (wizard|page)|/install\?|first[- ]run setup'
    title: The application booted into its first-run setup wizard
    text: >
      The app started with an empty configuration, which means the config file, not just
      the data, is missing from the backup. Everything looks healthy and nothing is
      restored. Many recipes set an INSTALL_LOCK-style variable to make this failure
      loud instead of quiet; if this recipe does not, that is worth an issue.

  - id: compose/service-exited-at-boot
    match: "(?i)(could not resolve host|name or service not known|no such host|temporary failure in name resolution)"
    title: A service exited before the probe could reach it
    text: >
      Docker's embedded DNS resolves a service name only while a container for it
      exists, so "could not resolve host" almost never means DNS is broken - it means
      the container is gone. The usual cause is an application that treats a refused
      first database connection as fatal.
    commands:
      - "drillback recipe test ./recipes/<name> --stage b --keep --log-level debug"

  - id: docker/daemon-unreachable
    match: '(?i)(cannot connect to the docker daemon|docker daemon is not running|/var/run/docker\.sock.*permission denied)'
    title: Docker is not reachable from here
    text: >
      drillback needs a working `docker` and `docker compose` v2. On rootless Docker,
      check that DOCKER_HOST points at your user socket. If you are inside WSL2, the
      Docker Desktop integration for this distribution may be switched off.
    commands:
      - "docker version"
      - "docker compose version"

  - id: workspace/no-space
    match: '(?i)(no space left on device|ENOSPC|write .*: disk quota exceeded)'
    title: The workspace ran out of disk
    text: >
      The whole restore is materialised on disk before the stack starts, so you need
      roughly as much free space as the inputs are large. Point --workspace at a
      filesystem with room; the default is the OS temp directory, which is often a small
      tmpfs in RAM.
```

That is 18 rules. `commands` are rendered with the same restricted template context as
recipes and are printed verbatim — `drillback` never executes them.

---

## 7. Round-trip harness

### 7.1 Why it exists

A recipe is a claim: *if these inputs are restored, these checks pass, and if they are
not, they fail.* Both halves have to be true. A recipe whose checks pass against an
empty database is worse than no recipe — it manufactures false confidence, which is the
exact failure mode the whole tool exists to destroy.

The harness proves both halves mechanically, so accepting a recipe PR requires no
maintainer judgement about whether the checks are meaningful.

### 7.2 Stage A — negative

```text
1. workspace/  with an EMPTY tree for every input:
     dir            → an empty directory
     sqlite         → a zero-length file
     postgres-dump  → a file containing only "-- intentionally empty\n"
2. compose up  (the "test" profile is NOT enabled)
3. run ready probes, with a reduced budget (90s): the app is expected to be
   startable-but-empty. If it never becomes ready, stage A is INCONCLUSIVE, not a
   failure — some apps refuse to boot with no data at all, which is itself acceptable
   evidence that the checks are data-sensitive. Report it as PASS-BY-STARTUP-REFUSAL
   and say so.
4. run every check
5. PASS if at least one check FAILED.
   FAIL if every check passed:  "recipe has no data-sensitive check"
6. teardown
```

The wording of the rejection is deliberate. It names what is missing rather than what
went wrong, because the author's next action is to add a check, not to debug one.

### 7.3 Stage B — positive, the round trip

```text
1. fresh workspace. NOT the same empty inputs as stage A: only the dir inputs, as
   empty directories, plus any non-dir input that compose bind-mounts. Stage B has
   nothing to restore and the application is about to create its own world, so a
   zero-length database file would stop it dead rather than start it clean.
   See DECISIONS.md ADR-053.
2. compose up --profile test
3. ready probes  (full budget)
4. run test.seed steps in order
5. run test.export steps in order
   - every dir input is exported automatically by copying <service>:<mount.into>
     out of the container into  workspace/staging/<default_path>
   - $DRILLBACK_EXPORT is bind-mounted at /export in every service, and maps to
     workspace/staging; a step with `produces: db` must leave its artifact where
     the `db` input's default_path says it will be
6. initialise a throwaway restic repository. restic runs in a pinned container with
   each staged input bind-mounted at its own default_path, so the snapshot records
   /srv/gitea/data rather than a path inside the workspace and step 8 needs no
   --input override (DECISIONS.md ADR-051):
     RESTIC_PASSWORD=drillback-recipe-test               # not a secret, deleted at step 9
     restic --repo /repo init --repository-version 2
     restic --repo /repo backup --tag drillback-recipe-test <each input's default_path>
7. teardown the stack completely (compose down -v), so nothing survives in a volume
   and the next step cannot accidentally read live state
8. run the real code path, not a special one:
     drillback check --recipe <dir> --source restic --from <workspace>/repo --snapshot latest
9. PASS if that run exits 0 and every check passed. Then teardown and delete the
   repository with the rest of the workspace.
```

Step 8 is the point of the whole design: **stage B ends by running exactly the command
a user runs.** No test-only restore path exists, so the harness cannot pass while the
real path is broken.

The throwaway restic repository is created with `--repository-version 2`, no
compression tuning, and a fixed literal password. It lives inside the run workspace and
is destroyed with it. Nothing about it is configurable, because a recipe author should
never have to think about restic to contribute a recipe.

### 7.4 Time budget

| Phase | Budget | Notes |
|---|---|---|
| Stage A | 5m | short ready budget (90s), then the checks |
| Stage B, seed | 5m | the usual bottleneck; app first-run migrations live here |
| Stage B, export + restic | 3m | test fixtures are small by construction |
| Stage B, `drillback check` | 7m | the normal run budget |
| Total per recipe | **20m** (`--timeout`) | CI job budget is 25m, giving teardown room |

A recipe that cannot complete a round trip in 20 minutes is out of scope for v0.1 and
should say so in its README. That is a real constraint on which applications get
recipes, and it is better stated than discovered.

Where stage B is a *documented approximation* — the Uptime Kuma recipe seeds SQLite
rows directly rather than driving the socket.io setup wizard — the recipe must say so
in a comment on the seed step. The harness proves the restore path, and the comment
records honestly which part of the application path it does not exercise.

### 7.5 How CI selects changed recipes

```sh
# .github/workflows/recipes.yml — the selection step
git diff --name-only "origin/${BASE_REF}...HEAD" \
  | grep -E '^recipes/[^/]+/' \
  | cut -d/ -f2 \
  | sort -u > changed.txt
```

- Empty `changed.txt` → the job is skipped, and reports success.
- Non-empty → a matrix job per recipe, `fail-fast: false`, so five changed recipes give
  five independent verdicts rather than one truncated log.
- A change to `internal/**`, `schema/**`, or `docs/hints.yaml` sets `changed.txt` to
  **all** recipes, because those are the things that can break every recipe at once.
- The weekly scheduled run always uses all recipes (section 11.4).
- Matrix size is capped at 20; above that the job falls back to running all recipes
  sequentially in one runner, which is slower but does not exhaust the concurrency
  allowance.

---

## 8. Contribution nudge

### 8.1 Behaviour

When a `check` run finishes and **all** of these hold:

1. the verdict is PASS,
2. the recipe came from a path, not from the bundled registry,
3. `--no-nudge` was not given and `defaults.nudge` is not `false`,
4. stdout or stderr is a TTY (never in `--json`, never in cron),
5. the recipe passes `recipe validate --strict`,

`drillback` prints:

```text
  ────────────────────────────────────────────────────────────────────────
  This recipe is not in the bundled registry, and it just proved a restore.
  Other people running Paperless-ngx would use it. Adding it is a fork and a
  four-line pull request:

    1. fork  https://github.com/spelingbee/drillback
    2. cp -r /home/you/recipes/paperless recipes/paperless
    3. drillback recipe test ./recipes/paperless     # this is what CI runs
    4. open a PR

  drillback does not touch your clipboard. Your recipe is at
  /home/you/recipes/paperless.

  (silence this with --no-nudge, or `nudge: false` in drillback.yaml)
  ────────────────────────────────────────────────────────────────────────
```

Condition 5 matters: nudging someone toward a PR that CI will immediately reject wastes
their goodwill, which is the scarcest resource this project has.

The invitation is one sentence, states the reason ("other people running X would use
it"), and is silenceable in the same breath. It appears at most once per run, after the
verdict, never before it.

### 8.2 Why there is no URL

There used to be one: a `github.com/.../new/main?filename=...&value=...` link that
opened GitHub's file editor with the recipe prefilled. It is gone, for two reasons that
compound. See DECISIONS.md ADR-065 and ADR-066.

It produced a branch containing `recipe.yaml` and nothing else. A recipe is a
directory; `compose.yaml` is not optional; and the first thing `recipes.yml` does to a
branch is `recipe validate`, which cannot pass without it. So the highest-volume
acquisition surface in the project pointed at a guaranteed red X - which is precisely
what condition 5 above exists to prevent, one level up.

And it was unreadable. A recipe percent-encodes to a few thousand characters, and a few
thousand characters of `%0A` and `%3A` wrapped across twenty lines of a terminal is not
a shortcut; it pushes the report the user is reading off the screen. That went unnoticed
for three sessions because the invitation only fires on a TTY and everything - tests,
golden files, five independent reviewers - captured output through a pipe. Rendering
`docs/demo/demo.tape` is what found it, and is now the regression test for it.

What is sent instead is four lines of instruction. The recipe is left on disk, as the
user wrote it, with two exceptions applied to the file the instructions point at:

- any `--set` and `--input` overrides are folded back into `vars` and `default_path`, so
  the recipe that gets contributed reflects what actually worked;
- `metadata.maintainers` is left exactly as it is. `drillback` does not guess the
  contributor's GitHub handle and does not read git config for it.

`drillback` never opens a browser, never writes to the clipboard, and never sends
anything anywhere. It prints four lines and stops.

---

## 9. Threat model

### 9.1 What `drillback` is asked to do

Run somebody else's YAML, which starts somebody else's container images, over the
contents of a backup that may itself be adversarial, on a machine with a Docker socket.
Every one of those three is a trust boundary.

### 9.2 Assets

| Asset | Why it is attractive |
|---|---|
| The host filesystem | Docker socket access is root-equivalent |
| The backup repository and its password | The user's entire data set |
| Cloud credentials in the environment (`AWS_*`, `B2_*`) | Lateral movement |
| The CI runner and its `GITHUB_TOKEN` | Supply-chain foothold in this project |
| The user's other running containers | Data and secrets |

### 9.3 Trust boundaries and mitigations

**A recipe is code.** A recipe chooses container images and shell commands that run on
the user's machine. There is no sandbox that makes an arbitrary recipe safe, and
pretending otherwise would be the most dangerous thing in this document.

- *Mitigated:* the structural escapes. No `privileged`, no `network_mode: host`, no
  `pid`/`ipc: host`, no `devices`, no `cgroup_parent`, no `userns_mode`, no `build:`
  (which would run a Dockerfile), no `extends`/`include` (which would pull in files
  outside the recipe), no `security_opt` unconfining seccomp or AppArmor, `cap_add`
  restricted to a five-entry allowlist, all networks `internal: true`, no published
  ports, and bind mounts expressible only through `${DRILLBACK_*}` placeholders that
  `drillback` resolves inside its own workspace. All of these are schema-level hard
  failures (section 3.5), enforced before anything is started.
- *Mitigated:* a runaway recipe. Every stage is time-boxed; teardown is unconditional;
  every object is labelled `com.drillback.run=<runid>` so orphans are findable.
- *Accepted and documented:* a recipe can still name a malicious image, and that image
  runs with normal container privileges on the user's Docker. Recipes are therefore
  treated as **code review artifacts**: bundled recipes are reviewed like Go code,
  images must be pinned to a tag (`:latest` is rejected), and the README says in plain
  words: *running a recipe from a stranger is running a container from a stranger.*
- *Accepted:* `drillback` does not verify image signatures in v0.1. Digest pinning is
  supported by the schema and encouraged; it is not required, because requiring it
  would make recipes stale within weeks and destroy the contribution flow that is the
  project's entire point. This is a deliberate trade recorded in ADR-020.

**Docker socket access.** `drillback` requires it and is root-equivalent through it.

- *Accepted and documented:* if you can run `drillback`, you can already run `docker`,
  so `drillback` grants no privilege the user did not have. The README says so rather
  than implying a sandbox that does not exist.
- *Mitigated:* `drillback` never mounts the Docker socket **into** a container, so a
  compromised recipe image does not inherit socket access.

**Secrets.** restic passwords and cloud credentials are in the environment.

- *Mitigated:* `drillback` passes the environment to restic and never parses, stores, or
  logs credential values. The JSON report contains the repository *URL* but no
  credentials, and the URL is scrubbed of any `user:password@` userinfo.
- *Mitigated:* recipe `vars` are printed in reports, so recipes must not contain real
  secrets. The Gitea recipe's `db_password` is commented for exactly this reason.
- *NOT IMPLEMENTED:* the entropy warning described here - `recipe validate --strict`
  warning on any var whose name matches `(?i)(secret|token|apikey|api_key|private)` and
  whose value looks random - does not exist. It is SEC-11 in `docs/review/security.md`
  and is in the backlog. Until it does, nothing mechanical stops a contributor pasting
  a real credential into `vars`, and the line above is a convention rather than a
  control.
- *Accepted:* recipe-declared throwaway passwords are visible in reports and in
  `docker inspect`. They protect a database that exists for ninety seconds on an
  internal network. This is fine, and stating it prevents someone "fixing" it later
  with a secrets mechanism nobody needs.

**A hostile backup.** The restored tree is attacker-controlled if the backup is.

- *Mitigated:* symlinks whose targets escape the workspace are neutralised before
  anything is mounted, and reported as warnings (section 4.3). Without this, a backup
  containing `data/config -> /etc` would hand a container a view of the host's `/etc`.
- *Mitigated:* `..` components are rejected on the `dir` source path; restic's
  `--target` restore does not emit them.
- *Mitigated:* inputs are always **copied or restored into** the workspace, never
  bind-mounted from their original location, so nothing a container does can reach the
  user's real data. There is no mode in which `drillback` mounts a live backup directory
  read-write into a container.
- *Accepted:* a very large or zip-bomb-like backup fills the workspace disk. This
  produces a clean exit 2 with the `workspace/no-space` hint, not a wedged machine.
- *Accepted:* file *contents* are not scanned. A malicious SQL dump does whatever SQL
  can do inside a throwaway Postgres container on an internal network.

**Image pulling.** `drillback` pulls images from the internet.

- *Mitigated:* pulls happen on the host, before containers join the internal network,
  so a running recipe container has no egress at all.
- *Mitigated:* `--pull never` allows fully offline operation from a warm image cache.
- *Accepted:* `--pull missing` (the default) means a tag can change under you between
  runs. Digest pinning avoids it and is documented as the hardened option.

**CI.** The round-trip harness runs contributor-authored recipes on GitHub-hosted
runners.

- *Mitigated:* recipe jobs run only on `pull_request` (never `pull_request_target`), so
  a fork PR gets the read-only, secret-less `GITHUB_TOKEN`.
- *Mitigated:* `permissions: contents: read` at workflow level; nothing in the recipe
  job can write to the repository.
- *Mitigated:* GitHub-hosted runners are ephemeral, so a hostile recipe gets a VM that
  is destroyed regardless.
- *Accepted:* a hostile PR can consume Actions minutes and can egress from the runner
  during the image pull. Concurrency limits and the 25-minute job cap bound it; the
  weekly all-recipes run is `workflow_dispatch`/`schedule` only, never triggered by a
  fork.

### 9.4 Explicit non-mitigations

Stated plainly so nobody assumes otherwise:

1. `drillback` is **not** a sandbox for untrusted recipes. Read a recipe before running
   it, exactly as you would read a `docker-compose.yml` before running it.
2. A PASS verdict says *these checks passed on this snapshot*. It is not a guarantee of
   completeness, and a recipe with weak checks yields a weak PASS. The round-trip
   harness raises the floor; it does not set the ceiling.
3. `drillback` does not protect against a compromised Docker daemon, a compromised
   restic binary, or a compromised host.
4. No cryptographic attestation of results in v0.1. The JSON report is evidence for a
   human, not a signed artifact.

---

## 10. Testing pyramid

### 10.1 Unit — fast, hermetic, no Docker

`go test ./...` with no build tags. Target: under 10 seconds on a laptop. These must
never need a daemon, a network, or a fixture larger than a few kilobytes.

| Area | What is tested |
|---|---|
| Schema | Every rule in section 3.4 and 3.5, each with a minimal valid document and a minimal invalid one. Table-driven, one row per constraint. Includes a golden test that every bundled recipe validates `--strict`. |
| Compose safety | The rejection list, one case per forbidden construct, plus the three Go-only rules: YAML tags, unresolved `${}` placeholders, unknown service references. |
| Report | Rendering is a pure function of the report struct. Golden files for PASS, RESTORE UNUSABLE, and the multi-target `--all` shape, in both colour and `NO_COLOR`. The JSON report is round-tripped and validated against its own schema. |
| Hints | Every rule in `hints.yaml` has at least one fixture string it must match and at least one near-miss it must not. Rule ordering is asserted, so adding a broad rule cannot silently shadow a specific one. |
| Snapshot selection | `latest` with and without `--tag`/`--host`, ties broken by time then id, an explicit id, an ambiguous prefix, and a missing snapshot — driven by recorded `restic snapshots --json` output, not a live repository. |
| Input mapping | Recipe default, `drillback.yaml` target, `--input`, and `within:` resolution; precedence between them; `..` rejection; the collision case where two inputs resolve to the same path. |
| Dump detection | `PGDMP` magic bytes vs plain SQL vs an HTML error page vs an empty file. |
| Expect evaluation | Each key in the `expect` vocabulary against matching and non-matching observations. |
| Templating | The restricted context, and that an unknown function or field is an error rather than an empty string. |

### 10.2 Integration — real Docker

`go test -tags integration ./...`. Behind a build tag so `go test ./...` stays hermetic
for someone who just cloned the repo.

- A real `docker compose up` of a two-service fixture stack in `testdata/`, asserting:
  the project name is `drillback-<runid>`, the network is internal, no ports are
  published, and teardown removes everything including volumes.
- Teardown under `SIGINT`, under a panic, and under a `--keep` run followed by manual
  cleanup.
- A real restic repository created in `t.TempDir()`, backed up and restored, including
  a snapshot containing a symlink that escapes the tree — asserting it is neutralised
  and warned about.
- Loading a real plain-SQL dump and a real custom-format dump into a real Postgres
  container, and the failure modes: truncated dump, missing role, wrong version.
- The exit-code contract end to end: a deliberately empty backup gives 1, a missing
  docker binary gives 2.

Each integration test skips with a clear message, not a failure, when Docker is
unavailable — so a contributor without Docker still gets a green `go test ./...`.

### 10.3 Recipe round-trips

`drillback recipe test ./recipes/<name>` per recipe, as specified in section 7. This is
the acceptance test for the domain the project actually lives in. It runs in CI on
changed recipes per PR and on all recipes weekly.

### 10.4 Fresh-clone smoke test

The test that catches everything the others assume. On a clean runner, from nothing:

```sh
git clone https://github.com/spelingbee/drillback && cd drillback
go build ./cmd/drillback          # builds with no network beyond the module cache
./drillback version               # exits 0 with docker/restic absent
./drillback recipe validate ./recipes/*/ --strict
./drillback recipe test ./recipes/uptime-kuma   # the cheapest recipe, end to end
go test ./...                    # green without Docker
```

Run in CI on every push to `main` and on a schedule, in a container that has *neither*
docker nor restic on `PATH` for the first three commands, to prove the error messages
for a missing dependency are the ones we think they are.

### 10.5 What is deliberately not tested

No mocked Docker API. Mocking the thing whose real behaviour is the entire risk surface
would produce a green suite and a broken tool. Docker interaction is either integration
tested against a real daemon or not tested.

---

## 11. CI plan

All workflows: `permissions: contents: read` at the top level, elevated only in the
release job; `concurrency` groups keyed on the ref so a force-push cancels the previous
run; all third-party actions pinned by commit SHA.

### 11.1 `ci.yml` — every push and PR

| Job | Runner | Budget | Does |
|---|---|---|---|
| `lint` | ubuntu-latest | 5m | `gofmt -l`, `go vet`, `golangci-lint`, `english-only`, `yamlfmt --lint` |
| `unit` | ubuntu-latest, macos-latest, windows-latest | 8m | `go test ./... -race -coverprofile` |
| `build` | ubuntu-latest | 8m | `goreleaser build --snapshot --clean` — proves all six targets compile every commit, not just at tag time |
| `integration` | ubuntu-latest | 20m | `go test -tags integration ./... -timeout 18m`. Docker is preinstalled on GitHub-hosted `ubuntu-latest` runners, so this needs no setup step. Skipped on the macOS and Windows runners, which have no Docker daemon. |
| `schema` | ubuntu-latest | 3m | validates every bundled recipe against the schema with an independent validator (`check-jsonschema`), so a bug in our own validator cannot pass itself |

The `english-only` lint flags any non-ASCII character outside a small explicit
allowlist: the box-drawing set used in diagrams and report frames, `—`, `–`, `…`, `·`,
`§`, `→`, `⇒`, `▼`, `✔`, `✘`. Anything else — including a stray accented word or a
non-Latin script — fails the job and names the file and line. The allowlist is a
constant in `scripts/lint-english.sh`, so adding to it is a reviewable diff.

Caching: `actions/setup-go` with `cache: true` for modules and the build cache, keyed on
`go.sum`. Docker images used by integration tests and recipes are **not** cached — a
cache hit would hide a broken image reference, which is a failure worth seeing.

### 11.2 `recipes.yml` — PRs touching `recipes/**`

```yaml
on:
  pull_request:
    paths: ["recipes/**", "internal/**", "schema/**", "docs/hints.yaml"]
```

`pull_request`, never `pull_request_target` (section 9.3). Steps:

1. compute changed recipes (section 7.5);
2. matrix over them, `fail-fast: false`, max-parallel 5;
3. per job, 25-minute timeout: `drillback recipe test ./recipes/<name> --json --report r.json`;
4. upload `r.json` as an artifact and post it as a sticky PR comment, so a contributor
   sees the same report locally and in the PR;
5. a final `recipes-ok` job that depends on the matrix and is the only required status
   check — otherwise a matrix that produces zero jobs cannot satisfy a branch rule.

### 11.3 `smoke.yml` — fresh clone

Section 10.4, on push to `main` and daily. Two jobs: one in a container without docker
or restic on `PATH`, one on a normal runner for the full sequence.

### 11.4 `recipe-health.yml` — weekly

`schedule: cron: "0 5 * * 1"` plus `workflow_dispatch`. Runs the round trip for **every**
bundled recipe against current upstream images. This is the workflow that catches an
upstream image that changed its schema, moved its data directory, or vanished — the
thing that silently rots a recipe registry.

For each failing recipe it opens or updates a single issue titled
`recipe health: <name> failing`, labelled `recipe-health` and `help wanted`, containing
the JSON report and the failing check ids. On the next green run it closes the issue
with a comment. One issue per recipe, updated rather than duplicated, so a recipe broken
for a month is one notification and not thirty.

`permissions: issues: write` for that job only.

### 11.5 `release.yml`

`on: push: tags: ["v*"]`. Runs lint, unit, and build first, then goreleaser.
`permissions: contents: write, packages: write, id-token: write`. Detailed in section 12.

### 11.6 Runtime budget

A full PR touching one recipe costs roughly: lint 2m + unit 3m x 3 runners + build 4m +
integration 12m + schema 1m + one recipe job 12m, mostly in parallel — about 15 minutes
of wall clock and about 40 runner-minutes. That is affordable on the free tier for a
public repository and, more importantly, short enough that a contributor waits for it
rather than leaving.

---

## 12. Release process

### 12.1 Versioning

SemVer, tags `vMAJOR.MINOR.PATCH`. While `0.x`, the CLI surface and the recipe
`apiVersion` may change on minor bumps; the JSON report's `schema_version` and the exit
codes may not.

### 12.2 goreleaser

Targets: `linux`, `darwin`, `windows` x `amd64`, `arm64` — six archives.

- `CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w -X main.version=... -X main.commit=... -X main.date=..."`, giving one reproducible static binary per target.
- SQLite is read through a **pure-Go** driver (`modernc.org/sqlite`), so `CGO_ENABLED=0`
  holds and cross-compilation stays trivial. This constrains the driver choice and is
  recorded in ADR-024.
- `checksums.txt` plus a cosign keyless signature of it, and SBOMs via syft.
- Archives are `.tar.gz` except Windows, which is `.zip`.
- Release notes are generated from Conventional Commits, with a hand-written
  "Highlights" section at the top.

### 12.3 Container image

`ghcr.io/spelingbee/drillback`, tagged `vX.Y.Z`, `vX.Y`, `vX`, and `latest`, multi-arch
`linux/amd64,linux/arm64`.

The image bundles what `drillback` shells out to, so a user can drill without installing
anything: the `drillback` binary, the **docker CLI and the compose v2 plugin** (client
only — the image talks to the host's daemon through a mounted socket), `restic`,
`postgresql-client` (for `psql`/`pg_restore`), and `ca-certificates`. Base:
`debian:bookworm-slim`, because the Postgres client and the compose plugin are simply
more reliable there than on Alpine's musl.

Documented usage, with the socket mount stated as the privilege it is:

```sh
# The socket mount is root-equivalent access to the host's Docker. Read section 9.
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/backups/restic:/repo:ro \
  -v /var/tmp/drillback:/workspace \
  -e RESTIC_REPOSITORY=/repo \
  -e RESTIC_PASSWORD_FILE=/run/secrets/restic \
  ghcr.io/spelingbee/drillback:v0.1.0 check --recipe gitea --workspace /workspace
```

### 12.4 Homebrew tap

`OWNER/homebrew-tap`, formula `drillback`, pushed by goreleaser on tag. `brew install
OWNER/tap/drillback`. The formula declares no dependency on docker or restic — it
`caveats`-warns if they are missing, because on macOS people install Docker Desktop
outside brew and a hard dependency would be wrong.

Submission to homebrew-core is explicitly deferred until the project has an actual user
base; the name `drillback` is available there (section *name-check*), which is worth
preserving but not worth blocking on.

### 12.5 `install.sh`

Served from the repository, `curl -fsSL https://…/install.sh | sh` — with the caveat
that the README shows the two-step download-then-inspect form first, because piping a
script from the internet into a shell is the exact habit this project's audience should
not be encouraged in.

The script:

1. detects OS and architecture, maps them to a goreleaser archive name, and refuses to
   guess on anything unrecognised;
2. resolves the version — `latest` from the GitHub API, or `DRILLBACK_VERSION` if set;
3. downloads the archive **and** `checksums.txt` **and** its cosign signature;
4. verifies the checksum with `sha256sum` or `shasum -a 256`, and **exits non-zero on
   mismatch before extracting anything** — no fallback, no warning-and-continue;
5. verifies the cosign signature if `cosign` is on `PATH`, and prints a clear
   "signature not verified (cosign not installed)" line when it is not, rather than
   silently skipping;
6. installs to `$DRILLBACK_INSTALL_DIR`, else `/usr/local/bin` if writable, else
   `~/.local/bin`, and says which it chose;
7. is `set -euo pipefail`, POSIX `sh`, does no `sudo` of its own, and prints every
   command it runs with `-v`.

`install.sh` itself is covered by the smoke workflow, which runs it against the latest
release on Linux and macOS runners and asserts the checksum-mismatch path fails.

### 12.6 Release checklist

A session or a human must stop and get sign-off before tagging (see
[CLAUDE.md](CLAUDE.md)). The canonical, ordered checklist - including the repository
settings that have to be turned on before any of this is safe - is
[docs/release-checklist.md](docs/release-checklist.md). In summary:

1. `CHANGELOG.md` updated, highlights written by a human;
2. all recipes green on the latest `recipe-health` run;
3. `docs/demo/*.txt` and `docs/demo/demo.gif` regenerated from a real run on this commit;
4. `install.sh` tested against the *previous* release - for v0.1.0 there is no previous
   release, so it is tested against the draft's own assets before the draft is published;
5. version bumped nowhere in source — the version comes from the tag via ldflags only.

---

## 13. Repository layout

```text
drillback/
├── cmd/
│   └── drillback/
│       └── main.go              # wiring only: flags → config → run. No logic.
├── internal/
│   ├── cli/                     # cobra commands, flag parsing, exit-code mapping
│   ├── config/                  # drillback.yaml: load, merge, precedence
│   ├── recipe/                  # types, loader, templating, JSON Schema validation
│   │   └── safety/              # compose safety rules (schema + the three Go rules)
│   ├── source/                  # backup sources
│   │   ├── restic/              # snapshot selection, restore, `restic` process I/O
│   │   └── dir/                 # already-restored trees
│   ├── workspace/               # temp dir lifecycle, sanitisation, teardown registry
│   ├── compose/                 # docker compose v2 invocation, project naming, labels
│   ├── loader/                  # postgres-dump and sqlite input loading
│   ├── probe/                   # ready probes: http, tcp, exec (with retry)
│   ├── check/                   # checks: http, exec, sql, file + expect evaluation
│   ├── report/                  # report struct, TTY renderer, JSON renderer
│   ├── hints/                   # hints.yaml catalog, matching, ordering
│   ├── nudge/                   # URL construction, length fallback
│   └── harness/                 # `recipe test`: stage A, stage B, throwaway restic
├── recipes/
│   ├── gitea/
│   ├── uptime-kuma/
│   └── …                        # one directory per application; the contribution unit
├── schema/
│   ├── recipe.schema.json
│   └── compose-safety.schema.json
├── docs/
│   ├── hints.yaml               # embedded at build time
│   ├── name-check.md
│   ├── recipes.md               # how to write a recipe (the contributor's entry point)
│   ├── security.md              # section 9, expanded, for people who will not read SPEC
│   └── demo/                    # captured output — never hand-written
│       ├── pass.txt
│       └── unusable.txt
├── scripts/
│   ├── capture-demo.sh          # produces docs/demo/*.txt from real runs
│   └── install.sh
├── testdata/                    # fixture stacks, recorded restic JSON, dump samples
├── .github/workflows/           # ci.yml, recipes.yml, smoke.yml, recipe-health.yml, release.yml
├── SPEC.md   DECISIONS.md   PROGRESS.md   CLAUDE.md
├── README.md   CONTRIBUTING.md   CODE_OF_CONDUCT.md   SECURITY.md   CHANGELOG.md
├── LICENSE   .gitignore   .editorconfig   .goreleaser.yaml   Makefile
└── go.mod   go.sum
```

### 13.1 Package boundaries

The rules that keep this from collapsing into one package with an import cycle:

- **`internal/recipe` imports nothing else in `internal/`.** It is types plus
  validation. Everything depends on it; it depends on nothing. If `recipe` ever needs to
  import `compose`, the design is wrong.
- **`internal/report` is a pure function of its input struct.** It does no I/O beyond
  writing to a supplied `io.Writer`, and it never reaches back into `check` or
  `compose`. This is what makes golden-file tests possible.
- **Only `internal/compose` and `internal/source/restic` shell out.** Every other
  package receives already-executed results. This is where the integration tests point,
  and keeping the surface at two packages keeps that boundary small.
- **`internal/workspace` owns every path.** No other package constructs a path inside
  the run directory; they ask for one. This is the mechanism that makes "nothing outside
  the workspace" a structural property rather than a habit.
- **`internal/check` does not know what a recipe is.** It receives a resolved check and
  an execution context. Same for `probe`. This is what lets `harness` reuse them for
  stage A and stage B without a special code path.
- **`internal/cli` is the only package that knows about exit codes**, and the only one
  that writes to stdout/stderr directly.
- **`cmd/drillback/main.go` contains no logic**, so that a future `drillback` used as a
  library, or a second entry point, costs nothing.

---

## 14. Roadmap

### v0.1 — "it boots"

The whole of this document. Ships when: `check` works end to end against restic and
`dir`, `recipe test` works, at least **six** bundled recipes pass round trips (Gitea,
Uptime Kuma, Vaultwarden, Paperless-ngx, Miniflux, Nextcloud), CI runs all of it, and
the install paths work. Six is chosen so the recipe format has been stressed by more
than its two authors before it is frozen.

### v0.2 — "more of everything"

- **`mysql-dump` input kind.** Format detection, charset handling, and the
  `--single-transaction` / `--routines` caveats that make MySQL dumps quietly lossy.
- **borg and kopia sources.** Both behind the same `source` interface restic already
  implements, which is the point of having defined it that way in v0.1.
- **Notifiers.** `ntfy`, Gotify, and healthchecks.io, configured per target in
  `drillback.yaml`. Deliberately thin: they post the verdict and a link to the report.
- **Cron mode with history.** `drillback check --all --history /var/lib/drillback`,
  keeping the last N JSON reports per target and adding `drillback history` to show a
  pass/fail timeline. This is the feature that turns a one-shot answer into a trend, and
  it is the one most likely to make people keep the tool.
- **`drillback doctor`.** One command that checks docker, compose, restic, disk space,
  and repository reachability, and explains each failure. Most exit-2 issues are
  environment issues, and a dedicated command makes them self-service.

### v0.3 — "evidence you can show someone"

- **HTML report.** A single self-contained file per run, suitable for attaching to a
  compliance ticket or emailing to whoever asks whether the backups are tested.
- **"restore verified" badge.** A shields.io endpoint fed by the JSON report, showing
  the date and verdict of the last successful drill. This is the artifact that makes
  other people ask what it is, which is the cheapest recruitment channel the project
  has.
- **Recipe registry beyond the binary.** Only if the bundled set has outgrown a release
  cycle by then — with signing, and with the trust question from section 9 answered
  first, not after.

### Not on the roadmap

Kubernetes. A hosted service. A GUI. Restoring to production. Each is a different
product, and taking any of them on before v0.2 would cost the recipe count that is the
only metric this project has agreed to care about.
