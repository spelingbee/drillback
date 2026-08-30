# Vaultwarden

A password manager whose backup does not restore is the worst case this tool exists
for, and Vaultwarden's backup is unusually easy to get wrong: SQLite in WAL mode means
the bytes you need are in three files, not one.

## Inputs

| input | what it is | where it usually lives |
|---|---|---|
| `data` | the whole `/data` directory | `/srv/vaultwarden/data` |
| `db` | `db.sqlite3`, inside `data` | `/srv/vaultwarden/data/db.sqlite3` |

`db` is declared `within: data`, so it is not restored twice and
`--input data=/elsewhere` moves both.

## Mapping a typical install

**The docker-compose.yml from the Vaultwarden wiki** mounts `./vw-data` at `/data`.
If your compose file lives in `/opt/vaultwarden`, the directory to back up is
`/opt/vaultwarden/vw-data`:

    drillback check --recipe vaultwarden --input data=/opt/vaultwarden/vw-data

**A named volume** (`vaultwarden_data:/data`) puts it under
`/var/lib/docker/volumes/vaultwarden_data/_data`. Backing that path up directly works,
and so does this:

    drillback check --recipe vaultwarden --input data=/var/lib/docker/volumes/vaultwarden_data/_data

**A bare-metal install** with `DATA_FOLDER` set puts it wherever that says.

Whatever the path, the two inputs move together:

    drillback check --recipe vaultwarden \
      --input data=/mnt/backup/vaultwarden \
      --input db=/mnt/backup/vaultwarden/db.sqlite3

## Back up all three SQLite files, or none of them

Vaultwarden runs SQLite in WAL mode. At any moment the newest writes are in
`db.sqlite3-wal`, not in `db.sqlite3`. A backup that copies only `db.sqlite3` restores
a database that is intact, passes `PRAGMA integrity_check`, and is silently missing
everything since the last checkpoint - which for a busy instance can be days.

Backing up the whole `/data` directory covers it, because `-wal` and `-shm` are in
there. If your backup enumerates files, add both. Better still, use Vaultwarden's own
backup command, which takes a consistent copy:

    sqlite3 /data/db.sqlite3 ".backup '/data/backup/db.sqlite3'"

`drillback` reports a missing `-wal` as a hint, not a guess: see `docs/hints.yaml`,
rule `sqlite/wal-missing`.

## What the checks prove

| check | would it fail if the backup were empty? |
|---|---|
| `web-vault-renders` | no - Vaultwarden serves the same web vault against an empty database |
| `db-integrity` | no - an empty database is a valid database |
| `accounts-present` | **yes** |
| `account-keys-present` | no, but it catches an account restored without the key that decrypts its vault |
| `rsa-key-on-disk` | no, but it catches a backup that took the database and not the directory |

`accounts-present` is the check that makes this recipe worth anything.
`account-keys-present` and `rsa-key-on-disk` are the two failures that are invisible
from the web vault: an account with no `akey` is an account nobody can log in to, and
a missing `rsa_key.pem` invalidates every token every client is holding.

## Round trip

    drillback recipe test ./recipes/vaultwarden

The harness registers an account through `POST /api/accounts/register`, which is the
request the web vault itself makes. The key material it sends is nonsense, because the
server stores it and never inspects it - which is the point of Bitwarden's design and
is why this can be seeded through the front door at all.

Measured on the development machine: **28 seconds**, both stages.
