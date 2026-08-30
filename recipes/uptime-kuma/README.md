# Uptime Kuma

Everything Uptime Kuma knows is in one SQLite file. That makes the backup easy and
makes exactly one mistake easy too: copying `kuma.db` while Kuma is running, and
leaving `kuma.db-wal` behind.

## Inputs

| input | what it is | where it usually lives |
|---|---|---|
| `data` | the whole `/app/data` directory | `/srv/uptime-kuma/data` |
| `db` | `kuma.db`, inside `data` | `/srv/uptime-kuma/data/kuma.db` |

`db` is declared `within: data`, so it is not restored twice and
`--input data=/elsewhere` moves both.

## Mapping a typical install

**The docker-compose.yml from Kuma's docs** uses a named volume,
`uptime-kuma:/app/data`:

    restored check --recipe uptime-kuma \
      --input data=/var/lib/docker/volumes/uptime-kuma/_data

**A bind mount** (`./data:/app/data`) is whatever directory that is:

    restored check --recipe uptime-kuma --input data=/opt/uptime-kuma/data

## Back up the WAL file too

Uptime Kuma writes SQLite in WAL mode. The heartbeats from the last few minutes - the
ones that make the dashboard look alive - are usually in `kuma.db-wal`, not in
`kuma.db`. Backing up the whole `/app/data` directory covers it. Backing up only
`kuma.db` gives you a database that opens, passes an integrity check, and has lost the
recent history.

`restored` reports that case as a hint rather than as a mystery: see `docs/hints.yaml`,
rule `sqlite/wal-missing`.

## What the checks prove

| check | would it fail if the backup were empty? |
|---|---|
| `dashboard-renders` | no |
| `db-integrity` | no - an empty database is a valid database |
| `monitors-present` | **yes** |
| `users-present` | **yes** |
| `heartbeats-present` | **yes** - and this is the one that catches a WAL-less copy |
| `api-entry-page` | no |

`heartbeats-present` is the most interesting check in this recipe. Monitors and users
are written rarely and are almost certainly in the main database file; heartbeats are
written constantly and are the rows most likely to be sitting in the WAL when the
backup ran.

## What this recipe does not prove

Uptime Kuma's first-run setup is a socket.io wizard, which is expensive to drive from a
shell. The harness seeds rows into the database directly, and says so in a comment on
the seed step. That exercises the restore path in full and does not exercise Kuma's
own migration path - a distinction worth being honest about, because the recipe would
not notice a schema change that Kuma handles on startup.

## Round trip

    restored recipe test ./recipes/uptime-kuma

Stage A of this recipe reports **PASS-BY-STARTUP-REFUSAL**: given a zero-length
`kuma.db`, Kuma does not start at all rather than starting empty. That is an accepted
outcome, and an honest one - it is itself evidence that the checks depend on the data.

Measured on the development machine: **2m27s**, both stages, of which 94 seconds is
stage A waiting for a Kuma that was never going to come up.
