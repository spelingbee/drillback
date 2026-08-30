# Memos - the official backup documentation

- **Application:** [usememos/memos](https://github.com/usememos/memos), 62,640 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 0.30.0, the current release on the day of the drill. Image
  `neosmemo/memos:0.30.0`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No. There is deployment guidance that tells you what to keep, and that is all.

Pages read:

- <https://usememos.com/docs/deploy/docker-compose>
- <https://usememos.com/docs/deploy/docker>
- <https://usememos.com/docs/faq>

## What the documentation says

The compose page gives this file, verbatim:

```yaml
services:
  memos:
    image: neosmemo/memos:stable
    container_name: memos
    restart: unless-stopped
    ports:
      - "5230:5230"
    volumes:
      - ./data:/var/opt/memos
    environment:
      MEMOS_PORT: 5230
      MEMOS_DRIVER: sqlite
      MEMOS_INSTANCE_URL: https://memos.example.com
```

and, on persistence:

> "keep your data directory on persistent storage"

> "back up both the database and any local assets if you do not use database-backed
> attachments"

The Docker page adds where that directory is:

> Memos stores persistent data in `/var/opt/memos` inside the container, and the host
> path `~/.memos` will contain the SQLite database and local assets.

The FAQ says:

> "Data is stored in your own database. Attachments can live in the database, on the
> local filesystem, or in S3-compatible storage."

There is no page that says *how* to take the backup: no command, no note about stopping
the container first, and no restore procedure.

## The ambiguity, stated plainly

"Back up both the database and any local assets" has two readings:

- **A: the data directory.** Copy `/var/opt/memos` - everything in it.
- **B: the database.** Copy the database, which a person reading that sentence would
  reasonably take to mean the file called `memos_prod.db`.

Both were tested. They do not give the same result, and the difference is not small:
see [result.md](result.md).
