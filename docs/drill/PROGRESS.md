# Drill progress

Resume point for the official-docs restore drill. Updated after every application.

**Read [README.md](README.md) first** for what the drill is and what the verdicts mean.

## State

- **Started:** 2026-08-30.
- **Order:** `docs/recipes-wanted.txt`, by stars, top down. Applications that cannot run
  in CI-class resources are recorded as SKIPPED with the reason and the drill moves on.
- **Target:** 15 tested applications, extending toward 30 if the per-application budget
  (about 20 minutes) allows.

## Done

| # | App | Verdict | Registry recipe | Notes |
|---|---|---|---|---|
| 1 | n8n | PARTIAL (documented export) / PASS (data directory) | `recipes/n8n`, passes `recipe test` | Two documented readings, tested both. |
| 2 | memos | PASS (data directory) / FAIL ("the database" alone) | `recipes/memos`, passes `recipe test` | The -wal file holds everything; the .db is 4 KiB. |
| 4 | open-webui | SKIPPED (verbatim, host cannot restore its symlinks) / PASS (without cache/) | `recipes/open-webui`, passes `recipe test` | 1.1 GB of backup for 1 MB of data. |
| 5 | filebrowser | PASS (all three volumes) / PARTIAL (files without the database) | `recipes/filebrowser`, passes `recipe test` | No backup documentation at all; project archived 2026-09-01. |
| 6 | navidrome | FAIL (documented restore) / PASS (a copy of /data) | `recipes/navidrome`, passes `recipe test` | "Restore complete" and an empty instance. |
| 3 | gogs | FAIL (`gogs restore`) / PASS (the /data volume) | `recipes/gogs`, passes `recipe test` | The documented restore command does not run in the official image. |

## Next

Working down `docs/recipes-wanted.txt` from the top:

- immich (112,933)
- Stirling-PDF (90,910)
- appwrite (57,160)
- siyuan (46,036)

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
