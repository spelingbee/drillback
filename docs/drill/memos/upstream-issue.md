# Draft issue for Memos - NOT FILED

**Status: draft. Nothing has been filed (CLAUDE.md stop point 2).**

Where it would go: <https://github.com/usememos/memos/issues>, or the documentation
source if the site is generated from a separate repository.

---

**Title:** `Docs: "back up the database" points at memos_prod.db, which on a running instance is mostly empty`

**Environment**

- Memos 0.30.0, `neosmemo/memos:0.30.0`, Docker, `MEMOS_DRIVER=sqlite`, the compose file
  from <https://usememos.com/docs/deploy/docker-compose>.

**What the documentation says**

The deployment pages say to "keep your data directory on persistent storage" and to
"back up both the database and any local assets if you do not use database-backed
attachments". There is no backup page and no restore procedure, so that sentence is the
guidance a person acts on.

**What I did**

1. Started Memos with the documented compose file.
2. Created the host account and one memo through the API.
3. Looked at the data directory:

   ```text
   -rw-r--r--  4096    memos_prod.db
   -rw-r--r--  32768   memos_prod.db-shm
   -rw-r--r--  160712  memos_prod.db-wal
   ```

4. Copied `memos_prod.db` - "the database" - and restored it into a fresh instance.

**What I observed**

Memos starts, `/healthz` answers 200, `PRAGMA integrity_check` returns `ok`, and the
instance is empty: no accounts, no memos, and the first-run screen asking for a new host
account. Copying the whole directory instead (all three files) restores everything
correctly.

**Why this is worth a documentation change**

`memos_prod.db` is the file whose name matches the phrase "the database", and on a
running instance almost nothing is in it - the writes are in `-wal` until a checkpoint.
The failure is silent in both directions: the backup looks fine (a valid SQLite file)
and the restore looks fine (a healthy, empty Memos). Nothing warns the person that they
have lost anything.

**Suggested change**

A short *Backup and restore* page, or one sentence on the Docker and Docker Compose
pages:

> Back up the whole `/var/opt/memos` directory. If you copy the SQLite database on its
> own, copy `memos_prod.db-wal` and `memos_prod.db-shm` with it, or stop Memos first -
> otherwise the copy will be missing everything written since the last checkpoint.

Adding the reverse direction - "to restore, stop Memos, replace the directory, start it
again" - would round it out; there is no restore documentation at the moment.

I am happy to open a docs PR with that page if it would be welcome. Thank you for
Memos - the deployment story is otherwise about as simple as self-hosting gets.
