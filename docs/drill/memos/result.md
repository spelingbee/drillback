# Memos - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the data directory | all of `/var/opt/memos` (3 files, 192.9 KiB) | **PASS** (exit 0, 5 of 5 checks) | [result.txt](result.txt) |
| B - "the database" | `memos_prod.db` alone (1 file, 4.0 KiB) | **FAIL** (`RESTORE UNUSABLE`, exit 1, 2 of 5 checks) | [result-db-only.txt](result-db-only.txt) |

Memos is not doing anything wrong here. Copy the directory and everything comes back.
The finding is that the documentation's own phrase - "back up both the database and any
local assets" - points at a file that, on a live instance, does not contain the data.

## Reading B, in full

```text
  source     restic  .../restic-db-only
  snapshot   0969cf23  2026-08-30 11:38:28  host=drill  tags=[drill]
  inputs     data  /var/opt/memos                  4.0 KiB  1 file
             db    /var/opt/memos/memos_prod.db    4.0 KiB  1 file

  restore    ok          1.7s   2 inputs
  compose    ok         0.95s   1 service
  load db    ok         0.02s   db: sqlite integrity_check, 0 stderr lines
  ready      ok         0.44s   Memos answers /healthz

  CHECKS
  ✔  healthz        The server answers its health endpoint               0.49s
  ✔  db-integrity   The SQLite database passes PRAGMA integrity_check    0.01s
                    → ok
  ✘  users-present  At least one account survived the restore            0.01s
                      query   SELECT count(*) FROM user;
                      expect  scalar_int_min: 1
                      got     0
  ✘  memos-present  The memos survived the restore                       0.01s
                      query   SELECT count(*) FROM memo;
                      expect  scalar_int_min: 1
                      got     0
  ✘  signin-works   The restored account can still sign in               0.59s
                      expect  status: 200

  RESTORE UNUSABLE  2/5 checks  ·  total 5.0s  ·  teardown ok
```

Note what passes. Memos starts. `/healthz` answers 200. The database opens and
`PRAGMA integrity_check` says `ok`. Nothing is corrupt. Every table is there, and every
table is empty - which is exactly the state the tool's `db/tables-empty` hint describes,
and it fired:

> Every table the checks read exists and holds nothing. ... Or the dump carried nothing
> at all and the application rebuilt an empty schema for itself on start, which is what
> an application with automatic migrations does the moment it meets an empty database.

A person restoring this way gets a Memos that looks completely healthy and asks them to
create their first account.

## Root cause

Memos runs SQLite in WAL mode and does not checkpoint on a schedule that matters here.
On the seeded instance, immediately after creating one account and one memo:

```text
-rw-r--r-- 1 kadyr 197609   4096 Aug 30 17:38 memos_prod.db
-rw-r--r-- 1 kadyr 197609  32768 Aug 30 17:38 memos_prod.db-shm
-rw-r--r-- 1 kadyr 197609 160712 Aug 30 17:38 memos_prod.db-wal
```

4 KiB in the file named "the database"; 160 KiB in the file next to it. Everything that
had ever been written to this instance - the schema migrations, the account, the memo -
was in `-wal`. Copying `memos_prod.db` and nothing else copies an empty database, and
SQLite opens it without complaint because it *is* a valid empty database.

This is not specific to a fresh instance: a busy Memos checkpoints when the WAL grows
past its threshold, so the split changes but never disappears. Any copy of the `.db`
alone is missing everything written since the last checkpoint.

## Reading A works, and this is why the recipe declares both

[`recipes/memos`](../../../recipes/memos) declares the data directory as one input and
`memos_prod.db` as a second input `within` it. That second declaration is what turns
"the -wal was left behind" into a named failed check instead of an instance that
mysteriously forgot everybody.

## What the documentation would need to say instead

**Corrected 2026-09-04:** most of this already exists, on a page the drill missed -
*Backup & Restore* under Operations (see [docs.md](docs.md)). It says to stop and copy
the whole directory or to use `sqlite3 .backup`. What the deploy pages would need is a
link to it, and the page's "What to back up: the database itself" would need half a
sentence:

> Back up the whole `/var/opt/memos` directory. If you copy the SQLite database on its
> own, copy `memos_prod.db-wal` and `memos_prod.db-shm` with it, stop Memos first, or
> use `sqlite3 .backup` - otherwise the copy will be missing everything written since
> the last checkpoint.

The Backup & Restore page also has the restore direction for MySQL and PostgreSQL and,
for SQLite, the stop-copy-start sequence covers it.

## Not tested

- **Attachments on the local filesystem.** The seeded memo has no attachment, so the
  `attachment` table and the assets beside the database were not exercised. Reading A
  copies them by construction; reading B would miss them for the same reason it misses
  everything else.
- **MySQL and PostgreSQL drivers.** Only the documented SQLite default was tested.

## Reproduced

Both verdicts twice, from an empty scratch directory each time, by running `run.sh` end
to end (2026-08-30 11:37 UTC and 11:39 UTC). Identical checks failed both times.
