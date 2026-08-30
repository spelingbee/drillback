# ConvertX - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the data directory | all of `/app/data` | **PASS** (exit 0, 6 of 6 checks) | [result.txt](result.txt) |
| B - "the database" | `mydb.sqlite` on its own | **FAIL** (`RESTORE UNUSABLE`, exit 1, 2 of 6 checks) | [result-db-only.txt](result-db-only.txt) |

Reading B fails twice over, which is what makes it worth recording after four other
applications produced the same shape: the database file alone is both **incomplete**
(the `-wal` beside it holds the rows) and **insufficient** (the uploaded files are not
in it at all).

## Reading B, in full

```text
  inputs     data  /app/data              20.0 KiB  1 file
             db    /app/data/mydb.sqlite  20.0 KiB  1 file

  CHECKS
  ok  serves                 The service answers
  ok  db-integrity           The SQLite database passes PRAGMA integrity_check -> ok
  X   users-present          The account survived the restore -> 0
  X   jobs-present           The conversion history survived the restore -> 0
  X   uploaded-file-on-disk  The file a job refers to is on disk, not just in the database
                               expect  glob_min_count: 1 for */*/*
                               got     - matches
  X   signin-works           The restored account can still sign in
                               got     login status: 403

  RESTORE UNUSABLE  2/6 checks
```

The verbatim report is in [result-db-only.txt](result-db-only.txt).

`signin-works` is the check that says what this means to a person: **403**. Not "your
files are missing" - you cannot get in at all. And because ConvertX only lets the first
account be registered, the instance is now sitting on the setup page waiting for
whoever finds it first to become its owner.

## The numbers

After registering one account and uploading one file:

```text
-rw-r--r-- 1 root root 20480 mydb.sqlite
-rw-r--r-- 1 root root 32768 mydb.sqlite-shm
-rw-r--r-- 1 root root 16512 mydb.sqlite-wal
/app/data/uploads/1/1/drill-canary.png
```

Not as extreme a split as Memos (4 KiB against 160 KiB) or Beszel (4 KiB against
700 KiB) - here the main file is the larger of the two - but the account and the job
were in the `-wal`, which is all it takes.

## Reading A works

The data directory restores the account, the job, the uploaded file, and a sign-in that
answers 302. [`recipes/convertx`](../../../recipes/convertx) is the recipe, and it passes
both stages of `restored recipe test`.

One thing in it is worth copying: `signin-works` is an `exec` check running `curl` inside
the application's own container, because ConvertX's login is a form post and the `http`
check kind can only send JSON. A 302 is a successful sign-in and a 403 is not, which
makes it a clean binary - and it is the check a person cares about most.

## What the documentation would need to say

There is no backup section at all. One line under Deployment would do it:

> Back up the whole `data` directory. It holds `mydb.sqlite` - together with its `-wal`
> file, where recent rows live until SQLite checkpoints them - and the `uploads/` and
> `output/` directories holding the files your conversion jobs refer to. Copying
> `mydb.sqlite` alone gives you an instance you cannot log in to.

## Not tested

- **A completed conversion.** The `convert` request answered 500 on this instance and
  `output/` stayed empty, so what the drill restored is an upload and its job row rather
  than a finished conversion. That is enough for the finding - the file is on disk and
  the row is in the database - but a finished conversion was not exercised, and the 500
  was not chased down. It is not claimed as a defect.
- **Multiple accounts.** ConvertX only lets the first account register unless
  `ACCOUNT_REGISTRATION` is on.
- **`JWT_SECRET` left unset.** The drill set it, as the README recommends.

## Reproduced

Both verdicts twice, from an empty scratch directory each time, by running `run.sh` end
to end (2026-08-30 14:49 UTC and 14:52 UTC).
