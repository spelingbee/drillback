# Open WebUI - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the documented directory, all five items | `/app/backend/data`, 1.1 GB, 115 files, **46 symlinks** | **SKIPPED** - could not be restored on this host, see below | [result-full.txt](result-full.txt) |
| B - the same directory without `cache/` | 1,012 KiB | **PASS** (exit 0, 5 of 5 checks) | [result.txt](result.txt) |

Nothing here is a failure of Open WebUI's documentation to describe the data. Everything
a person put into the instance came back. The finding is about the fifth item on the
page's list.

## The numbers

Immediately after a single sign-up and a single saved chat, the documented data
directory looked like this:

```text
1.1G    backup/data
   0    backup/data/uploads
 32K    backup/data/webui.db-shm
160K    backup/data/webui.db-wal
184K    backup/data/vector_db
632K    backup/data/webui.db
1.1G    backup/data/cache
-- symlinks in it: 46
```

**1.1 GB of backup for 1,012 KiB of data.** The `cache/` directory is a Hugging Face hub
tree that Open WebUI downloads on first start: `all-MiniLM-L6-v2` for embeddings and
`faster-whisper-base` for speech. It is not user data, it is not configuration, and it
is byte-for-byte re-downloadable. The backup page lists it as one of the five things to
back up.

For anyone keeping daily snapshots, that is the difference between a backup measured in
megabytes and one measured in gigabytes - and, with restic's deduplication, between a
repository that grows with what people do and one that grows every time a model file's
mtime changes.

## Why reading A is SKIPPED and not FAIL

The restore never reached a verdict on this machine:

```text
  restore    FAILED      2.8s
             restic restore: exit status 1: ignoring error for
             \opt\open-webui\cache\embedding\models\models--sentence-transformers--all-MiniLM-L6-v2\
               snapshots\...\tokenizer_config.json:
             symlink ..\..\blobs\c79f2b6a0cea6f4b564fed1938984bace9d30ff0
             ... A required privilege is not held by the client.
             ... Fatal: There were 46 errors

  ERROR  0/0 checks  ·  total 3.5s  ·  teardown ok
```

The Hugging Face cache stores each file once under `blobs/` and points at it from
`snapshots/` with a relative symlink. Windows will not let an unprivileged process
create a symlink, so restic cannot lay the tree back down - 46 files, 46 errors, and the
run ends as a tool error (exit 2) rather than a verdict. This repository already
documents the same limitation for its own test suite (CLAUDE.md, *On this machine*).

That is a property of the host, not of Open WebUI, so it is recorded as SKIPPED. It is
still worth saying out loud: **a backup of Open WebUI's data directory contains symlinks,
and any restore path that cannot make symlinks - another operating system, a filesystem
without them, an archive format that stores them as their targets - silently changes what
comes back.** Reproduced twice, from an empty scratch directory each time, with the same
46 errors.

## Reading B, in full

```text
  source     restic  .../restic-no-cache
  inputs     data  /opt/open-webui               1012.0 KiB  4 files
             db    /opt/open-webui/webui.db        632.0 KiB  1 file

  CHECKS
  ok  health          The server reports itself healthy
  ok  db-integrity    The SQLite database passes PRAGMA integrity_check
  ok  users-present   At least one account survived the restore
  ok  chats-present   Chat history survived the restore
  ok  signin-works    The restored account can still sign in

  PASS  5/5 checks
```

The verbatim report, with the tool's own glyphs, is in [result.txt](result.txt).

`signin-works` is the check worth naming: it posts the seeded address and password to
`/api/v1/auths/signin` and requires a token back. A row in the `user` table does not
prove that anybody can get back in; a successful sign-in does.

## What the documentation would need to say instead

Two sentences on the existing page:

1. About `cache/`:

   > `cache/` holds embedding and speech models that Open WebUI downloads on demand.
   > It is safe to exclude from backups - it will be re-downloaded - and on a fresh
   > instance it is already larger than everything else in this directory put together.

2. A restore procedure, which the page does not have at all:

   > To restore: stop the stack, replace `/app/backend/data` with the backup, and start
   > it again. Keep `webui.db-wal` with `webui.db` if you copied files individually, or
   > the copy will be missing everything written since the last checkpoint.

The second half of that second sentence is worth having for the same reason it was
worth having for Memos: on this instance `webui.db-wal` was 160 KiB against a 632 KiB
database.

## Not tested

- **`uploads/` and RAG documents.** The seeded instance has no uploaded file and no
  document in the vector database - `uploads/` was empty at 0 bytes. So the drill did
  not exercise the path where a database row points at a file under `uploads/`.
- **The `audit.log`** item on the page's list did not exist in this deployment; auditing
  is off by default.
- **PostgreSQL deployments.** Only the documented SQLite default was tested.

## Reproduced

Reading A's restic failure twice (2026-08-30 12:07 UTC and 12:14 UTC), identically.
Reading B passed 5 of 5, and `restored recipe test recipes/open-webui` passes both
stages independently.
