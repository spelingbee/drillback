# Paperless-ngx

Paperless is two halves that have to survive together: a PostgreSQL database that says
what every document is, and a media directory that holds the documents themselves.
Restoring one without the other gives you either an archive of filenames pointing at
nothing, or a pile of PDFs with no idea what is in them.

## Inputs

| input | what it is | where it usually lives |
|---|---|---|
| `media` | `/usr/src/paperless/media`: originals, archived PDF/A copies, thumbnails | `/srv/paperless/media` |
| `data` | `/usr/src/paperless/data`: the search index and the trained classifier | `/srv/paperless/data` |
| `db` | a `pg_dump` of the Paperless database | `/srv/paperless/db.sql` |

`media` is the irreplaceable one. `data` can be rebuilt with
`document_index reindex`, but a restore that leaves it behind starts with an empty
search box, so it is treated as required.

## Mapping a typical install

**The upstream docker-compose.postgres.yml** uses named volumes:

    volumes:
      - data:/usr/src/paperless/data
      - media:/usr/src/paperless/media

so the host paths are under `/var/lib/docker/volumes/`:

    drillback check --recipe paperless-ngx \
      --input media=/var/lib/docker/volumes/paperless_media/_data \
      --input data=/var/lib/docker/volumes/paperless_data/_data \
      --input db=/srv/backups/paperless/db.sql

**A bind-mount install** points the same two at directories you chose, which is the
easy case:

    drillback check --recipe paperless-ngx \
      --input media=/opt/paperless/media --input data=/opt/paperless/data

**If you use `document_exporter`** instead of `pg_dump`, this recipe is the wrong
shape for your backup. The exporter writes a single directory containing a manifest
and the original files, and restoring it means `document_importer`, not psql. That is
a different recipe, and a good one to contribute: see CONTRIBUTING.md.

## Getting the dump

    docker compose exec -T db pg_dump -U paperless paperless > /srv/backups/paperless/db.sql

Custom format works too; `drillback` detects it from the file's magic bytes rather than
its extension:

    docker compose exec -T db pg_dump -Fc -U paperless paperless > /srv/backups/paperless/db.dump

## What the checks prove

| check | would it fail if the backup were empty? |
|---|---|
| `login-page-renders` | no - Paperless serves the same login page against an empty database |
| `users-present` | **yes** |
| `superuser-present` | **yes** |
| `media-dir-present` | **yes**, if the media directory was missed entirely |
| `documents-consistent` | no, but it catches document rows whose file reference was lost |

A fresh Paperless creates two accounts of its own, `consumer` and `AnonymousUser`,
before anybody signs up. `users-present` excludes them by name for that reason - it
was measured on an empty stack, not assumed - and it is why this recipe's compose file
deliberately does not set `PAPERLESS_ADMIN_USER`.

## What this recipe does not prove

The round trip does not drive a document through consumption. Seeding one means either
a multipart upload or a file dropped into the consume directory and then waited for,
and the harness's step vocabulary has neither. So the harness proves the database and
the media directory come back; it does not prove that Paperless can re-OCR and re-serve
a document you gave it three years ago. That is on the roadmap
(`docs/roadmap.md`), and it is a good contribution.

`documents-consistent` is the partial answer: it fails if any document row lost the
filename that points at the file on disk, which is the shape a database-only restore
takes.

## Round trip

    drillback recipe test ./recipes/paperless-ngx

Measured on the development machine: **3m11s**, both stages. Paperless's first-start
migrations are most of it.
