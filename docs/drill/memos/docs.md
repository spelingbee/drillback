# Memos - the official backup documentation

- **Application:** [usememos/memos](https://github.com/usememos/memos), 62,640 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 0.30.0, the current release on the day of the drill. Image
  `neosmemo/memos:0.30.0`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

**Yes, and the drill missed it.** On 2026-08-30 this section said "No", on the strength
of three pages:

- <https://usememos.com/docs/deploy/docker-compose>
- <https://usememos.com/docs/deploy/docker>
- <https://usememos.com/docs/faq>

On 2026-09-04, while preparing a documentation PR, the session found
<https://usememos.com/docs/operations/backup-restore> - *Backup & Restore*, under
Operations, linked from a card on the documentation index page since March 2026. It
says to stop Memos and copy the whole data directory, or to use `sqlite3 .backup`,
"which handles WAL mode correctly", and it names the assets directory. That is the
right guidance, and the issue filed on the strength of the "No" above was corrected
and retitled the same day: <https://github.com/usememos/memos/issues/6271>.

What is still true: the verdict below was produced by following the deploy pages,
which are the pages a person deploying with Docker reads, and neither of them links
the Backup & Restore page. The compose page's production note says "back up both the
database and any local assets", and *the database* read alone is `memos_prod.db`.
That is the remaining gap, and it is one link and half a sentence.

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
