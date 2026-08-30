# The official-docs restore drill - summary

Fifteen popular self-hosted applications. For each one: read its own backup
documentation, take the backup that documentation describes, restore it, and check
whether the application comes back with its data in it.

Every verdict below was produced by `restored check` against a restic repository, and
every FAIL and PARTIAL was reproduced twice from an empty scratch directory. The reports
are in each application's folder. Nothing has been filed anywhere: the `upstream-issue.md`
drafts are for a human to review.

## Totals

**15 applications tested.** The verdict is on the *primary* documented reading - what a
person following the documentation most obviously ends up with.

| Verdict | Count | Applications |
|---|---|---|
| **PASS** | 10 | Beszel, changedetection.io, ConvertX, File Browser, FreshRSS, Gotify, Mealie, Memos, SiYuan, Trilium |
| **PARTIAL** - it boots, something is missing | 2 | n8n, listmonk |
| **FAIL** - nothing came back | 2 | Gogs, Navidrome |
| **SKIPPED** - not decidable on this host | 1 | Open WebUI (its backup contains symlinks restic cannot recreate on Windows without privilege; the same backup minus the model cache passes) |

Secondary readings were tested wherever the documentation could be read two ways. Five
more failed:

| App | The second reading | Verdict |
|---|---|---|
| Memos | "the database" = `memos_prod.db` | **FAIL** - empty instance |
| Beszel | "the database" = `data.db` | **PARTIAL** - empty hub, and you can still log in |
| File Browser | the files without `/database` | **PARTIAL** - files back, every user and share gone |
| Gotify | `gotify.db` without `images/` | **PARTIAL** - icons gone |
| ConvertX | "the database" = `mydb.sqlite` | **FAIL** - login answers 403 |

And the documentation itself:

- **8 of 15 have no page about backups.** n8n, File Browser, Memos, listmonk, Gotify,
  Beszel, SiYuan, ConvertX. In File Browser's and ConvertX's case the word does not
  appear anywhere in the documentation at all.
- **6 have a page whose subject is backups**: Open WebUI, Navidrome, Trilium, Mealie,
  FreshRSS, and changedetection.io (a wiki page about restoring, plus the feature in the
  application). Gogs has a section on its CLI reference page.
- **7 of the 15 describe a restore at all**: n8n, Gogs, Navidrome, Trilium, changedetection.io,
  Mealie and FreshRSS. The drill followed five of those procedures step by step, and
  **three of the five did not work as written**: Gogs' `restore` cannot run in its own
  image, Navidrome's reports success and restores nothing, and n8n's
  `import:credentials` aborts on the directory n8n's own export commands write. The two
  that worked were Trilium's and changedetection.io's. Mealie's (an upload through the
  admin interface) and FreshRSS's (`db-restore.php`) were not followed end to end and are
  counted in neither column.

## The three most instructive cases

**Navidrome writes a good backup and cannot put it back.** Its backup page is the best
in this drill - it is the only one that says plainly what the backup does *not* contain
("ONLY backs up the database ... does NOT back up the music or the config"). Following
its restore section exactly, `navidrome backup restore` refuses to run: `fatal: No
existing database`, which is the state of every machine anybody restores onto. Supply
the two conditions the page does not mention - start the server once so a database
exists, and set `ND_BACKUP_PATH` because `--backup-file` is resolved inside it rather
than taken as a path - and the command answers `Restore complete` in six milliseconds
and leaves the instance empty. `POST /auth/createAdmin` then returns 200, which Navidrome
only does when it has no users at all, and the account from the backup gets a 401. The
backup file, read directly, has the user, the playlist and the library row in it. A
careful page, a complete backup, and a restore path nobody has walked recently.

**Gogs ships a scheduled backup its own image cannot restore.** `gogs backup` produces a
complete archive - configuration, 38 table dumps, the repositories, and the avatar files
that live outside both - and the official image will produce one every night if you turn
the feature on. `gogs restore --from` then dies at `mkdir /app/gogs/data: permission
denied`, because it connects to the database before reading the archive's configuration
and resolves the database path against its working directory instead of `GOGS_CUSTOM`.
Make that directory writable and it gets one step further, to `rename ... invalid
cross-device link`, having already moved the live `/data/gogs` aside to `/data/gogs.bak` -
so an instance that had a working configuration a moment ago has none. This is not new:
issues #4339 and #7684 describe the same wall. What the drill adds is that the backup is
fine and the restore is the broken half, which is the difference between "your backups
are worthless" and "there is one bug to fix".

**Memos, and the file that is not the database.** Memos' deployment pages say to "back up
both the database and any local assets". The file called `memos_prod.db` was 4 KiB.
The file called `memos_prod.db-wal`, sitting beside it, was 160 KiB, and held the schema,
the account and the memo. Copy the first and restore it and Memos starts, answers
`/healthz` with 200, passes `PRAGMA integrity_check`, and asks you to create your first
account. Nothing is corrupt; every table is there and every table is empty. Beszel
repeated it exactly - 4 KiB against 700 KiB - with an extra twist: because the
documented `USER_EMAIL` and `USER_PASSWORD` variables recreate the account on an empty
database, the person restoring signs in successfully with their own password and simply
finds nothing there.

