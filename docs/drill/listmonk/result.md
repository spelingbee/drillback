# listmonk - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the Postgres database | `pg_dump`, 71.0 KiB of plain SQL | **PARTIAL** (`RESTORE UNUSABLE`, exit 1, 5 of 6 checks) | [result-db-only.txt](result-db-only.txt) |
| B - the database and the uploads directory | the same dump plus `/listmonk/uploads` | **PASS** (exit 0, 6 of 6 checks) | [result.txt](result.txt) |

listmonk restores beautifully from a plain `pg_dump`. Lists, subscribers, campaigns,
templates, settings and the admin account all come back. What does not come back is
every image anybody ever put in a campaign, and nothing anywhere warns about it.

## Reading A, in full

```text
  inputs     db       /srv/listmonk/db.sql    71.0 KiB  plain SQL
             uploads  /srv/listmonk/uploads        0 B  0 files

  CHECKS
  ok  health              The server reports itself healthy
  ok  lists-in-db         The lists survived the restore -> 1
  ok  subscribers-in-db   The subscribers survived the restore -> 1
  ok  users-in-db         The admin account survived the restore -> 1
  ok  media-row-in-db     The media library still has its row -> 1
  X   media-file-on-disk  The uploaded file is on disk, not just in the database
                            expect  glob_min_count: 1 for *.png
                            got     0 matches

  RESTORE UNUSABLE  5/6 checks
```

The verbatim report is in [result-db-only.txt](result-db-only.txt).

Five of six. The database is perfect. `media-row-in-db` passes and
`media-file-on-disk` fails, and that pair is the whole finding: **the row survived and
the file it names did not.** listmonk starts, the media library page lists the image,
and the image is a broken link. So is every campaign that used it - including ones
already sent, whose archived copies now point at nothing.

## Root cause

listmonk's default media provider is `filesystem`, and its default upload path resolves
to `/listmonk/uploads` in the official image. With a completely stock configuration, the
drill uploaded one image through the API and listmonk wrote two files:

```text
-rw-r--r-- 1 root root  70 drill-canary.png
-rw-r--r-- 1 root root 814 thumb_drill-canary.png
```

The `media` table holds a row naming that file. Nothing about it is in the database.

The documentation says "take a backup of the Postgres database" twice, in bold, on two
different pages, and never mentions this directory. The project's own
`docker-compose.yml` does mount it - with a comment saying you have to change a setting
before it is used, which is not true of a stock install and makes it easy to conclude
that the directory is optional.

## Reading B works

The same dump with the uploads directory beside it: 6 of 6.
[`recipes/listmonk`](../../../recipes/listmonk) is the recipe, and it passes both stages
of `restored recipe test`. It declares two inputs on purpose - the dump and the
directory - so that a backup of only the first is reported as this and not as a
mysteriously broken media library.

## What the documentation would need to say instead

A backup page, or two lines wherever the warning already appears:

> To back up listmonk, take a `pg_dump` of the Postgres database **and** a copy of the
> media uploads directory (`/listmonk/uploads` in the Docker image; whatever
> Admin -> Settings -> Media names, if you have changed it). The database holds a row
> per uploaded file; the files themselves are only on disk.

The existing warning is on the upgrade page, which is the right place for it and the
wrong place for it to be the only one: people upgrade far more often than they restore,
and the backup they take before an upgrade is the backup they will still have when
something else goes wrong.

## Not tested

- **The S3 media provider.** With `upload.provider = s3` the files are somewhere else
  entirely and this finding does not apply - which is another thing a backup page would
  need to say.
- **Campaign archives and bounce data.** Both live in Postgres, so reading A covers
  them; the drill did not seed either.
- **Larger installations.** The dump here is 71 KiB. Nothing in the result depends on
  size, but the timings do.

## Reproduced

Both verdicts twice, from an empty scratch directory each time, by running `run.sh` end
to end (2026-08-30 12:43 UTC and 12:46 UTC).
