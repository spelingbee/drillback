# The official-docs restore drill

For each popular self-hosted application: follow its **own backup documentation
literally**, take the backup that documentation describes, restore it with `drillback`,
and record whether the application comes back with its data.

The finding is never "application X loses data". It is "if you follow the documentation
as written, here is what you get back". A gap in the documentation is the expected
result, and each one is written up as a respectful draft issue that a human reviews and
files - or does not.

Rules this drill runs under, in full, are in the session brief; the ones that matter to a
reader:

- The backup is **exactly** what the documentation says. Improving it defeats the study.
- Every FAIL and PARTIAL is reproduced twice from an empty scratch directory. No FAIL
  without logs.
- Ambiguity in the documentation is itself a finding. Where a page can be read two ways,
  the most obvious reading is the verdict and the other one is tested too when it is
  cheap.
- **Nothing is filed without a human.** `upstream-issue.md` in each folder was written as a draft; the ones that were filed say so at the top, with the link, and were filed on 2026-09-04 with the human's sign-off. The list is in [summary.md](summary.md#what-was-filed-upstream).

**Start here:** [summary.md](summary.md) has the totals, the three most instructive
cases, the patterns that repeat, and the caveats. [SKIPPED.md](SKIPPED.md) records every
application the drill passed over and why.

## Results

