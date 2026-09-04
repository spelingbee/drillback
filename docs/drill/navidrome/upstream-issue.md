# Draft issue for Navidrome - filed 2026-09-04

**Status: filed on 2026-09-04, with the human's sign-off (stop point 2).** Issue 1 is <https://github.com/navidrome/navidrome/issues/6083>, in the shape of the repository's bug form; issue 2 is <https://github.com/navidrome/website/issues/436>, on the documentation repository, and links the first.

Where it would go: <https://github.com/navidrome/navidrome/issues>. A human should
search for an existing issue about `backup restore` before opening this; the reproduction
below can be added to one instead.

Two issues are proposed. The first is a bug and the second is documentation, and the
first is the one that matters.

---

## Issue 1 (bug) - `backup restore` reports "Restore complete" and leaves an empty database

**Title:** `backup restore reports "Restore complete" but the instance comes up empty (0.63.2, Docker)`

**Environment**

- Navidrome 0.63.2, `deluan/navidrome:0.63.2`, Docker, SQLite, `/data` on a volume.
- Backup taken with `docker compose run navidrome backup create`, as the backup page
  documents.

**Steps**

1. Fresh instance. Create the admin through `/auth/createAdmin`, let a one-track library
   scan, create a playlist.
2. `docker compose run navidrome backup create` -> a 684 KB
   `navidrome_backup_<timestamp>.db`.
3. Destroy the instance and its volume.
4. New instance. Start it once so a database exists, stop it, then:

   ```sh
   ND_BACKUP_PATH=/backup navidrome backup restore --datafolder /data \
     -b navidrome_backup_2026.08.30_12.26.34.db --force
   ```

   ```text
   level=info msg="Restore complete" elapsed=5.9ms
   ```

5. Start Navidrome.

**Observed**

```text
POST /auth/createAdmin  -> 200   (Navidrome only offers this when there are no users)
POST /auth/login as the account that was in the backup -> 401
```

The instance is empty. The backup file is not - reading it directly:

```text
$ sqlite3 navidrome_backup_2026.08.30_12.26.34.db \
    "select user_name from user; select count(*) from playlist; select count(*) from media_file;"
drilladmin
1
1
```

**Expected**

After `Restore complete`, the users, playlists and library rows from the backup file are
in the database.

A note on how this was measured, since SQLite makes it easy to fool yourself: the
conclusion rests on the application's own answers (`createAdmin` returning 200, sign-in
returning 401), not on reading the database file. Reading a live Navidrome database by
copying `navidrome.db`, `-wal` and `-shm` together gives a misleading result, because a
stale `-shm` makes SQLite report an empty database.

---

## Issue 2 (documentation) - the restore section is missing the steps that make it work

**Title:** `Docs: backup restore needs an existing database, and --backup-file is a name inside Backup.Path`

The [backup page](https://www.navidrome.org/docs/usage/admin/backup/) is unusually good -
the scope note ("ONLY backs up the database ... does NOT back up the music or the
config") is the sentence most projects are missing. The restore section is three
sentences, and following it exactly does not work:

1. **It refuses to run without an existing database.**

   ```text
   level=fatal msg="No existing database" path=/data/navidrome.db
   ```

   That is the state a new machine is in, which is the machine people restore onto. The
   working sequence is: start Navidrome once so it creates a database, stop it, then
   restore - and the page's two warnings are both about *not* having it running, which
   makes the missing step easy to reason past.

2. **`--backup-file` is not documented, and it is not a path.** The value is resolved
   inside `Backup.Path`. Given an absolute path, the command fails with an empty path in
   its own error message:

   ```text
   level=fatal msg="Error restoring database" backup path= \
     error="getting backup connection: unable to open database file: no such file or directory"
   ```

   so `Backup.Path` / `ND_BACKUP_PATH` has to be configured for the restore as well.

3. **`--force` is not documented** either, and without it the command asks a question
   that no script can answer.

Suggested replacement for the restore section:

> Restoring requires a database to already exist. On a new machine, start Navidrome
> once so that it creates one, stop it, and then run the restore.
>
> ```sh
> navidrome backup restore --backup-file navidrome_backup_2026.01.02_03.04.05.db --force
> ```
>
> `--backup-file` takes a file name inside `Backup.Path`, not a path, so `Backup.Path`
> (or `ND_BACKUP_PATH`) must be set when restoring as well as when backing up.

I would be glad to send a docs PR for that if it would be welcome - though it is worth
holding until issue 1 is resolved, since the sequence above still does not currently
produce a restored instance.

Thank you for Navidrome, and for the scope note in particular. It is the only page in a
set of applications I have been testing that says plainly what its backup does not
contain.