## The patterns, in order of how often they came up

**1. The database is not the data.** Six applications keep something a person would call
their data outside the database, and the documentation of every one of them points only
at the database:

| App | What is outside it |
|---|---|
| listmonk | every uploaded image, in `/listmonk/uploads`; the `media` row survives and the file does not |
| Gotify | uploaded application icons, in `/app/data/images` |
| File Browser | the reverse - the files are what people back up, and `/database` holds every user, permission and share link |
| n8n | the encryption key, in `.n8n/config`; without it a restored credential is a base64 string |
| Gogs | avatars and attachments under `gogs/data` - in the archive, as it happens, but in neither the database dump nor the repositories |
| ConvertX | the uploaded and converted files, under `uploads/<user>/<job>/` and `output/<user>/<job>/`; the `jobs` rows point at them by id |

**2. The `-wal` is where the data is.** Five applications keep SQLite in WAL mode, and
the split is not subtle on a young instance:

| App | `.db` | `-wal` | What a copy of the `.db` alone gets you |
|---|---|---|---|
| Memos | 4 KiB | 160 KiB | nothing - tested, FAIL |
| Beszel | 4 KiB | 700 KiB | nothing - tested, PARTIAL |
| n8n | 1.5 MB | 4.1 MB | not tested separately |
| Open WebUI | 632 KiB | 160 KiB | not tested separately; here the main file holds most of it |
| ConvertX | 20 KiB | 16 KiB | nothing - tested, FAIL |

The three that were tested came back completely empty. The other two are listed to show
that the split is normal rather than exotic, not to claim a failure that was not
measured: on Open WebUI in particular the main file held most of the data, so a `.db`
copy would have lost the newest writes rather than everything.

Exactly one project's documentation names the `-wal` and `-shm` files: Trilium's, in its
restore procedure. Nobody else mentions they exist.

**3. An empty restore looks healthy.** This is what makes the two patterns above
dangerous rather than merely annoying. In every case where the restore came back empty -
Memos, Beszel, File Browser, Gogs, Navidrome, ConvertX - the application started,
answered its health endpoint, passed an integrity check where there was one to pass, and
offered to create the first account. ConvertX is the closest thing to an exception, and
only just: the login answers 403 rather than letting you in, so at least something says
no - and what it then offers is the setup page. There is no error anywhere. The signal that something is wrong
is a screen that looks exactly like a new installation, on a day when you are not in the
mood to notice.

**4. Backup pages are about backing up.** Seven of the fifteen describe a restore at
all. The drill followed five of those procedures step by step and three did not work as
written. Backing up is the half that gets exercised - by cron, nightly, for years.
Restoring is the half nobody runs until the day it matters, and it shows.

**5. Backups carry a lot of re-downloadable content, and nobody says so.**

| App | Backup | Of which regenerable |
|---|---|---|
| Open WebUI | 1.1 GB | 1.1 GB of Hugging Face model cache, for 1,012 KiB of user data |
| SiYuan | 38 MB | 37 MB of bundled themes and icon sets, for 11 KiB of notes |
| FreshRSS | 1.3 MB | 720 KiB of cache - **and its documentation says so** |

FreshRSS is the exception that shows what the sentence costs: "You can skip `cache/`;
FreshRSS rebuilds it."

## Three headline options for a launch post

Numbers only as measured, over the fifteen applications tested.

1. **"We followed fifteen self-hosted apps' own backup instructions. Eight of them have
   no backup page at all, and of the five documented restore procedures we could follow
   step by step, three did not work as written."**

2. **"'Restore complete.' The instance was empty."** - Navidrome's restore command
   reports success in six milliseconds and leaves nothing behind; Gogs' restore cannot
   run inside Gogs' own Docker image. Both projects' backups are fine. It is the other
   half that nobody runs.

3. **"On three of fifteen apps, copying the file called 'the database' restored
   nothing."** - Memos' `.db` was 4 KiB against a 160 KiB `-wal`, Beszel's 4 KiB against
   700 KiB, ConvertX's 20 KiB against 16 KiB.
   Restore the `.db` alone and the application starts, passes its own integrity check,
   and asks you to create your first account. One project out of fourteen tells you
   those files exist.

## Caveats a reader is owed

- **Fifteen applications, not a survey.** They are the top of a popularity list,
  filtered by what fits in a twenty-minute budget, which skews the set towards
  single-container applications with SQLite databases. See [SKIPPED.md](SKIPPED.md) for
  everything passed over and why - including immich, which has good backup documentation
  and is the most valuable untested application in the list.
- **Fresh instances.** Every instance was seeded with one account and one or two objects,
  minutes old. The WAL sizes in particular look different on an instance that has been
  running for a year and has checkpointed many times; the split does not disappear, but
  the ratio changes.
- **One host, one operating system.** Windows with Docker Desktop. It cost Open WebUI a
  verdict (symlinks) and nothing else, but it is not the platform most of these run on.
- **Each application is one version, on one day.** Every folder records which.
- **Where a page could be read two ways, the drill chose one as primary.** That choice is
  stated in each `docs.md` and it is arguable; the other reading was tested too wherever
  it was cheap.
