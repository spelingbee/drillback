# restored

[![ci](https://github.com/spelingbee/restored/actions/workflows/ci.yml/badge.svg)](https://github.com/spelingbee/restored/actions/workflows/ci.yml)
[![recipes](https://github.com/spelingbee/restored/actions/workflows/recipes.yml/badge.svg)](https://github.com/spelingbee/restored/actions/workflows/recipes.yml)
[![recipe-health](https://github.com/spelingbee/restored/actions/workflows/recipe-health.yml/badge.svg)](https://github.com/spelingbee/restored/actions/workflows/recipe-health.yml)
[![license](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

**Your backup is a lie until it boots.**

`restored` restores a backup into a throwaway, isolated environment, starts the
application with `docker compose`, and tells you whether the data actually came back.
One command, about a minute, and an exit code a cron job can act on.

![restored proving a Gitea backup, and then failing one](docs/demo/demo.gif)

*A real recording, played at double speed. It stands up Gitea and PostgreSQL, seeds
them, backs them up with restic, destroys the stack, and restores it - twice: once from
a good backup, once from a backup whose database dump is empty. The two commands are
typed by [`docs/demo/demo.tape`](docs/demo/demo.tape); everything underneath them is
what the tool printed, unedited, and it is the same `scripts/demo.sh` you can run
yourself.*

> Pre-release, and not tagged. `restored check` works end to end against restic and
> against an already-restored tree; `restored recipe test` runs the round trip that
> proves a recipe both ways; five recipes ship. See [PROGRESS.md](PROGRESS.md) for what
> is not built yet.

---

## Why this exists

A green backup job is not evidence. The dump taken with `--schema-only`, the SQLite
file copied while the application was writing to it, the bind mount that was empty
because the data was in a named volume — all of them are silent until the day you need
the restore, and then you find out under pressure. The only evidence is a restore that
boots, and doing that by hand is expensive enough that nobody does it.

---

## Install

```sh
# any Linux or macOS, verifies the checksum, installs to ~/.local/bin
curl -fsSL https://raw.githubusercontent.com/spelingbee/restored/main/install.sh | sh

# macOS, via Homebrew
brew install spelingbee/tap/restored

# anywhere Go is installed
go install github.com/spelingbee/restored/cmd/restored@latest
```

Or download an archive from [Releases](https://github.com/spelingbee/restored/releases)
and unpack it: `checksums.txt` covers every one, and an SBOM ships beside each.

For a NAS, there is an image with `restored`, the Docker CLI, the Compose plugin and
`restic` already in it. Read [docs/docker.md](docs/docker.md) first - it needs the
Docker socket, and it is blunt about what that grants.

```sh
docker run --rm ghcr.io/spelingbee/restored:0.1.0 version
```

**What you need:** a Docker daemon, and `restic` if your backups are restic
repositories. `restored version` tells you what it can see:

```sh
restored version      # exits 0 even when docker and restic are both missing
```

---

## See it work, without a backup of your own

This is the fastest way to find out whether the tool does what this page says, and it
needs nothing but Docker and restic:

```sh
git clone https://github.com/spelingbee/restored && cd restored
./scripts/demo.sh          # stands up a real Gitea, backs it up, restores it: PASS, exit 0
./scripts/demo-broken.sh   # the same backup with an empty dump: RESTORE UNUSABLE, exit 1
```

Both build a real application, put real data in it, back it up with restic, destroy it,
and restore it. About a minute each. The second one is the more useful of the two: it
is what a backup that passes its own backup job and fails a restore looks like.

---

## What it looks like

A Gitea backup that is fine:

<!-- BEGIN docs/demo/pass.txt -->
```text
restored 0.1.0-dev · recipe gitea · run 72ixb2f2

  source     restic  C:/Users/kadyr/AppData/Local/Temp/restored-demo/repo
  snapshot   21a01801  2026-08-30 03:55:56  host=demo-host  tags=[gitea]
  inputs     data  /srv/gitea/data    102.2 KiB  54 files
             db    /srv/gitea/db.sql  231.1 KiB  plain SQL

  restore    ok          1.8s   2 inputs
  compose    ok          1.2s   2 services, db first for the dump
  load db    ok          3.0s   db: psql, 0 stderr lines
  ready      ok          5.4s   postgres accepts connections, gitea answers on the internal network

  CHECKS
  ✔  web-ui-renders      The web UI renders the instance home page       0.53s
  ✔  repos-in-db         The database contains at least one repository   0.44s
                         row → 1
  ✔  users-in-db         The database contains at least one real user    0.45s
                         account → 1
  ✔  repo-files-on-disk  At least one bare repository exists on disk     0.00s
                         → 1 match for */*.git/HEAD
  ✔  api-lists-repos     The API lists repositories, so the database     0.53s
                         and the disk agree → 1 item

  PASS  5/5 checks  ·  total 13.8s  ·  teardown ok

This backup boots.
```
<!-- END docs/demo/pass.txt -->

The same backup, taken by a cron line whose `pg_dump` names the wrong database. It
runs every night, it exits 0, and it produces a file:

<!-- BEGIN docs/demo/fail.txt -->
```text
restored 0.1.0-dev · recipe gitea · run hfnneq6j

  source     restic  C:/Users/kadyr/AppData/Local/Temp/restored-demo/repo
  snapshot   782c24b1  2026-08-30 03:56:39  host=demo-host  tags=[gitea-broken]
  inputs     data  /srv/gitea/data    102.2 KiB  54 files
             db    /srv/gitea/db.sql      489 B  plain SQL

  restore    ok          1.8s   2 inputs
  compose    ok          1.2s   2 services, db first for the dump
  load db    ok          2.3s   db: psql, 0 stderr lines
  ready      ok          5.3s   postgres accepts connections, gitea answers on the internal network

  CHECKS
  ✔  web-ui-renders      The web UI renders the instance home page       0.53s
  ✘  repos-in-db         The database contains at least one repository   0.43s
                         row
                           query   SELECT count(*) FROM repository;
                           expect  scalar_int_min: 1
                           got     0
  ✘  users-in-db         The database contains at least one real user    0.43s
                         account
                           query   SELECT count(*) FROM "user" WHERE lower_name <> 'ghost';
                           expect  scalar_int_min: 1
                           got     0
  ✔  repo-files-on-disk  At least one bare repository exists on disk     0.00s
                         → 1 match for */*.git/HEAD
  ✘  api-lists-repos     The API lists repositories, so the database     0.54s
                         and the disk agree
                           expect  json_path_len_min: 1
                           got     0

  RESTORE UNUSABLE  2/5 checks  ·  total 13.9s  ·  teardown ok

  LIKELY CAUSE
    The application's tables are there, but they are empty

    Every table the checks read exists and holds nothing. Two causes produce
    exactly this. Either the dump was taken from the wrong database, or with
    `pg_dump --schema-only`, or narrowed with `--table`, so it carried a
    schema and none of the rows. Or the dump carried nothing at all and the
    application rebuilt an empty schema for itself on start, which is what
    an application with automatic migrations does the moment it meets an
    empty database. Compare the size of the dump with the size of the data
    directory in the report above: a forge with repositories on disk and a
    half-kilobyte dump is not a backup.

      grep -c 'INSERT INTO' /srv/gitea/db.sql
      ls -l /srv/gitea/db.sql
                                                      (hint: db/tables-empty)

  Service logs from the failure window are in the JSON report (--report).
  Re-run with --keep to keep the stack up and poke at it yourself.
```
<!-- END docs/demo/fail.txt -->

Both blocks are captured from real runs by
[`scripts/capture-demo.sh`](scripts/capture-demo.sh). Nothing in this README is a mock.
Reproduce them yourself:

```sh
make build
./scripts/demo.sh          # PASS, exit 0
./scripts/demo-broken.sh   # RESTORE UNUSABLE, exit 1
```

Each script stands up a real Gitea and PostgreSQL, seeds a user, a repository and a
commit, backs the data up with restic, destroys the stack, and hands the backup to
`restored`. They need docker and restic, they clean up after themselves, and they can
be run twice in a row.

---

## Quick start, against your own backup

```sh
export RESTIC_REPOSITORY=/mnt/backups/restic
export RESTIC_PASSWORD_FILE=/etc/restic/pass        # your restic password file

restored recipe show gitea --inputs-only            # which paths does this recipe want?
# data  dir            required  /srv/gitea/data
# db    postgres-dump  required  /srv/gitea/db.sql

restored check --recipe gitea                       # if your paths match the two above
echo $?                                             # 0 pass, 1 unusable, 2 tool error
```

If your layout differs - and it usually does, because those defaults are the recipe
author's machine - point each input at your path:

```sh
restored check --recipe gitea   --input data=/var/lib/gitea   --input db=/srv/dumps/gitea.sql
```

`restic ls latest | head -50` shows what your snapshot actually contains, which is the
fastest way to fill those in. If you get it wrong, the error says so and names the flag.

Exit codes are the contract:

| code | meaning | what a cron job should do |
|---|---|---|
| 0 | every check passed | nothing |
| 1 | **restore unusable** - this backup will not save you | page someone |
| 2 | tool error: docker missing, restic failed, recipe invalid, out of time | fix the runner, then re-run; **do not** treat as a pass |

Exit 2 says nothing about your backup. Only 0 and 1 are verdicts.

---

## How a recipe works

A recipe is about sixty lines of YAML. It declares the *logical inputs* an application
needs — not your paths — plus the probes that say the app is up and the checks that say
the data survived.

```yaml
inputs:
  data:
    kind: dir
    title: Gitea data directory
    default_path: /srv/gitea/data      # a guess; you override it with --input
    mount:
      env: RESTORED_INPUT_data         # compose.yaml refers to ${RESTORED_INPUT_data}
      into: gitea:/data

  db:
    kind: postgres-dump
    title: Gitea database dump
    default_path: /srv/gitea/db.sql
    load:
      service: db
      database: "{{ .vars.db_name }}"
      user: "{{ .vars.db_user }}"

checks:
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
```

The check above is the one that matters. It is *data-sensitive*: it fails against an
empty database. A recipe whose checks all pass against an empty stack is worse than no
recipe, because it manufactures the confidence this tool exists to destroy.

The `expect` vocabulary is closed and small on purpose — `status`, `body_matches`,
`json_path_len_min`, `scalar_int_min`, `glob_min_count` and a dozen more. There is no
expression language: a recipe is data, not a program.

Every run is isolated, and the isolation is enforced by a schema rather than by
convention. No `privileged`, no host networking, no host PID or IPC namespace, no
published ports, no bind mount outside the run's own workspace. HTTP checks run from a
helper container attached to the run's internal network. `restored recipe validate`
rejects a recipe that breaks any of it.

---

## Bundled recipes

<!-- BEGIN recipes-table -->

| recipe | application | state it restores | checks |
|---|---|---|---|
| [`beszel`](recipes/beszel/) | Beszel (PocketBase / SQLite) | directories + SQLite | 5 |
| [`changedetection`](recipes/changedetection/) | changedetection.io (JSON datastore) | directories | 3 |
| [`filebrowser`](recipes/filebrowser/) | File Browser (Bolt database) | directories | 4 |
| [`freshrss`](recipes/freshrss/) | FreshRSS (SQLite) | directories + SQLite | 5 |
| [`gitea`](recipes/gitea/) | Gitea + PostgreSQL | directories + PostgreSQL | 5 |
| [`gogs`](recipes/gogs/) | Gogs (SQLite) | directories + SQLite | 7 |
| [`gotify`](recipes/gotify/) | Gotify (SQLite) | directories + SQLite | 6 |
| [`listmonk`](recipes/listmonk/) | listmonk (PostgreSQL) | PostgreSQL + directories | 6 |
| [`mealie`](recipes/mealie/) | Mealie (SQLite) | directories + SQLite | 5 |
| [`memos`](recipes/memos/) | Memos (SQLite) | directories + SQLite | 5 |
| [`n8n`](recipes/n8n/) | n8n (SQLite) | directories + SQLite | 6 |
| [`navidrome`](recipes/navidrome/) | Navidrome (SQLite) | directories + SQLite | 6 |
| [`nextcloud`](recipes/nextcloud/) | Nextcloud (PostgreSQL) | directories + PostgreSQL | 6 |
| [`open-webui`](recipes/open-webui/) | Open WebUI (SQLite) | directories + SQLite | 5 |
| [`paperless-ngx`](recipes/paperless-ngx/) | Paperless-ngx (PostgreSQL + Redis) | directories + PostgreSQL | 5 |
| [`trilium`](recipes/trilium/) | Trilium Notes (SQLite) | directories + SQLite | 4 |
| [`uptime-kuma`](recipes/uptime-kuma/) | Uptime Kuma (SQLite) | directories + SQLite | 6 |
| [`vaultwarden`](recipes/vaultwarden/) | Vaultwarden (SQLite) | directories + SQLite | 5 |

<!-- END recipes-table -->

Each directory has a README saying which of *your* directories each input is, for the
two or three ways that application is usually deployed. The full index, and the
[field reference generated from the JSON Schema](docs/recipe-spec.md), are in
[`recipes/`](recipes/README.md).

Every one of them has passed the round trip described below. Nothing is on that list
because somebody thought it looked right.

---

## Add a recipe in 10 minutes

The number of distinct external contributors with merged pull requests is the only
metric this project has agreed to care about, and a recipe is the unit of
contribution. Two YAML files, no Go.

If you already have a `docker-compose.yml` for the application, the first draft is one
command:

```sh
restored recipe init myapp --compose ~/docker/myapp/docker-compose.yml
restored recipe test ./recipes/myapp        # this is exactly what CI runs
```

`recipe init --compose` turns your volumes into inputs, recognises a PostgreSQL or
SQLite service, takes the container side of your published port for the ready probe,
writes the `healthcheck` and `depends_on` that stop your application racing its own
database, and leaves a TODO everywhere the answer is yours.

*Ten minutes is a measured number, not a slogan.* Before the first public release a
reviewer who had never seen this codebase wrote a recipe for an application that is not
in the registry, using only these documents and the scaffold, and got it passing both
stages in **13 minutes 26 seconds** - of which about seven were the scaffold's fault and
have since been fixed. Their walk, with every wrong turn in it, is in
[`docs/review/fresh-clone.md`](docs/review/fresh-clone.md). If yours takes much longer
than ten minutes, that is a bug in this project and worth an issue.

`recipe test` is the whole review process, mechanised:

- **Stage A** starts your stack with **empty** inputs and requires that at least one
  check **fails**. A recipe whose checks all pass against an empty database is rejected
  with `recipe has no data-sensitive check`.
- **Stage B** starts a fresh stack, seeds it, exports what a backup would have taken,
  puts that into a throwaway restic repository, tears everything down, and then runs an
  ordinary `restored check` against it. Every check must pass.

Stage B ends by running the command a user runs. There is no test-only restore path, so
the harness cannot pass while the real one is broken — which is why a maintainer can
merge your recipe without understanding your application.

**[CONTRIBUTING.md](CONTRIBUTING.md#add-a-recipe-in-10-minutes) is the full walk-through**,
including the review promise: a first response within 24 hours, and a merge within 48
when CI is green.

The other useful contribution needs no Go and no Docker: a rule in
[`docs/hints.yaml`](docs/hints.yaml). If you hit a confusing restore failure, the fix is
often twenty lines in that file.

---

## Contributors

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

None yet — this repository has never been public. The list above follows the
[all-contributors](https://allcontributors.org) specification and is configured in
[`.all-contributorsrc`](.all-contributorsrc), using that specification's built-in
contribution types; the bot that maintains it is not installed, because installing an
app is one of the stop points in [CLAUDE.md](CLAUDE.md).

`scripts/contributors.sh` prints the number this project is actually trying to move:
distinct people, other than the owner and other than a bot, with a pull request merged
in the trailing 365 days.

---

## Roadmap

- **v0.1, "it boots".** `restored check` against restic and `dir`, the round-trip
  harness, five bundled recipes, CI, and the install paths. A sixth - the one you
  write - is the point.
- **v0.2, "more of everything".** MySQL dumps, borg and kopia sources, notifiers, cron
  mode with history, `restored doctor`.
- **v0.3, "evidence you can show someone".** A self-contained HTML report, and a
  "restore verified" badge fed by the JSON report.

Not on the roadmap: Kubernetes, a hosted service, a GUI, and restoring to production.
`restored` verifies; a human restores.

---

## License

Apache-2.0. See [LICENSE](LICENSE).
