# Navidrome - result

| Reading | What was restored, and how | Verdict | Report |
|---|---|---|---|
| A - the documented commands | `navidrome backup create`, then `navidrome backup restore` on a new instance | **FAIL** (exit 1, 1 of 4 checks) | [result.txt](result.txt) |
| A' - the same, with the two undocumented conditions met | a database created first, `ND_BACKUP_PATH` set, `--backup-file` as a name inside it | **FAIL** (exit 1, 1 of 4 checks) - the command reports `Restore complete` and the instance is empty | [result-bootstrapped.txt](result-bootstrapped.txt) |
| Control - a copy of `/data` | the data folder, put back and started | **PASS** (exit 0, 6 of 6 checks) | [result-data.txt](result-data.txt) |

The data is recoverable. The documented way of recovering it is not the way that works.

## Reading A: the command will not start

```text
+ /app/navidrome backup restore --datafolder /data --musicfolder /music \
    --backup-file /backup/navidrome_backup_2026.08.30_12.26.34.db --force
time="..." level=fatal msg="No existing database" path=/data/navidrome.db
```

`navidrome backup restore` refuses to run unless a database is already there. On the
day you need it, there is not one: that is what restoring onto a new machine means. The
restore section of the backup page does not mention this, and its two warnings both
point the other way - *do not have Navidrome running* - which makes the missing step
easy to reason past.

Verdict from the tool, with the empty instance that results:

```text
  CHECKS
  ok  serves-ui              The web interface answers
  X   signin-works           The restored account can still sign in     401
  X   playlist-restored      The playlist is back, under its owner
  X   library-rows-restored  The library catalogue is back

  RESTORE UNUSABLE  1/4 checks
```

## Reading A': `Restore complete`, and nothing is restored

The second recipe adds the two steps the documentation does not have. Start Navidrome
once so it creates a database, stop it (the page is emphatic about that), set
`ND_BACKUP_PATH`, and pass `--backup-file` as a **file name** rather than a path -
because with an absolute path the command fails with an empty path in its own message:

```text
$ navidrome backup restore --datafolder /data -b /backup/navidrome_backup_....db -f
level=fatal msg="Error restoring database" backup path= error="getting backup
  connection: unable to open database file: no such file or directory"

$ ND_BACKUP_PATH=/backup navidrome backup restore --datafolder /data \
    -b navidrome_backup_....db -f
level=info msg="Restore complete" elapsed=5.9ms
```

With all of that satisfied the command succeeds, and the instance is still empty:

```text
  RESTORE UNUSABLE  1/4 checks
  X  signin-works  expect status: 200, got 401
                   got {"error":"Invalid username or password"}
```

### The evidence that this is real, and not a measurement mistake

The clearest test is the application's own, because it needs no assumptions about
SQLite files. Navidrome offers `POST /auth/createAdmin` **only when it has no users at
all**. On a stack whose restore had just reported `Restore complete`:

```text
--- createAdmin after the restore (200 means the user table is empty):
200
--- sign in as the backed-up account (200 means the restore worked):
401
```

Navidrome is offering to create the first administrator. The account that was in the
backup cannot sign in.

And the backup file itself is good - the same file, read directly, has everything in it:

```text
$ sqlite3 navidrome_backup_2026.08.30_12.26.34.db \
    "select user_name from user; select count(*) from playlist; select count(*) from media_file;"
drilladmin
1
1
```

So: a valid backup, a command that says it restored it, and an empty instance.

One measurement note, recorded because it caught this drill out first: reading a live
Navidrome database by copying `navidrome.db`, `-wal` **and** `-shm` gives nonsense - a
stale `-shm` makes SQLite report `page_count 1` and no tables. Copy the database and its
`-wal` and leave the `-shm` behind, or ask the running application instead. The
application-level probe above is what the conclusion rests on.

## The control: the data is fine

A plain copy of `/data`, put back and started:

```text
  PASS  6/6 checks
```

users, playlists, the library catalogue, `PRAGMA integrity_check`, and a sign-in with
the backed-up password. [`recipes/navidrome`](../../../recipes/navidrome) is that
recipe, and it passes both stages of `restored recipe test`.

Worth naming, because it is a recipe decision other music-server recipes will need:
scanning is switched **off** in that recipe (`ND_SCANSCHEDULE: "0"`,
`ND_SCANONSTARTUP: "false"`). A restored database describes music that a throwaway
stack does not have, and a scan reconciles the two by deleting the rows - so a restore
check that lets the application scan first is measuring what the application made of the
backup, not what was in it.

## What the documentation would need to say instead

The restore section is three sentences. It needs four more:

> Restoring requires a database to already exist. On a new machine, start Navidrome
> once so that it creates one, stop it, and then restore.
>
> Pass the backup with `--backup-file`. The value is a file name inside `Backup.Path`,
> not a path, so `Backup.Path` (or `ND_BACKUP_PATH`) must be configured for the restore
> as well as for the backup. Add `--force` to skip the confirmation prompt.

And then the behaviour behind reading A' needs fixing, because no wording makes
`Restore complete` followed by an empty database acceptable.

## Not tested

- **Non-Docker installs.** Only the official image was tested. The `No existing
  database` refusal and the `--backup-file` resolution are in the binary and are very
  unlikely to differ; the empty-database outcome may.
- **PostgreSQL.** Navidrome's SQLite default is what was tested.
- **Play counts and ratings.** The seeded instance has one track, one playlist and one
  user. Annotations were not exercised, so nothing here is claimed about them.

## Reproduced

All three verdicts twice, from an empty scratch directory each time, by running `run.sh`
end to end (2026-08-30 12:26 UTC and 12:33 UTC). Reading A and reading A' both
`RESTORE UNUSABLE 1/4`; the control `PASS 6/6`. The `createAdmin` probe was run
separately, twice.
