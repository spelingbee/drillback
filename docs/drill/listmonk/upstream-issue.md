# Draft issue for listmonk - NOT FILED

**Status: draft. Nothing has been filed (CLAUDE.md stop point 2).**

Where it would go: <https://github.com/knadh/listmonk/issues>.

---

**Title:** `Docs: "take a backup of the Postgres database" leaves every uploaded media file behind`

**Environment**

- listmonk v6.2.0, `listmonk/listmonk:v6.2.0` with `postgres:17-alpine`, deployed from
  the project's own `docker-compose.yml` with the image pinned.
- Stock configuration: media provider `filesystem`, upload path unchanged.

**What the documentation says**

There is no backup page. The word "backup" appears twice, both as warnings attached to
something else:

- upgrade page: "Always take a backup of the Postgres database before upgrading
  listmonk"
- installation page, on nightlies: "Always take a backup of your Postgres database
  before using a nightly release"

Both name the Postgres database and nothing else.

**What I did**

1. Started listmonk from the project's compose file.
2. Created a list and a subscriber, and uploaded one image through `/api/media`.
   With a stock configuration listmonk wrote two files:

   ```text
   /listmonk/uploads/drill-canary.png
   /listmonk/uploads/thumb_drill-canary.png
   ```

3. Took the documented backup: `pg_dump -U listmonk -d listmonk` (71 KiB).
4. Destroyed everything, restored the dump into a fresh Postgres, and started listmonk
   against it.

**What I observed**

Everything in the database came back - lists, subscribers, the admin account, settings,
and the row in `media` naming the uploaded file. The file itself is gone, so the media
library lists an image that cannot be loaded, and so does every campaign that used it.

**Why I think this is worth a documentation change**

The `media` table stores a file name; the file lives on disk. A person who follows the
documentation exactly has a backup that restores their database perfectly and loses
every image they have ever uploaded, with no error at any point - the restore looks
completely successful.

The project clearly knows the directory exists: `docker-compose.yml` mounts it. The
comment there says "To use this, change directory path in Admin -> Settings -> Media to
/listmonk/uploads", which reads as if the mount is optional - but the default upload
path already resolves to `/listmonk/uploads` inside the image, so files land there
without changing any setting.

**Suggested change**

A short backup section, or two lines wherever the existing warning appears:

> To back up listmonk, take a `pg_dump` of the Postgres database **and** a copy of the
> media uploads directory - `/listmonk/uploads` in the Docker image, or whatever
> Admin -> Settings -> Media names. The database stores a row per uploaded file; the
> files themselves are only on disk. (If you use the S3 provider, the files are in your
> bucket instead.)

It would also be worth putting that somewhere other than the upgrade page. People
upgrade far more often than they restore, and the backup they take before an upgrade is
the one they will still have when something else goes wrong.

I would be glad to send a docs PR for this if it would be welcome.
