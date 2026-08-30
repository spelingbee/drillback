# Memos - exactly what was run

Everything below is in [`run.sh`](run.sh), which is the file that was executed:

```sh
bash docs/drill/memos/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented compose file. Two deviations, recorded
here, neither touching what gets backed up:

| Deviation | Why |
|---|---|
| no `ports: 5230:5230` | The drill drives Memos from a curl container on the project's own network. |
| `MEMOS_INSTANCE_URL` left empty | The documentation itself gives that as the setting for a private deployment, and this one has no route out. |

```sh
docker compose -p drill-memos -f docs/drill/memos/compose.yaml up -d
```

## 2. Seed through Memos' own API

The first account created on a fresh instance becomes the host admin.

```sh
curl -X POST http://memos:5230/api/v1/users \
  -H 'content-type: application/json' \
  -d '{"username":"drill","password":"Drill-Password-1","role":"HOST"}'

curl -X POST http://memos:5230/api/v1/auth/signin \
  -H 'content-type: application/json' \
  -d '{"passwordCredentials":{"username":"drill","password":"Drill-Password-1"}}'

curl -X POST http://memos:5230/api/v1/memos \
  -H "authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"content":"drill-canary-memo: this line proves the restore","visibility":"PRIVATE"}'
```

```text
-- ready after 1s-ish: http://memos:5230/healthz -> 200
create-user: 200
signin: token of 303 characters
create-memo: 200
```

## 3. Back up, as documented

The documentation says to keep `/var/opt/memos` and to "back up both the database and
any local assets". It does not say to stop the container, so the copy is taken with
Memos running.

```sh
docker run --rm -v drill-memos_memos-data:/src:ro -v <backup>/data:/dst \
  alpine:3.20 sh -c 'cp -a /src/. /dst/'
```

```text
-rw-r--r-- 1 kadyr 197609   4096 Aug 30 17:38 memos_prod.db
-rw-r--r-- 1 kadyr 197609  32768 Aug 30 17:38 memos_prod.db-shm
-rw-r--r-- 1 kadyr 197609 160712 Aug 30 17:38 memos_prod.db-wal
```

Those three numbers are the whole finding, and they are worth reading twice. The file
called `memos_prod.db` is 4 KiB. The file called `memos_prod.db-wal` is 160 KiB.

### Reading B - "the database"

The second reading is one `cp` of one file:

```sh
cp <backup>/data/memos_prod.db <backup>/db-only/
```

## 4. Into restic, at the documented path

```sh
docker run --rm -e RESTIC_PASSWORD=drill -v <repo>:/repo restic/restic:0.19.1 \
  --no-cache --repo /repo init --repository-version 2
docker run --rm -e RESTIC_PASSWORD=drill -v <repo>:/repo \
  -v <backup>/data:/var/opt/memos:ro restic/restic:0.19.1 \
  --no-cache --repo /repo backup --tag drill --host drill /var/opt/memos
```

```text
processed 3 files, 192.945 KiB in 0:00
snapshot 1c10f613 saved
```

and for reading B, the same command over the one-file directory:

```text
processed 1 files, 4.000 KiB in 0:00
snapshot 0969cf23 saved
```

## 5. Tear down, then restore each

```sh
docker compose -p drill-memos down -v --remove-orphans

RESTIC_PASSWORD=drill restored check --recipe recipes/memos \
  --source restic --from <repo>         --report docs/drill/memos/result.json
RESTIC_PASSWORD=drill restored check --recipe recipes/memos \
  --source restic --from <repo-db-only> --report docs/drill/memos/result-db-only.json
```

Same recipe both times. The only difference between the two runs is which files are in
the snapshot. Verdicts in [result.md](result.md).
