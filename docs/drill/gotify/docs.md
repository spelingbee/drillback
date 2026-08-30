# Gotify - the official backup documentation

- **Application:** [gotify/server](https://github.com/gotify/server), 15,815 stars in
  `docs/recipes-wanted.txt` (gathered 2026-08-30).
- **Version tested:** 3.1.0. Image `gotify/server:3.1.0`.
- **Documentation read:** 2026-08-30.

## Is there a backup page?

No. <https://gotify.net/docs/> has installation, configuration, plugins, an API
reference and an FAQ. Searching the FAQ, the configuration page and the installation
pages for "backup" returns nothing.

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
