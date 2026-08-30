# Gogs - the official backup documentation

- **Application:** [gogs/gogs](https://github.com/gogs/gogs), 47,784 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 0.14.3, the current release on the day of the drill. Image
  `gogs/gogs:0.14.3`.
- **Documentation read:** 2026-08-30.

## Where the backup documentation is

Gogs has a documentation site again, at <https://gogs.io>, and it has a page that
documents the backup commands: the **CLI reference**,
<https://gogs.io/advancing/cli-reference>. Quoted in full, that is the whole of it:

> ## Backup and restore
>
> ```bash
> gogs backup
> gogs restore --from <archive>
> ```
>
> `backup` dumps the database, repositories, and related files into a single zip
> archive. `restore` imports everything back from an archive, which is useful for
> migrating Gogs to another server or switching database engines.
>
> Both commands support `--database-only` and `--exclude-repos` flags to narrow the
> scope. `backup` additionally supports `--exclude-mirror-repos` and `--target` to
> control where the archive is saved.

Worth recording, because it changes how a person finds this: the older documentation
URLs are gone. `https://gogs.io/docs` and `https://gogs.io/docs/intro/backup_and_restore`
both answer 404 (checked 2026-08-30), so search results and forum links pointing at the
old backup page lead nowhere, and the community threads that come up instead - notably
the maintainer's own "How to backup, restore and migrate" discussion - are what many
people will read.

## What the official image does with those commands

The `gogs/gogs` image ships a scheduled backup that calls exactly the documented
command. From `/app/gogs/docker/runtime/backup-job.sh` in the image:

```sh
BACKUP_ARGS="--target=${BACKUP_ARG_PATH}"
...
./gogs backup ${BACKUP_ARGS}
```

and `backup-init.sh` sets `BACKUP_PATH="/backup"`, creates it, and `chown git:git`s it,
driven by the `BACKUP_INTERVAL` and `BACKUP_RETENTION` environment variables. So `/backup`
holding `gogs-backup-<timestamp>.zip` is not a path this drill invented: it is what the
official image produces on a schedule if you turn the feature on.

## What the image documentation says to keep

`docker/README.md` in the repository documents the deployment:

```bash
docker run --name=gogs -p 10022:22 -p 10880:3000 -v /var/gogs:/data gogs/gogs
```

and, about upgrading:

> "Make sure you have volumed data to somewhere outside Docker container!"

One volume, `/data`, holding everything.

## The two readings

- **A: `gogs backup`, then `gogs restore --from`.** The documented commands, in the
  official image. This is the primary reading: it is what the documentation says, and
  what the image's own cron job does.
- **B: keep the `/data` volume.** What the deployment documentation implies.

Both were tested. See [result.md](result.md).
