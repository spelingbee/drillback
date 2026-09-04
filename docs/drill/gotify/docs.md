# Gotify - the official backup documentation

- **Application:** [gotify/server](https://github.com/gotify/server), 15,815 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 3.1.0. Image `gotify/server:3.1.0`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No page, but a sentence - **and the drill missed it.** On 2026-08-30 this section said
that searching the installation page for "backup" returns nothing. It does not: since
2026-07-07 the [installation page](https://gotify.net/docs/install) has said

> `/app/data` contains the database file (if SQLite is used), application images and
> certificates (if Let's Encrypt is enabled). In this example the directory is mounted
> to `/var/gotify/data`, include this directory in your backups.

which is exactly the line the issue filed on 2026-09-04 asked for. The issue was
closed by its author the same day with an apology:
<https://github.com/gotify/website/issues/106>. The verdicts below stand - reading A
is what that sentence describes, and it restores everything; reading B is the
configuration reference's `data/gotify.db` read as "the database", and it loses the
icons - but the claim that the documentation never says which to keep was wrong.

## What the documentation does say

The [configuration reference](https://gotify.net/docs/config) names the database and its
default location:

```text
# Database connection string. Format depends on the dialect.
# Example:
#   sqlite3: path/to/database.db
# GOTIFY_DATABASE_CONNECTION=data/gotify.db
```

and, in the YAML form of the same page:

```yaml
database:
  dialect: sqlite3
  connection: data/gotify.db
```

The installation page mounts one directory:

```sh
docker run -p 80:80 -v /var/gotify/data:/app/data gotify/server
```

So the documentation names two things without ever saying which to back up: the
directory, and the file inside it.

## The two readings

- **A: the data directory** - `/app/data`, which the installation page mounts.
- **B: "the database"** - `gotify.db`, which the configuration reference names.

Both were tested. See [result.md](result.md).
