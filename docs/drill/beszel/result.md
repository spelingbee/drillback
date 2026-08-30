# Beszel - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the data directory | all of `/beszel_data` | **PASS** (exit 0, 5 of 5 checks) | [result.txt](result.txt) |
| B - "the database" | `data.db` on its own | **PARTIAL** (`RESTORE UNUSABLE`, exit 1, 4 of 5 checks) | [result-db-only.txt](result-db-only.txt) |

Beszel restores completely from its data directory. Reading B is a repeat of the Memos
finding with a twist, and the twist is the interesting part.

## The numbers, again

Immediately after creating one account and one monitored system:

```text
-rw-r--r-- 1 kadyr 197609   4096 data.db
-rw-r--r-- 1 kadyr 197609  32768 data.db-shm
-rw-r--r-- 1 kadyr 197609 716912 data.db-wal
```

4 KiB in the file called `data.db`; 700 KiB in the file next to it. PocketBase runs
SQLite in WAL mode, and on a young instance essentially everything is in the `-wal`.

## Reading B, in full

```text
  CHECKS
  ok  serves-ui        The interface is served
  ok  db-integrity     The PocketBase database passes PRAGMA integrity_check -> ok
  ok  users-present    The accounts survived the restore -> 1
  X   systems-present  The monitored systems survived the restore
                         query   SELECT count(*) FROM systems WHERE name = 'drill-canary-system';
                         expect  scalar_int_min: 1
                         got     0
  ok  signin-works     The restored account can still sign in

  RESTORE UNUSABLE  4/5 checks
```

The verbatim report is in [result-db-only.txt](result-db-only.txt).

## The twist: four of the five checks pass, and the instance is empty

Look at which checks passed. `users-present` says there is an account. `signin-works`
says that account signs in with the password from before the restore. Both are true, and
neither has anything to do with the backup: the hub found an empty database, applied the
documented `USER_EMAIL` and `USER_PASSWORD` environment variables, and made a brand new
account with the same address and the same password.

A person restoring this way logs in successfully, on the first try, with their own
credentials - and finds no systems. Everything about the experience says the restore
worked.

The only check that could tell the difference is the one that asks for a row that only
the backup could have supplied. That is what `systems-present` is for, and it is why the
recipe names a specific system rather than counting rows.

## What the documentation would need to say

There is no backup page at all, so the smallest useful addition is a short one:

> Back up the whole `beszel_data` directory. It holds PocketBase's `data.db` and
> `auxiliary.db` together with their `-wal` files, and on a running hub most of your
> data is in the `-wal` until SQLite checkpoints it. Copying `data.db` alone gives you a
> valid, empty database. Stop the hub before copying, or use the backup feature in
> Settings.

The last clause matters too: the application has a backup feature that the documentation
never mentions.

## Not tested

- **Beszel's own backup feature** (PocketBase backups, including to S3). It is not in
  the documentation, so it is not a documented reading; it would be worth a drill of its
  own.
- **Agents and real metrics.** The seeded system is a row with `status: pending`; no
  agent ever connected, so `system_stats` and `container_stats` are empty. Reading A
  covers them by construction.

## Reproduced

Both verdicts twice, from an empty scratch directory each time, by running `run.sh` end
to end (2026-08-30 13:15 UTC and 13:17 UTC).
