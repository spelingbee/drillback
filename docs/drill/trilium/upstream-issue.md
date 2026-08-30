# Trilium - no issue drafted

**Status: nothing to file.**

Trilium is the first application in this drill whose documented backup and documented
restore both work exactly as written. There is no gap to report, so no draft issue is
proposed, and nothing has been filed.

Two observations were recorded in [result.md](result.md) rather than sent upstream,
because the backup page already makes both of them itself:

- the automatic backups live in `backup/`, inside the data directory, on the same disk
  as the database - and the page already says "This is only very basic backup solution,
  and you're encouraged to add some better backup solution";
- a fresh instance has no `backup/` directory at all until the first daily copy or a
  manual **Backup Now** - which follows from "once a day / once a week / once a month"
  and does not need saying twice.

If anything were worth sending, it would be a compliment rather than an issue: the
alternative restore procedure names `document.db-wal` and `document.db-shm` explicitly,
and two other applications in this drill lose data precisely because nobody told the
reader those files exist.
