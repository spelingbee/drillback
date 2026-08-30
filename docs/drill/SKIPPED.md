# What the drill did not test, and why

The session aimed at fifteen applications and finished fifteen, though not the fifteen
it set out to test. A skip is data. This page records every application from `docs/recipes-wanted.txt` that
the drill passed over on its way down the list, and which of three reasons applies. It
is deliberately blunt about the third one.

Star counts are from `docs/recipes-wanted.txt`, gathered 2026-08-30.

## Too big for the budget

Each of these needs more than the drill's per-application budget - about twenty minutes,
on hardware a CI runner would recognise - because of the number of services, the size of
the images, or both. Nothing here is a judgement about the software.

| App | Stars | What it takes to stand up |
|---|---|---|
| immich | 112,933 | Four services: server, machine-learning, PostgreSQL with a vector extension, Valkey. The ML image alone is several gigabytes, and a photo library has to be seeded through it before a backup means anything. **This is the highest-value untested application in the list**: its backup documentation is specific (a `pg_dumpall` plus the upload location) and it is exactly the sort of thing worth checking. |
| appwrite | 57,160 | More than ten containers - API, realtime, several workers, MariaDB, Redis. |
| reactive-resume | 41,925 | App, PostgreSQL, MinIO, and a browserless Chrome. |
| ToolJet | 40,790 | App, PostgreSQL, Redis; multi-gigabyte images. |
| khoj | 36,788 | App, PostgreSQL with pgvector, and model downloads. |
| langfuse | 33,910 | PostgreSQL, ClickHouse, MinIO, Redis. |
| signoz | 31,970 | ClickHouse plus collectors. |
| onyx | 31,827 | Multi-service with model downloads. |
| airbyte | 21,971 | A platform, not an application. |

`open-webui` nearly landed here too: its image is 7.16 GB and took about 35 minutes to
pull. It was tested anyway because it is second on the list, and the pull happened in the
background while other legs ran.

## Nothing a backup would be for

These have no user-created state worth restoring: the whole of the configuration is a
file the operator wrote by hand, usually kept in version control. A restore drill for
them would be testing whether a file copy copies a file.

| App | Stars | What its state is |
|---|---|---|
| glance | 36,715 | One `glance.yml`. Dashboards are defined in it; nothing is created at runtime. |
| homepage | 32,289 | A `config/` directory of YAML written by the operator. |
| dashy | 26,325 | One `conf.yml`, editable through the interface but still one file. |

This is worth saying plainly rather than leaving them off the list: "there is nothing to
restore" is a perfectly good answer, and it is the answer for a surprising share of the
self-hosted dashboards people run.

## Not reached

The honest category. These are feasible - most are one or two containers - and the drill
ran out of session, not out of capability. They are the obvious next legs, in this order:

| App | Stars | Why it is next |
|---|---|---|
| Stirling-PDF | 90,910 | **Attempted, and stopped short.** Fourth on the list, and its documentation has a *Database Backups* page describing automatic daily backups and an import path. The 3.38 GB image was pulled and the instance deployed; `/configs` came up holding `stirling-pdf-DB-2.3.232.mv.db`, `settings.yml` and a `backup/db` directory, which is a promising shape. The drill stopped because its login could not be driven from a script inside the budget: the v2 interface is a JavaScript client, `POST /login` answers 405, and HTTP Basic on every `/api/v1/user/...` endpoint answers 401. Without a way to seed an account or a setting through the application, a restore check cannot tell a restored instance from a fresh one - which is the drill's whole standard - so no verdict was recorded. Finding the login endpoint is probably ten minutes for somebody who reads the frontend bundle. |
| photoprism | 40,120 | App plus MariaDB, with real backup documentation and a `photoprism backup` command. |
| appsmith | 40,783 | One fat container with its own `appsmithctl export-db`. |
| plausible | 28,797 | PostgreSQL plus ClickHouse - heavier than the rest of this table, but its analytics data is the kind nobody can reconstruct. |
| karakeep | 28,665 | App, Chrome, Meilisearch. |
| ArchiveBox | 28,205 | A single container over a large data directory. |
| linkwarden | 19,629 | App plus PostgreSQL plus a file store: the exact shape that produced the listmonk finding. |
| docuseal, audiobookshelf, linkwarden's neighbours in the 13-19k range | - | All single-container SQLite applications; each is about twenty minutes. ConvertX was one of these and was tested in the end, after Stirling-PDF could not be. |

## What this means for the numbers

Every count in [summary.md](summary.md) is over the fifteen applications actually
tested. It is not a random sample of self-hosted software: it is the top of a
popularity list, filtered by what fits in a small budget, which skews it towards
single-container applications with SQLite databases. The findings that repeat across
those - the database is not the whole of the data, and the `-wal` is not the database -
are exactly the findings that a heavier, more service-oriented sample might not repeat.
