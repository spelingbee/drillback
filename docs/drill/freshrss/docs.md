# FreshRSS - the official backup documentation

- **Application:** [FreshRSS/FreshRSS](https://github.com/FreshRSS/FreshRSS), 15,873
  stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 1.29.1. Image `freshrss/freshrss:1.29.1`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

Yes: <https://freshrss.github.io/FreshRSS/en/admins/05_Backup.html>, a numbered chapter
in the administrator manual, sitting between *Updating* and *Linux install*. It is the
most precise backup page in this drill.

## What it says

It opens with a list of what to keep, and - unusually - what not to:

> **What to back up**
>
> - `./data/` - **required**. You can skip `cache/`; FreshRSS rebuilds it.
> - `./extensions/` - **recommended** if you use third-party extensions.
> - `./i/themes/` - **optional**, only if you have added custom themes.
> - **External database** (MySQL, MariaDB, PostgreSQL) - back up separately with
>   `./cli/db-backup.php` (portable SQLite per user) or `mysqldump` / `pg_dump`. SQLite
>   is covered by `./data/` above.
>
> All other folders belong to the source code and are restored by a fresh install or
> upgrade.

Four lines, each with a word saying how much it matters, and a sentence saying which of
the remaining folders you can ignore and why. Nothing else in this drill comes close.

It then gives the commands, in both directions - a per-user SQLite dump with
`./cli/db-backup.php`, a `tar -czf` of the installation directory, the matching
`tar -xzf`, and `./cli/db-restore.php --delete-backup --force-overwrite` - with:

> It is safer to stop your web server and cron during maintenance operations.

and documents an automatic periodic SQLite export with a retention count, configured in
`./data/config.php` under `auto_sqlite_export` and scheduled with
`./cli/export-sqlite-auto.php`.

## The reading

The page's own first sentence contains two claims worth testing: that `./data/` is what
you need, and that `cache/` inside it can be skipped. Both were tested. See
[result.md](result.md).
