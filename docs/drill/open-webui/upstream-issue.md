# Draft issue for Open WebUI - filed 2026-09-04

**Status: filed on 2026-09-04, with the human's sign-off (stop point 2):** <https://github.com/open-webui/docs/issues/1378>, on the documentation repository.
**Clarified the same day:** two of the page's three example scripts already exclude
`cache/`; "1.1 GB per snapshot" applies to its point-in-time rsync snapshot script and
to copying the directory whole, which the page also suggests. The ask narrowed to a
line in the files table and a restore section, in a comment on the issue.

Where it would go: <https://github.com/open-webui/open-webui/issues>, or the
documentation repository if the docs site is generated separately.

---

**Title:** `Docs: the backups page lists cache/ as data to back up - on a fresh instance it is 99.9% of the backup`

**Environment**

- Open WebUI v0.11.1, `ghcr.io/open-webui/open-webui:v0.11.1`, Docker, default SQLite,
  volume on `/app/backend/data`.
- Following <https://docs.openwebui.com/tutorials/maintenance/backups>.

**What I did**

Started a fresh instance, signed up one account, saved one chat, stopped the stack as
the page's scripts do, and copied `/app/backend/data`.

**What I observed**

```text
1.1G    data
   0    data/uploads
 32K    data/webui.db-shm
160K    data/webui.db-wal
184K    data/vector_db
632K    data/webui.db
1.1G    data/cache
```

`cache/` is a Hugging Face hub tree - `all-MiniLM-L6-v2` and `faster-whisper-base` -
downloaded on first start. The page lists `cache/` among the five things the data store
contains, next to `webui.db` and `uploads/`, with no note that it is regenerable. A
person following the page keeps 1.1 GB per snapshot for what is, at that moment, about
a megabyte of their own data.

Two smaller things from the same run, in case they are useful:

1. **The cache contains symlinks** - 46 of them in my copy, from `snapshots/` into
   `blobs/`. Any restore path that cannot create symlinks (a different operating system,
   a filesystem without them, some archive tooling) will not reproduce the tree. Excluding
   `cache/` makes this go away as well.

2. **There is no restore procedure on the page.** It covers taking the backup
   thoroughly and stops there. A person restoring for the first time is doing it on the
   worst day they have had this year, and the two things they need to be told are the
   order of operations and that `webui.db-wal` has to travel with `webui.db` - on my
   instance the `-wal` was 160 KiB against a 632 KiB database, so a copy of the `.db`
   alone would be missing the newest writes.

**Suggested change**

On the existing page, next to the list of five:

> `cache/` holds embedding and speech models that Open WebUI downloads on demand. It is
> safe to exclude from backups - it will be re-downloaded - and on a fresh instance it is
> already larger than everything else in this directory put together.

and a short *Restoring* section:

> Stop the stack, replace `/app/backend/data` with the backup, and start it again. If
> you copied files individually rather than the whole directory, make sure
> `webui.db-wal` came with `webui.db`.

I would be glad to send a docs PR with both if that would be welcome. The backups page
is already one of the better ones in self-hosted software - it is the only project in a
set I have been testing that has a page about backups at all.
