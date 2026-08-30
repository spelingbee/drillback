# listmonk - the official backup documentation

- **Application:** [knadh/listmonk](https://github.com/knadh/listmonk), 23,172 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** v6.2.0. Images `listmonk/listmonk:v6.2.0` and
  `postgres:17-alpine`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No. <https://listmonk.app/docs/> has sections for installation, upgrade, configuration,
maintenance/performance, querying, templating, roles, i18n, OIDC and the whole API - and
nothing about backups. `https://listmonk.app/docs/maintenance/backup/` answers 404.

## What the documentation does say

The word appears twice, both times as a warning attached to something else.

On the [upgrade page](https://listmonk.app/docs/upgrade/):

> **Warning:** Always take a backup of the Postgres database before upgrading listmonk

On the [installation page](https://listmonk.app/docs/installation/), about nightly
builds:

> Nightly releases are untested and may have bugs. Use at your own risk. Always take a
> backup of your Postgres database before using a nightly release.

Both name one thing: the Postgres database. Neither says how to take it, and neither
mentions anything else on disk.

## What the project's own compose file mounts

From <https://github.com/knadh/listmonk/raw/master/docker-compose.yml>, verbatim:

```yaml
    volumes:
      - ./uploads:/listmonk/uploads:rw   # Mount an uploads directory on the host to /listmonk/uploads inside the container.
                                         # To use this, change directory path in Admin -> Settings -> Media to /listmonk/uploads
```

So the project knows there is a directory of uploaded files. It is mounted in the file
people are told to download and run, and it is never mentioned where backups are.

Two things are worth being accurate about here, because the comment is misleading in a
way that makes the problem bigger rather than smaller:

- The default media provider is `filesystem`, and its default upload path is already
  `uploads` relative to the working directory - which in the image *is*
  `/listmonk/uploads`. Nothing has to be changed in Settings for files to start landing
  there. This drill uploaded an image through the API with a stock configuration and the
  file appeared in `/listmonk/uploads` immediately.
- listmonk also writes a thumbnail beside each upload, so the directory holds two files
  per image.

## The two readings

- **A: the Postgres database.** What the documentation names, twice, in bold.
- **B: the database and the uploads directory.**

Both were tested. See [result.md](result.md).
