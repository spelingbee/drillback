# FreshRSS - result

| Reading | What was backed up | Verdict | Report |
|---|---|---|---|
| A - `./data` and `./extensions` | 1.3 MiB and 8 KiB | **PASS** (exit 0, 5 of 5 checks) | [result.txt](result.txt) |
| B - the same, without `cache/` | 581 KiB and 8 KiB | **PASS** (exit 0, 5 of 5 checks) | [result-no-cache.txt](result-no-cache.txt) |

Both halves of the page's sentence are true. `./data/` is what you need, and `cache/`
can be skipped - it was 55% of the directory on an instance with two feeds.

## Reading B, in full

```text
  inputs     data        /var/www/FreshRSS/data                             581.0 KiB
             extensions  /var/www/FreshRSS/extensions                         8.0 KiB
             db          /var/www/FreshRSS/data/users/drilladmin/db.sqlite   508.0 KiB

  CHECKS
  ok  serves-ui             The interface is served
  ok  db-integrity          The user database passes PRAGMA integrity_check -> ok
  ok  feeds-present         The feed that was backed up survived the restore -> 1
  ok  entries-present       The articles survived the restore -> 11
  ok  user-config-present   The user's own configuration came back

  PASS  5/5 checks
```

The verbatim reports are in [result.txt](result.txt) and
[result-no-cache.txt](result-no-cache.txt).

The numbers are the point:

```text
1.3M   data
581K   data without cache/
8.0K   extensions
```

On a two-feed instance, more than half of the directory the page calls "required" is a
cache the same page tells you to leave out. On a real instance with hundreds of feeds
and years of favicons, that ratio is worth knowing.

`entries-present` returns 11 rather than 1 because the image's own first start subscribes
to the FreshRSS releases feed and fetches it. That is the image's behaviour, not the
drill's, and it is left in: an instance carrying a feed nobody asked for is a more honest
thing to restore than a synthetic one.

## Why `entries-present` and not just `feeds-present`

A feed list comes back from an OPML export in a minute, and FreshRSS has one built in.
What does not come back is the archive: which articles the instance has seen, which were
read, which were starred. That is what `entry` holds, and it is what a backup is for.

## Not tested

- **`./i/themes/`.** No custom theme was installed, so the "optional" line was not
  exercised.
- **The external-database path.** Only the SQLite default was tested, which is the case
  the page says `./data/` covers.
- **`db-backup.php` / `db-restore.php`.** The per-user SQLite export the page documents
  for migrations was not driven end to end; the drill tested the file-level backup the
  same page describes.
- **`auto_sqlite_export`.** Documented, not tested.

## Reproduced

Both verdicts are PASS, so the two-run rule for failures does not apply. The leg was run
once end to end (2026-08-30 13:28 UTC), and `restored recipe test recipes/freshrss` passes
both stages independently.
