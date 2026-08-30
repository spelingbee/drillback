# Trilium - the official backup documentation

- **Application:** [TriliumNext/Trilium](https://github.com/TriliumNext/Trilium),
  37,632 stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v0.105.0. Image `triliumnext/trilium:v0.105.0`.
- **Documentation read:** 2026-08-30, from
  `docs/User Guide/User Guide/Installation & Setup/Backup.md` in the repository, which
  is what the documentation site is built from.

## Is there a backup page?

Yes, and it is the most complete one in this drill. It covers what Trilium backs up by
itself, how to make a backup on demand, how to download one, and - unusually - two
different ways to put one back.

## What it says

On the automatic backups:

> Trilium supports simple backup scheme where it saves copy of the Database on these
> events:
>
> *   once a day
> *   once a week
> *   once a month
> *   before DB migration to newer version
>
> So in total you'll have at most 4 backups from different points in time which should
> protect you from various problems. These backups are stored by default in `backup`
> directory placed in the data directory.

and, immediately after, the sentence that makes this page honest:

> This is only very basic backup solution, and you're encouraged to add some better
> backup solution - e.g. backing up the Database to cloud / different computer etc.

On making one now:

> Go to **Settings -> Backup** and press the **Backup Now** button.

On restoring, the supported way:

> **For a new Trilium instance**: When setting up a new Trilium instance, use the
> "Restore from backup" option in the setup menu to guide you through restoring your
> existing backup.

and the way this drill tested, quoted in full because it is the one a person on a
command line will use:

> * find the data directory Trilium uses
> * find `~/trilium-data/backup/backup-weekly.db` - this is the Database backup.
> * at this point stop/kill Trilium
> * delete `~/trilium-data/document.db`, `~/trilium-data/document.db-wal` and
>   `~/trilium-data/document.db-shm` (latter two files are auto generated)
> * copy and rename this `~/trilium-data/backup/backup-weekly.db` to
>   `~/trilium-data/document.db`
> * make sure that the file is writable, e.g. with `chmod 600 document.db`
> * start Trilium again

That instruction names `document.db-wal` and `document.db-shm` explicitly. Two other
applications in this drill lose data precisely because nobody told the reader those
files exist.

## The reading

One: Trilium's own backup, restored by the procedure above. A copy of the whole data
directory was taken as a control. See [result.md](result.md).
