# Gitea

Gitea keeps its repositories as bare git directories on disk and everything else -
users, issues, pull requests, permissions - in a database. Restore one without the
other and you get either a forge with no code in it, or a directory of repositories
nobody can reach.

## Inputs

| input | what it is | where it usually lives |
|---|---|---|
| `data` | the whole `/data` directory, including `git/repositories/` | `/srv/gitea/data` |
| `db` | a `pg_dump` of the Gitea database | `/srv/gitea/db.sql` |

## Mapping a typical install

**The docker-compose.yml from Gitea's own docs** mounts `./gitea` at `/data`:

    restored check --recipe gitea --input data=/opt/gitea/gitea

**A named volume** (`gitea:/data`):

    restored check --recipe gitea --input data=/var/lib/docker/volumes/gitea/_data

**A package install** (not docker) splits what the container keeps in one place:
repositories under `/var/lib/gitea/data/gitea-repositories`, config in
`/etc/gitea/app.ini`. This recipe expects the container layout, where repositories are
at `/data/git/repositories`. If yours differ, `--input data=` at the directory whose
child is `git/repositories` is usually the right answer.

## Getting the dump

    docker compose exec -T db pg_dump -U gitea gitea > /srv/gitea/db.sql

Gitea also has `gitea dump`, which writes a zip of everything at once. That is a fine
backup and a different shape from this recipe - a `gitea-dump` recipe that unpacks it
would be a good contribution.

## What the checks prove

| check | would it fail if the backup were empty? |
|---|---|
| `web-ui-renders` | no - Gitea serves the same home page against an empty database |
| `repos-in-db` | **yes** |
| `users-in-db` | **yes** |
| `repo-files-on-disk` | **yes** |
| `api-lists-repos` | **yes** |

`repos-in-db` and `repo-files-on-disk` are the pair that matters: the first says the
database knows about a repository, the second says the bare repository is actually
there. A backup that captured the database and skipped the data directory passes the
first and fails the second, which is exactly the report you want.

## Round trip

    restored recipe test ./recipes/gitea

The harness creates an admin user with `gitea admin user create` and a repository
through the API, both of which are things a person does through Gitea's own front door.

Measured on the development machine: **1m19s**, both stages.
