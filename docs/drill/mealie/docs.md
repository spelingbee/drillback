# Mealie - the official backup documentation

- **Application:** [mealie-recipes/mealie](https://github.com/mealie-recipes/mealie),
  13,095 stars in `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v3.24.0. Image `ghcr.io/mealie-recipes/mealie:v3.24.0`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

Yes:
<https://docs.mealie.io/documentation/getting-started/usage/backups-and-restoring/>,
titled *Backups and Restores*. It covers both directions, which puts it in the better
half of this drill.

## What it says

On the integrated feature:

> Mealie provides an integrated mechanic for doing full installation backups of the
> database. Navigate to Settings > Admin Settings > Backups or manually by adding
> `/admin/backups` to your instance URL.
>
> From this page, you will be able to: See a list of available backups / Create a backup
> / Upload a backup / Delete a backup / Download a backup / Perform a restore

and then, in a tip box, the sentence this drill tested:

> If you're using Mealie with SQLite all your data is stored in the `/app/data/` folder
> in the container. You can easily perform entire site backups by stopping the
> container, and backing up this folder with your chosen tool. This is the **best** way
> to backup your data.

On restoring the integrated backup:

> To restore from a backup it needs to be uploaded to your instance which can be done
> through the web portal.

with three warnings that deserve quoting, because they are the honest kind:

> - This is a destructive action and will delete all data in the database
> - This action cannot be undone
> - If this action is successful you will be logged out and you will need to log back in

## The reading

The page recommends one thing in bold - stop the container, copy `/app/data` - so that
is the reading the drill tested. The integrated backup is recorded as untested, with the
reason, in [result.md](result.md).
