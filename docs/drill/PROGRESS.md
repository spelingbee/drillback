# Drill progress

Resume point for the official-docs restore drill. Updated after every application.

**Read [README.md](README.md) first** for what the drill is and what the verdicts mean.

## State

- **Started:** 2026-08-30.
- **Order:** `docs/recipes-wanted.txt`, by stars, top down. Applications that cannot run
  in CI-class resources are recorded as SKIPPED with the reason and the drill moves on.
- **Target:** 15 tested applications, extending toward 30 if the per-application budget
  (about 20 minutes) allows.
- **Reached:** 14. The fifteenth (Stirling-PDF) was deployed and abandoned when its
  login could not be scripted inside the budget; the reason is in
  [SKIPPED.md](SKIPPED.md), and no verdict was invented for it.

## Done

| # | App | Verdict | Registry recipe | Notes |
|---|---|---|---|---|
| 1 | n8n | PARTIAL (documented export) / PASS (data directory) | `recipes/n8n`, passes `recipe test` | Two documented readings, tested both. |
| 2 | memos | PASS (data directory) / FAIL ("the database" alone) | `recipes/memos`, passes `recipe test` | The -wal file holds everything; the .db is 4 KiB. |
| 4 | open-webui | SKIPPED (verbatim, host cannot restore its symlinks) / PASS (without cache/) | `recipes/open-webui`, passes `recipe test` | 1.1 GB of backup for 1 MB of data. |
| 5 | filebrowser | PASS (all three volumes) / PARTIAL (files without the database) | `recipes/filebrowser`, passes `recipe test` | No backup documentation at all; project archived 2026-09-01. |
| 6 | navidrome | FAIL (documented restore) / PASS (a copy of /data) | `recipes/navidrome`, passes `recipe test` | "Restore complete" and an empty instance. |
| 7 | listmonk | PARTIAL (the documented pg_dump) / PASS (dump plus uploads) | `recipes/listmonk`, passes `recipe test` | The media rows come back; the files do not. |
| 8 | gotify | PASS (the data directory) / PARTIAL (gotify.db alone) | `recipes/gotify`, passes `recipe test` | Uploaded application icons are files, not rows. |
| 9 | trilium | PASS (documented backup and documented restore) | `recipes/trilium`, passes `recipe test` | The first leg with nothing to report upstream. |
| 10 | changedetection.io | PASS (the documented zip and the documented restore) | `recipes/changedetection`, passes `recipe test` | Second leg with nothing to report. |
| 11 | beszel | PASS (the data directory) / PARTIAL (data.db alone) | `recipes/beszel`, passes `recipe test` | Four checks of five pass on an empty restore. |
| 12 | mealie | PASS (the documented "best way") | `recipes/mealie`, passes `recipe test` | Integrated backup zip untested, reason recorded. |
| 13 | FreshRSS | PASS (both readings) | `recipes/freshrss`, passes `recipe test` | The best backup page in the drill; both of its claims hold. |
| 14 | siyuan | PASS (both readings) | `recipes/siyuan`, passes `recipe test` | 37 of 38 MB of the workspace are bundled themes. |
| 3 | gogs | FAIL (`gogs restore`) / PASS (the /data volume) | `recipes/gogs`, passes `recipe test` | The documented restore command does not run in the official image. |

## Next

Fourteen applications tested. [SKIPPED.md](SKIPPED.md) has the full list of what was
passed over and why; the short version of what to do next:

- **Stirling-PDF (90,910)** - fourth on the list and the one this session ran out of
  time on. Its documentation has a *Database Backups* page describing automatic daily
  backups and an import path, which is exactly the shape that failed for Gogs and
  Navidrome. The image pull was still running when the session ended.
- **immich (112,933)** - the highest-value untested application anywhere in the list.
  Specific backup documentation, four services, several gigabytes of images. It needs a
  session of its own, not a slot in a queue.
- **photoprism, appsmith, plausible, karakeep, ArchiveBox, linkwarden** - all feasible,
  none reached.

## Machine notes for the next session

- `export PATH="/c/My/Projects/Work/gotool/go/bin:/c/Users/kadyr/go/bin:$PATH"` for go
  and restic; `export MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'` for docker.
- With `MSYS_NO_PATHCONV=1`, every path handed to `docker` **and to `restored.exe`** has
  to go through `docker_path` (`cygpath -m`), or Git Bash's `/c/...` is read by Windows
  as `C:\c\...`.
- `export PYTHONIOENCODING=utf-8` before any python that prints a report: the console
  codepage here is cp1251 and a check mark in the output kills the script.
- Scratch lives in `$TMPDIR/restored-drill/<app>`; `drill_init` removes the previous
  compose project *and* its volumes before it starts.
- Leftover stacks from an interrupted run: `docker compose ls --all`, then
  `docker compose -p <name> down -v`. Do not touch projects that are not `drill-*` or
  `restored-*`; there is other work on this machine.
