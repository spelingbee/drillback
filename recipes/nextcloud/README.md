# Nextcloud

The Nextcloud backup people take is the data directory and the database. The one they
need is the data directory, the database, **and `config/config.php`** - which holds
`passwordsalt`, `secret` and `instanceid`. Without those three values the files are
there and unreadable, and Nextcloud will cheerfully offer you a setup wizard and
install itself over the top of them.

That is the failure this recipe is built to catch.

## Inputs

| input | what it is | where it usually lives |
|---|---|---|
| `data` | `/var/www/html/data`: every user's files, plus `appdata_<instanceid>` | `/srv/nextcloud/data` |
| `config` | `/var/www/html/config`, holding `config.php` | `/srv/nextcloud/config` |
| `db` | a `pg_dump` of the Nextcloud database | `/srv/nextcloud/db.sql` |

Note what is **not** an input: `/var/www/html` itself. The application code comes from
the image. Restoring it would add about a gigabyte to every drill and prove nothing,
because a Nextcloud restored onto a different version of its own code is a Nextcloud
you would have to upgrade anyway.

## Mapping a typical install

**The official docker-compose.yml** mounts one volume at `/var/www/html`, so both
inputs are inside it:

    drillback check --recipe nextcloud \
      --input data=/var/lib/docker/volumes/nextcloud_nextcloud/_data/data \
      --input config=/var/lib/docker/volumes/nextcloud_nextcloud/_data/config \
      --input db=/srv/backups/nextcloud/db.sql

**A separate data volume** (`NEXTCLOUD_DATA_DIR=/srv/nextcloud-data`) is the layout
this recipe's defaults assume, give or take the prefix.

**Nextcloud AIO** manages its own volumes and takes its own backups with borg. This
recipe does not fit it; restoring an AIO backup is AIO's own command.

## Getting the dump

    docker compose exec -T db pg_dump --no-owner --no-acl -U nextcloud nextcloud \
      > /srv/backups/nextcloud/db.sql

`--no-owner --no-acl` matters more here than for most applications. Nextcloud's
installer notices when it was given a PostgreSQL superuser and creates a dedicated
role for itself called `oc_<your admin user>`, which then owns every table. A plain
dump carries `ALTER ... OWNER TO oc_admin` into a database where that role does not
exist and psql stops. If you restore without it, `drillback` will tell you exactly that:
hint `postgres/role-missing`.

## What restored changes about your instance, and why

A restore drill runs your Nextcloud somewhere it has never been, and two things in
your own configuration would stop it before any check could speak. The `prepare`
service in `compose.yaml` deals with both, on every run:

1. **Ownership.** Your backup records the uids of the machine it came from. Inside a
   fresh container the web server is uid 33 and those uids mean nothing, so the
   restored tree is handed to uid 33 with mode 0770 - the same
   `chown -R www-data:www-data` Nextcloud's own restore documentation asks for.
   Without it Nextcloud answers 503, "your data directory is not writable".

2. **A config overlay.** Nextcloud merges `config/*.config.php` over `config.php`, and
   the drill drops one in that names this run's hostname and this run's throwaway
   database. Your `config.php` names your domain and, very likely, that `oc_` database
   role. Neither is wrong; neither is reachable here. Reporting either as an unusable
   restore would be a false alarm, and false alarms are the thing this tool exists to
   remove.

Nothing else in the restored copy is touched, and the copy is destroyed when the run
ends.

## What the checks prove

| check | would it fail if the backup were empty? |
|---|---|
| `instance-installed` | **yes** - and this is the one that catches a missing `config.php` |
| `users-present` | **yes** |
| `storages-present` | **yes** |
| `config-php-present` | **yes**, if `config/` was left out of the backup |
| `login-page-renders` | no - an uninstalled Nextcloud still renders a page, and it says Nextcloud on it |
| `data-dir-not-empty` | no on an empty stack, because the bind mount exists; yes if `data/` was left out |

Three of the six fail on an empty stack, measured rather than assumed. The
`instance-installed` one is the one that matters: a Nextcloud with no `config.php` is
not a running Nextcloud at all, it is a setup wizard offering to install itself over
your files.

## Round trip

    drillback recipe test ./recipes/nextcloud

The harness installs Nextcloud with `occ maintenance:install`, which is the command
its own documentation gives for an unattended install, and which creates the schema,
`config.php`, and the admin account's files in one step.

Measured on the development machine: **1m53s**, both stages.
