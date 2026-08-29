# restored

**Your backup is a lie until it boots.**

`restored` restores a backup into a throwaway, isolated environment, starts the
application with `docker compose`, and tells you whether the data actually came back.
One command, about a minute, and an exit code a cron job can act on.

> Pre-release. `restored check` works end to end against restic and against an
> already-restored tree, with two bundled recipes. See [PROGRESS.md](PROGRESS.md) for
> what is not built yet.

---

## What it looks like

A Gitea backup that is fine:

<!-- BEGIN docs/demo/pass.txt -->
```text
restored 0.1.0-dev · recipe gitea · run rjsaa3v3

  source     restic  C:/Users/kadyr/AppData/Local/Temp/restored-demo/repo
  snapshot   512f0707  2026-08-29 20:26:47  host=demo-host  tags=[gitea]
  inputs     data  /srv/gitea/data    102.2 KiB  54 files
             db    /srv/gitea/db.sql  231.1 KiB  plain SQL

  restore    ok          1.9s   2 inputs
  compose    ok          2.0s   2 services, db first for the dump
  load db    ok          4.3s   db: psql, 0 stderr lines
  ready      ok          6.9s   postgres accepts connections, gitea answers on the internal network

  CHECKS
  ✔  web-ui-renders      The web UI renders the instance home page       0.86s
  ✔  repos-in-db         The database contains at least one repository   0.73s
                         row → 1
  ✔  users-in-db         The database contains at least one real user    0.69s
                         account → 1
  ✔  repo-files-on-disk  At least one bare repository exists on disk     0.00s
                         → 1 match for */*.git/HEAD
  ✔  api-lists-repos     The API lists repositories, so the database     0.80s
                         and the disk agree → 1 item

  PASS  5/5 checks  ·  total 19.0s  ·  teardown ok

This backup boots.
```
<!-- END docs/demo/pass.txt -->

The same backup, taken by a cron line whose `pg_dump` names the wrong database. It
runs every night, it exits 0, and it produces a file:

<!-- BEGIN docs/demo/fail.txt -->
```text
restored 0.1.0-dev · recipe gitea · run en5bklxe

  source     restic  C:/Users/kadyr/AppData/Local/Temp/restored-demo/repo
  snapshot   c06074e4  2026-08-29 20:27:45  host=demo-host  tags=[gitea-broken]
  inputs     data  /srv/gitea/data    102.2 KiB  54 files
             db    /srv/gitea/db.sql      489 B  plain SQL

  restore    ok          1.8s   2 inputs
  compose    ok          2.0s   2 services, db first for the dump
  load db    ok          3.0s   db: psql, 0 stderr lines
  ready      ok          6.7s   postgres accepts connections, gitea answers on the internal network

  CHECKS
  ✔  web-ui-renders      The web UI renders the instance home page       0.91s
  ✘  repos-in-db         The database contains at least one repository   0.75s
                         row
                           query   SELECT count(*) FROM repository;
                           expect  scalar_int_min: 1
                           got     0
  ✘  users-in-db         The database contains at least one real user    0.70s
                         account
                           query   SELECT count(*) FROM "user" WHERE lower_name <> 'ghost';
                           expect  scalar_int_min: 1
                           got     0
  ✔  repo-files-on-disk  At least one bare repository exists on disk     0.00s
                         → 1 match for */*.git/HEAD
  ✘  api-lists-repos     The API lists repositories, so the database     0.84s
                         and the disk agree
                           expect  json_path_len_min: 1
                           got     0

  RESTORE UNUSABLE  2/5 checks  ·  total 18.6s  ·  teardown ok

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

## Why this exists

A green backup job is not evidence. The dump taken with `--schema-only`, the SQLite
file copied while the application was writing to it, the bind mount that was empty
because the data was in a named volume — all of them are silent until the day you need
the restore, and then you find out under pressure. The only evidence is a restore that
boots, and doing that by hand is expensive enough that nobody does it.

---

## Quick start

```sh
export RESTIC_REPOSITORY=/mnt/backups/restic
export RESTIC_PASSWORD_FILE=/etc/restic/pass
restored recipe show gitea --inputs-only          # which paths does this recipe want?
restored check --recipe gitea --input data=/srv/gitea --input db=/srv/dumps/gitea.sql
echo $?                                           # 0 pass, 1 unusable, 2 tool error
```

Exit codes are the contract:

| code | meaning | what a cron job should do |
|---|---|---|
| 0 | every check passed | nothing |
| 1 | **restore unusable** — this backup will not save you | page someone |
| 2 | tool error: docker missing, restic failed, recipe invalid | fix the runner, then re-run; **do not** treat as a pass |

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

Bundled so far: `gitea` (PostgreSQL) and `uptime-kuma` (SQLite).

---

## Add a recipe in 10 minutes

```sh
restored recipe init paperless --db postgres-dump --with-dir media
restored recipe validate ./recipes/paperless --strict
```

The number of distinct external contributors with merged PRs is the only metric this
project has agreed to care about, and a recipe is the unit of contribution. The full
guide — what makes a check data-sensitive, how the round-trip harness proves your
recipe both ways, and what CI will run on your PR — is going into `CONTRIBUTING.md`
next. Until it lands, [SPEC.md](SPEC.md) section 3 is the field reference and
[`recipes/gitea/`](recipes/gitea/) is the worked example.

The other useful contribution needs no Go and no Docker: a rule in
[`docs/hints.yaml`](docs/hints.yaml). If you hit a confusing restore failure, the fix is
often twenty lines in that file.

---

## Roadmap

- **v0.1, "it boots".** `restored check` against restic and `dir`, the round-trip
  harness, six bundled recipes, CI, and the install paths.
- **v0.2, "more of everything".** MySQL dumps, borg and kopia sources, notifiers, cron
  mode with history, `restored doctor`.
- **v0.3, "evidence you can show someone".** A self-contained HTML report, and a
  "restore verified" badge fed by the JSON report.

Not on the roadmap: Kubernetes, a hosted service, a GUI, and restoring to production.
`restored` verifies; a human restores.

---

## License

Apache-2.0. See [LICENSE](LICENSE).
