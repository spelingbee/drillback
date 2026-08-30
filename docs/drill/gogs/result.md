# Gogs - result

| Reading | What was backed up, and how it was restored | Verdict | Report |
|---|---|---|---|
| A - the documented commands | `gogs backup` -> a 22.5 KiB zip; restored with `gogs restore --from <archive>` in the official image | **FAIL** (`RESTORE UNUSABLE`, exit 1, **0 of 4** checks) | [result-archive.txt](result-archive.txt) |
| B - the documented volume | a copy of `/data` (40 files, 405 KiB); restored by starting the image over it | **PASS** (exit 0, 7 of 7 checks) | [result-data.txt](result-data.txt) |

The backup is fine. **The documented restore command does not run in the official
Docker image**, and what it does before it fails leaves the instance worse than it found
it.

## Reading A, in full

```text
  inputs     archive  /backup   22.5 KiB  1 file

  restore    ok          1.7s   1 input
  compose    ok          1.8s   2 services
  ready      ok         0.52s   Gogs is accepting connections

  CHECKS
  X  not-the-installer     The restored instance is installed, not asking to be set up
                             expect  status: 404
                             got     status: 200
  X  repository-on-disk    The bare repository is on disk
  X  repository-browsable  Gogs serves the repository page
                             expect  status: 200
                             got     status: 302
                             got     <a href="/install">Found</a>.
  X  avatar-file-present   The uploaded avatar came back

  RESTORE UNUSABLE  0/4 checks

  LIKELY CAUSE
    The application booted into its first-run setup wizard
                                                   (hint: app/still-in-setup)
```

Nothing came back. The restored instance is a brand new Gogs asking to be installed.

## Root cause

The documented command was run the way the image runs Gogs - as the `git` user, from
`/app/gogs`, with `GOGS_CUSTOM=/data/gogs` in the environment. From the `restore`
service's log, in the JSON report:

```text
+ gosu git ./gogs restore '--from=/backup/gogs-backup-20260830120140.zip'
2026/08/30 12:01:58 [FATAL] [gogs.io/gogs/gogs.go:36 main()]
  Failed to start application: set engine: connect to database:
  create directories: mkdir /app/gogs/data: permission denied
```

`gogs restore` connects to the database **before** it reads the configuration out of the
archive, and it resolves the SQLite path `data/gogs.db` from `app.ini` against its own
working directory rather than against `GOGS_CUSTOM`. In the official image that is
`/app/gogs`, which belongs to root and which the `git` user cannot write to. The command
stops there, having done nothing.

Three further things were established by hand, because the first failure hides them:

**1. It is not fixed by giving it the configuration.** With the archive's own
`custom/conf/app.ini` copied to `/data/gogs/conf/app.ini` first, and with
`--config=/data/gogs/conf/app.ini` passed explicitly, the command fails identically at
`mkdir /app/gogs/data`.

**2. With `/app/gogs` made writable, it fails on a cross-device rename, after moving the
live configuration out of the way.**

```text
[FATAL] [internal/cmd/restore.go:126 runRestore()]
  Failed to import 'custom': rename /tmp/gogs-backup/custom /data/gogs:
  invalid cross-device link
```

`restore` unpacks the archive under `TMPDIR` and moves the pieces into place with
`os.Rename`. In the official image `/tmp` is the container's own filesystem and `/data`
is a volume: different devices, so every move fails. And it fails *after* renaming the
existing `/data/gogs` to `/data/gogs.bak`, so an instance that had a working
configuration a moment ago now has none.

**3. Pointing `TMPDIR` at the volume moves the failure, it does not remove it.** With
`TMPDIR=/data/tmp` and a writable `/app/gogs`, `custom` imports and the next one fails:

```text
[FATAL] [internal/cmd/restore.go:145 runRestore()]
  Failed to import 'data': rename /data/tmp/gogs-backup/data/avatars
  /app/gogs/data/avatars: invalid cross-device link
```

because the destination for `data` is still the wrong `/app/gogs/data` from cause 1.
There is no single value of `TMPDIR` that puts the temporary directory on the same
device as both the configured `/data` tree and the mis-resolved `/app/gogs` tree.

This is not new. Two open issues describe the same wall from different directions:
[#4339 "Cannot restore inside Docker container"](https://github.com/gogs/gogs/issues/4339)
and [#7684 "gogs restore in Docker failure - GOGS_CUSTOM /data/gogs is moved"](https://github.com/gogs/gogs/issues/7684).
The drill's contribution is a reproducible run and the observation that the image's own
scheduled backup produces an archive that the image cannot restore.

## The archive itself is complete

This matters, and it is the difference between "your backup is bad" and "the restore
command is broken". The 22.5 KiB archive contains:

```text
gogs-backup/metadata.ini
gogs-backup/repositories.zip
gogs-backup/custom/conf/app.ini
gogs-backup/custom/log/...
gogs-backup/data/avatars/1          <- the uploaded avatar
gogs-backup/db/User.json
gogs-backup/db/Repository.json
gogs-backup/db/Attachment.json
gogs-backup/db/LFSObject.json
... 38 tables in all, one JSON file each
```

Configuration, every table, the repositories, and the files under `data` that are
neither - the avatar this drill uploaded is in there. Nothing is missing from the
backup. What is missing is a way to put it back.

Not tested: whether the archive restores correctly outside Docker, on a host where
`/tmp` and the Gogs tree are on one filesystem and the process can write its own
directory. The archive's contents make that plausible; this drill did not measure it.

## Reading B works

A copy of `/data` restores to a Gogs that is installed, serves the repository page, has
the bare repository on disk and still has the avatar file.
[`recipes/gogs`](../../../recipes/gogs) is that recipe, and it passes both stages of
`restored recipe test`. Its `avatar-file-present` check is deliberate: avatars,
attachments and LFS objects live under `gogs/data`, in neither the database nor the
repositories, and a backup that takes "the database and the repos" leaves the rows
pointing at files that are gone.

## What the documentation would need to say instead

The CLI reference presents `backup` and `restore` as a pair. For anyone running the
official image - which is how most people run Gogs - the second half does not work. The
smallest honest fix is a note on that page:

> `restore` is not currently usable inside the official Docker image: it resolves the
> database path against its working directory rather than `GOGS_CUSTOM`, and it moves
> unpacked files with `rename`, which fails across the container filesystem and the
> `/data` volume. See #4339. To restore a Dockerised instance today, keep a copy of the
> `/data` volume and put it back.

## Reproduced

Reading A twice, from an empty scratch directory each time, by running `run.sh` end to
end (2026-08-30 11:58 UTC and 12:01 UTC): `RESTORE UNUSABLE`, 0 of 4 checks, the same
`mkdir /app/gogs/data: permission denied` in the restore service's log both times.
Reading B passed 7 of 7 in both runs.