| App | Stars | Docs | Version | Result | One-line cause |
|---|---|---|---|---|---|
| [n8n](n8n/) | 202,807 | [CLI page](https://docs.n8n.io/deploy/host-n8n/configure-n8n/use-the-command-line) | 2.36.8 | **PARTIAL** | The only thing the docs call a backup - `export:workflow/credentials --backup` - has no users in it and no encryption key, so the restored instance asks you to create an owner and cannot decrypt a credential. Backing up `.n8n` instead: PASS. |
| [open-webui](open-webui/) | 150,348 | [Backups](https://docs.openwebui.com/tutorials/maintenance/backups) | v0.11.1 | **SKIPPED** / **PASS** | The documented data directory is 1.1 GB after one sign-up and one chat, of which 1,012 KiB is the user's data and the rest is a re-downloadable model cache the page lists as data - and its 46 symlinks are why the verbatim reading could not be restored on this host. Without cache/: PASS. |
| [memos](memos/) | 62,640 | [Docker Compose](https://usememos.com/docs/deploy/docker-compose) | 0.30.0 | **PASS** / **FAIL** | Copying the data directory restores everything. Copying the file the deploy page calls "the database" restores an empty Memos: memos_prod.db was 4 KiB and its -wal was 160 KiB. The Backup & Restore page, which the deploy page does not link, gets it right; the drill missed it (corrected 2026-09-04). |
| [gogs](gogs/) | 47,784 | [CLI reference](https://gogs.io/advancing/cli-reference) | 0.14.3 | **FAIL** / **PASS** | `gogs backup` writes a complete archive and `gogs restore --from` cannot put it back inside the official image: it resolves the database path against its working directory, and moves unpacked files with `rename` across a volume boundary. Copying /data instead: PASS. |
| [siyuan](siyuan/) | 46,036 | [README, Docker section](https://github.com/siyuan-note/siyuan#docker-deployment) | v3.8.2 | **PASS** | No backup section. The workspace restores, and 37 of its 38 MB are the bundled themes under conf/appearance - the notes were 11 KB. The README's own start command also no longer runs on v3.8.2. |
| [trilium](trilium/) | 37,632 | [Backup.md](https://github.com/TriliumNext/Trilium/blob/main/docs/User%20Guide/User%20Guide/Installation%20%26%20Setup/Backup.md) | v0.105.0 | **PASS** | The first application here whose documented backup and documented restore both work as written - and whose restore procedure names document.db-wal and -shm, which is what two other applications in this drill lose data for want of. |
| [filebrowser](filebrowser/) | 35,968 | [installation.md](https://github.com/filebrowser/filebrowser/blob/master/docs/installation.md) | v2.63.23 | **PASS** / **PARTIAL** | The word "backup" appears nowhere in the documentation. Keep all three volumes and everything comes back; keep the files without /database and File Browser starts, serves your files, and creates a new admin whose password is printed once to a log. The project is archived on 2026-09-01. |
| [changedetection](changedetection/) | 33,415 | [Restoring backup files](https://github.com/dgtlmoon/changedetection.io/wiki/Restoring-backup-files) | 0.55.8 | **PASS** | The zip its Backups page writes restores the watches and their history snapshots, by the procedure its own wiki gives - and that wiki opens by telling you to stop the instance first, which is the sentence two other applications here lose data for want of. |
| [beszel](beszel/) | 24,830 | [hub installation](https://beszel.dev/guide/hub-installation) | 0.18.8 | **PASS** / **PARTIAL** | No page on what to back up. The data directory restores everything; data.db alone restores an empty hub - 4 KiB in the file, 700 KiB in its -wal - and because the documented USER_EMAIL variables recreate the account, the person signs in with their own password and finds nothing there. |
| [navidrome](navidrome/) | 23,213 | [Automated Backup](https://www.navidrome.org/docs/usage/admin/backup/) | 0.63.2 | **FAIL** | The best backup page in the drill, and its restore does not work: `navidrome backup restore` refuses to run without an existing database, and once given one it reports `Restore complete` and leaves the instance empty. A copy of /data restores everything. |
| [listmonk](listmonk/) | 23,172 | [upgrade page warning](https://listmonk.app/docs/upgrade/) | v6.2.0 | **PARTIAL** | No backup page; the word appears twice, both times as "take a backup of the Postgres database". The dump restores everything in the database including the row naming each uploaded image - and the images themselves live in /listmonk/uploads, which nothing mentions. |
| [convertx](convertx/) | 18,630 | [README, Deployment](https://github.com/C4illin/ConvertX#deployment) | v0.18.0 | **PASS** / **FAIL** | No backup guidance and no mention of the word. The data directory restores everything; mydb.sqlite alone gives an instance that answers 403 at the login page, because the rows were in the -wal and the uploaded files were never in the database at all. |
| [freshrss](freshrss/) | 15,873 | [Backup](https://freshrss.github.io/FreshRSS/en/admins/05_Backup.html) | 1.29.1 | **PASS** | The most precise backup page in the drill - each item marked required, recommended or optional, and one directory named as skippable. Both claims in its first sentence hold: ./data is what you need, and cache/ (55% of it here) can be left out. |
| [gotify](gotify/) | 15,815 | [configuration reference](https://gotify.net/docs/config) | 3.1.0 | **PASS** / **PARTIAL** | No backup page. The data directory restores everything; gotify.db on its own restores the accounts, applications, tokens and messages, and leaves every uploaded application icon behind as a broken image. |
| [mealie](mealie/) | 13,095 | [Backups and Restores](https://docs.mealie.io/documentation/getting-started/usage/backups-and-restoring/) | v3.24.0 | **PASS** | The page recommends stopping the container and copying /app/data in bold, and that is exactly what works. The integrated backup zip was not tested; the reason is in result.md. |

Verdicts:

- **PASS** - the documented backup restores an application with its data in it.
- **PARTIAL** - it boots, and something a person would expect to have is missing.
- **FAIL** - it does not come back.
- **SKIPPED** - not testable inside the drill's budget or isolation rules. The reason is
  recorded; a skip is data too.

## What is in each folder

| File | What it is |
|---|---|
| `docs.md` | The official documentation: URL, version, and the steps quoted as written. |
| `compose.yaml` | The application deployed the way its own documentation says to. |
| `run.sh` | Every command the drill ran, in order. This is the file that was executed. |
| `steps.md` | The same commands with their real output, and every deviation named. |
| `recipe/` | The recipe used against the documented backup, when it differs from the registry one. |
| `result.md` | The verdict, the report, the root cause with evidence, and what the docs would need to say. |
| `result-*.txt`, `result-*.json` | The reports, written by the tool. Never by hand. |
| `upstream-issue.md` | The draft issue, and at the top whether it was filed and where. |

## Running one

From the repository root, with docker and restic on PATH:

```sh
bash docs/drill/n8n/run.sh
```

Shared helpers are in [`lib.sh`](lib.sh). Every application gets its own compose project
named `drill-<app>`, and it is torn down with its volumes at the end of the run.
