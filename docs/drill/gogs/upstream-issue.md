# Draft comment / issue for Gogs - NOT FILED

**Status: draft. Nothing has been filed (CLAUDE.md stop point 2).**

**Important: this is very likely a comment on an existing issue, not a new one.** Two
open issues already describe this wall:

- [#4339 - Cannot restore inside Docker container](https://github.com/gogs/gogs/issues/4339)
- [#7684 - gogs restore in Docker failure - GOGS_CUSTOM /data/gogs is moved](https://github.com/gogs/gogs/issues/7684)

A human should read both before anything is posted, and should prefer adding the
reproduction below to whichever of them is the better home over opening a third. A
separate, small documentation issue is proposed at the end.

---

## Draft comment for #4339 (or #7684)

Hello - a reproduction on 0.14.3 with the official image, in case a current one is
useful. I hit the same wall while testing restores across a set of self-hosted
applications, and the three failure modes turn out to be separate.

**Environment:** Gogs 0.14.3, `gogs/gogs:0.14.3`, SQLite, one volume on `/data`, run as
the image runs it (git user, working directory `/app/gogs`, `GOGS_CUSTOM=/data/gogs`).
The archive was produced by `./gogs backup --target=/backup`, which is what the image's
own `docker/runtime/backup-job.sh` runs.

**1. Restore never reaches the archive's configuration.**

```text
$ gosu git ./gogs restore --from=/backup/gogs-backup-20260830120140.zip
[FATAL] Failed to start application: set engine: connect to database:
  create directories: mkdir /app/gogs/data: permission denied
```

It connects to the database first and resolves `[database] PATH = data/gogs.db` against
the working directory instead of `GOGS_CUSTOM`, so it tries to create `/app/gogs/data`,
which the git user cannot write. Copying the archive's own `custom/conf/app.ini` to
`/data/gogs/conf/app.ini` first does not change this, and neither does passing
`--config=/data/gogs/conf/app.ini` explicitly.

**2. With `/app/gogs` writable, the import fails on a cross-device rename - after moving
the live configuration aside.**

```text
[FATAL] Failed to import 'custom': rename /tmp/gogs-backup/custom /data/gogs:
  invalid cross-device link
```

`/tmp` is the container filesystem and `/data` is a volume, so `os.Rename` cannot move
between them. The rename of the *existing* `/data/gogs` to `/data/gogs.bak` has already
happened at that point, so an instance that had a configuration a moment earlier has
none.

**3. `TMPDIR` on the volume moves the failure rather than removing it.**

```text
[FATAL] Failed to import 'data': rename /data/tmp/gogs-backup/data/avatars
  /app/gogs/data/avatars: invalid cross-device link
```

because the destination is still the mis-resolved `/app/gogs/data` from (1). There is no
single `TMPDIR` that is on the same device as both trees.

The archive itself looks complete - `custom/conf/app.ini`, 38 table dumps under `db/`,
`repositories.zip`, and `data/avatars/1` for an avatar I had uploaded. So this reads as
a restore-path problem rather than a backup problem, which is the good news.

If it would help, the two shapes that would make (2) and (3) go away are copy-then-remove
instead of `os.Rename` when the rename returns `EXDEV`, and unpacking into a temporary
directory inside the destination tree. I would be glad to send a PR for either if you
would take it - please say which you would prefer, or if you would rather leave the
command as a non-Docker tool and say so in the documentation.

Thank you for Gogs. The `backup` side worked perfectly.

---

## Draft documentation issue (small, separate)

**Title:** `Docs: CLI reference presents backup/restore as a pair, but restore does not work in the official Docker image`

The [CLI reference](https://gogs.io/advancing/cli-reference) documents:

```bash
gogs backup
gogs restore --from <archive>
```

For anyone running the official image, the second command does not work today (see
#4339 and the reproduction above). The image also ships a scheduled backup that calls
`gogs backup` for you, so it is easy to end up with a directory of archives and the
reasonable belief that they can be restored.

Suggested note on that page:

> **Docker:** `restore` is not currently usable inside the official image. It resolves
> the database path against its working directory rather than `GOGS_CUSTOM`, and it
> moves unpacked files with `rename`, which fails between the container filesystem and
> the `/data` volume (#4339). To restore a Dockerised instance, keep a copy of the
> `/data` volume and put it back.

Two smaller things noticed while reading, in case they are useful: `https://gogs.io/docs`
and `https://gogs.io/docs/intro/backup_and_restore` both 404, so older links to the
backup page - including ones in search results and forum threads - lead nowhere.

Happy to open a docs PR for the note if it would be welcome.
