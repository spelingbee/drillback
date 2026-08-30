# Gotify - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - the data directory | all of `/app/data` | **PASS** (exit 0, 6 of 6 checks) | [result.txt](result.txt) |
| B - "the database" | `gotify.db` on its own | **PARTIAL** (`RESTORE UNUSABLE`, exit 1, 5 of 6 checks) | [result-db-only.txt](result-db-only.txt) |

Gotify restores cleanly from its data directory. The finding is small, and it is exactly
the same shape as listmonk's: an uploaded file lives next to the database, not in it.

## Reading B, in full

```text
  CHECKS
  ok  health                    The server reports itself healthy
  ok  db-integrity              The SQLite database passes PRAGMA integrity_check
  ok  account-authenticates     The restored account can authenticate
  ok  applications-restored     The applications are in the restored instance -> 1 item
  ok  messages-restored         The message history is in the restored instance -> 1 item
  X   application-icon-on-disk  The uploaded application icon came back with its row
                                  expect  glob_min_count: 1 for *.png
                                  got     0 matches

  RESTORE UNUSABLE  5/6 checks
```

The verbatim report is in [result-db-only.txt](result-db-only.txt).

Everything a person would think of as their data is in `gotify.db`: the accounts, the
applications, their tokens, and the message history. What is not in it is the icon they
uploaded for an application. Gotify stores that as a file:

```text
/app/data/gotify.db
/app/data/images/mlWa3Pfzp_bEro2JciusCFAJM.png
/app/data/plugins/
```

and stores `image/mlWa3Pfzp_bEro2JciusCFAJM.png` in the `applications` row. Restore the
database without the directory and every custom icon in the interface is a broken image,
silently.

This is the mildest finding in the drill so far - an icon is not a message - but it is
the same mistake in the same place as three other applications here, and the honest
summary of it is that "the database" is almost never the whole answer.

## What the documentation would need to say instead

One line on the installation page:

> Back up the whole `/app/data` directory. It holds `gotify.db`, the uploaded
> application icons in `images/`, and any plugins.

## Not tested

- **Plugins.** `/app/data/plugins` was empty. It is inside reading A by construction.
- **MySQL and PostgreSQL dialects.** Only the documented SQLite default was tested; with
  an external database, `images/` is the entire on-disk state and the finding gets
  sharper.

## Reproduced

Both verdicts twice, from an empty scratch directory each time, by running `run.sh` end
to end (2026-08-30 12:52 UTC and 12:54 UTC).
