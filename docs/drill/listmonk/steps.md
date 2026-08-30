# listmonk - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/listmonk/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the project's own `docker-compose.yml` with the image
pinned to v6.2.0. Deviations, recorded here:

| Deviation | Why |
|---|---|
| no published ports | the drill drives listmonk from a curl container on the project's network |
| `LISTMONK_ADMIN_USER` / `_PASSWORD` set | the installation page documents these as the way to create the Super Admin during setup, and the drill needs a known account |

The `restart: unless-stopped` and the `--install --idempotent --yes` command are kept
exactly as the official file has them; the app does not wait for Postgres, it exits and
is restarted until the database is up.

## 2. Seed through listmonk's own admin session

`/api` wants an API user's token or an admin session. A person has the session, so that
is what the drill uses.

```sh
curl -X POST http://app:9000/admin/login -d 'username=drilladmin&password=Drill-Password-1'
curl -X POST http://app:9000/api/lists       -d '{"name":"drill-canary-list",...}'
curl -X POST http://app:9000/api/subscribers -d '{"email":"drill-canary@example.invalid",...}'
curl -X POST http://app:9000/api/media       -F 'file=@drill-canary.png;type=image/png'
```

```text
login: 302
create list: 200
list id: 3
create subscriber: 200
upload media: 200
```

and what listmonk put on disk for that one upload:

```text
-rw-r--r-- 1 root root  70 drill-canary.png
-rw-r--r-- 1 root root 814 thumb_drill-canary.png
```

## 3. Back up, as documented

The documentation names one thing, so the drill takes one thing:

```sh
docker compose -p drill-listmonk exec -T db sh -c 'pg_dump -U listmonk -d listmonk' > db.sql
```

```text
-rw-r--r-- 1 kadyr 197609 72667 db.sql
```

and, for the second reading, a copy of the uploads volume.

## 4. Into restic at the paths a real machine would have

`/srv/listmonk/db.sql` and `/srv/listmonk/uploads`. Reading A puts an **empty**
directory at the second path, which is what a snapshot of a machine that backed up only
the database actually contains.

## 5. Tear down, then restore each

```sh
restored check --recipe recipes/listmonk --source restic --from <repo-db-only>
restored check --recipe recipes/listmonk --source restic --from <repo-all>
```

Same recipe both times. Verdicts in [result.md](result.md).
