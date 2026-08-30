# Trilium - result

| Reading | What was backed up, and how it was restored | Verdict | Report |
|---|---|---|---|
| A - Trilium's own backup | `backup/backup-now.db`, restored by the documented procedure: delete `document.db` and its `-wal`/`-shm`, copy the backup over it, `chmod 600`, start | **PASS** (exit 0, 3 of 3 checks) | [result.txt](result.txt) |
| Control - the data directory | all of `/home/node/trilium-data`, put back and started | **PASS** (exit 0, 4 of 4 checks) | [result-data.txt](result-data.txt) |

Trilium is the first application in this drill where the documented backup and the
documented restore both work, exactly as written, with no missing step. Follow the page
and you get your notes back, and the password that opened the instance still opens it.

## Reading A, in full

```text
  inputs     backups  /home/node/trilium-data/backup   3.1 MiB  1 file

  CHECKS
  ok  health-check             The server answers its health check
  ok  note-restored            The note that was backed up is in the restored database
                                 drill-canary-note rows: 1
  ok  password-still-opens-it  The password the instance was set up with still opens it

  PASS  3/3 checks
```

The verbatim report is in [result.txt](result.txt).

`password-still-opens-it` is the check worth naming. Trilium keeps the password
verification hash in the `options` table inside `document.db`, so a restore that brought
the notes back without it would produce an instance nobody could open. It came back.
Attachments and images live in the same file, which is why one 3.1 MB database is the
whole backup.

## What is worth saying anyway

Two things that are true, that the page is honest about, and that are still worth a
reader's attention:

**The backups live inside the data directory.** `backup/` is a sibling of `document.db`,
on the same disk, in the same volume, in the same container. They protect against a bad
migration or a person deleting the wrong note; they do not protect against losing the
machine. The page says as much - "you're encouraged to add some better backup solution" -
and it is the single most important sentence on it.

**A young instance has no backup at all.** The automatic copies are daily, weekly and
monthly. In this drill the data directory looked like this after setup:

```text
config.ini
document.db
document.db-shm
document.db-wal
log/
session_secret.txt
tmp/
```

There is no `backup` directory. It appeared only after `Backup Now` was pressed. An
instance that is four hours old, on the day something goes wrong, has nothing in the
place its documentation points at.

## Not tested

- **The "Restore from backup" option in the setup menu.** That is a browser flow; the
  drill tested the command-line alternative the same page documents.
- **`.tnbackup` files** (the encrypted/protected form). The page says the alternative
  restore procedure does not support them, and the drill did not try.
- **Protected notes.** None were created, so nothing here is claimed about restoring an
  instance whose protected session key matters.
- **Synchronisation as a backup.** The page mentions it; the drill did not test it.

## Reproduced

Both verdicts are PASS, so the two-run rule for failures does not apply. The leg was run
once end to end (2026-08-30 12:59 UTC), and `restored recipe test recipes/trilium` passes
both stages independently.
