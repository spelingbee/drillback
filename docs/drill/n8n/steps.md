# n8n - exactly what was run

Everything below is in [`run.sh`](run.sh), which is the file that was executed. Run it
from the repository root:

```sh
bash docs/drill/n8n/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented `docker run` command written as a
compose file: the `n8nio/n8n` image, a named volume on `/home/node/.n8n`, and the
environment variables the installation page sets.

Two deviations, neither of which touches what gets backed up:

| Deviation | Why |
|---|---|
| no `-p 5678:5678` | The drill drives n8n from a curl container attached to the project's own network. Nothing it runs is reachable from outside the machine. |
| `N8N_SECURE_COOKIE=false` | n8n's session cookie is `Secure` by default, so a client speaking plain HTTP is never given a session and every `/rest` call answers 401. n8n's own configuration reference documents this variable for exactly this case. |

```sh
docker compose -p drill-n8n -f docs/drill/n8n/compose.yaml up -d
```

## 2. Wait for it to be up - and note which endpoint says so

```sh
# NOT /healthz, and NOT /rest/settings. See result.md, finding 4.
curl http://n8n:5678/healthz/readiness    # from a container on the project network
```

## 3. Seed realistic data through n8n's own API

An owner account, one workflow, one credential. The owner is created from inside the
container so the call can be retried:

```sh
docker compose -p drill-n8n exec -T n8n node -e "<POST /rest/owner/setup, retrying>"
curl -X POST http://n8n:5678/rest/login       -d '{"emailOrLdapLoginId":"drill@example.invalid","password":"Drill-Password-1"}'
curl -X POST http://n8n:5678/rest/workflows   -d '{"name":"drill-canary-workflow", ...}'
curl -X POST http://n8n:5678/rest/credentials -d '{"name":"drill-canary-credential","type":"httpHeaderAuth","data":{"name":"X-Drill","value":"drill-secret-value"}}'
```

Observed:

```text
-- ready after 5s-ish: http://n8n:5678/healthz/readiness -> 200
owner account created
login: 200
create-workflow: 200
create-credential: 200
```

Not seeded: executions (n8n has no documented CLI to create one) and variables (an
enterprise feature). Both are recorded as untested in `result.md` rather than claimed.

## 4. Back up exactly as the documentation says

### Reading A - the documented export

```sh
docker compose -p drill-n8n exec -T n8n n8n export:workflow    --backup --output=/home/node/backups/latest/
docker compose -p drill-n8n exec -T n8n n8n export:credentials --backup --output=/home/node/backups/latest/
docker compose -p drill-n8n cp n8n:/home/node/backups/latest/. <workdir>/backup/export/
```

```text
Successfully exported 1 workflows.
Successfully exported 1 credentials.
total 5
-rw-r--r-- 1 kadyr 197609 1497 Aug 30 17:19 riwLpvvbxGDL47wy.json
-rw-r--r-- 1 kadyr 197609  436 Aug 30 17:19 xqqqbRokcCbMKAf6.json
```

The documentation's examples write `--output=backups/latest/`, a relative path. An
absolute one is used here only so the result can be copied out of the container; the
directory is the same single directory for both commands, which is what the examples do.

### Reading B - the directory the installation page says to keep

```sh
docker compose -p drill-n8n cp n8n:/home/node/.n8n/. <workdir>/backup/dotn8n/
```

```text
-rw-r--r-- 1 kadyr 197609      56 Aug 30 17:19 config
-rw-r--r-- 1 kadyr 197609 1519616 Aug 30 17:19 database.sqlite
-rw-r--r-- 1 kadyr 197609   32768 Aug 30 17:19 database.sqlite-shm
-rw-r--r-- 1 kadyr 197609 4136512 Aug 30 17:19 database.sqlite-wal
-rw-r--r-- 1 kadyr 197609    1935 Aug 30 17:19 n8nEventLog.log
drwxr-xr-x 1 kadyr 197609       0 Aug 30 17:19 nodes
drwxr-xr-x 1 kadyr 197609       0 Aug 30 17:19 storage
```

Taken with the container running, because nothing in the documentation says to stop it.

## 5. Into restic, at the paths a real machine would have

restic runs in a container with the backup bound at its own absolute path, so the
snapshot records `/home/node/.n8n` rather than a path inside this drill's scratch
directory. This is the same thing `restored recipe test` does (ADR-051).

```sh
docker run --rm -e RESTIC_PASSWORD=drill -v <repo>:/repo restic/restic:0.19.1 \
  --no-cache --repo /repo init --repository-version 2
docker run --rm -e RESTIC_PASSWORD=drill -v <repo>:/repo \
  -v <backup>/export:/home/node/backups/latest:ro restic/restic:0.19.1 \
  --no-cache --repo /repo backup --tag drill --host drill /home/node/backups/latest
```

```text
processed 2 files, 1.888 KiB in 0:00
snapshot af70d8e8 saved
...
processed 7 files, 5.427 MiB in 0:00
snapshot 649d6bab saved
```

## 6. Tear the seeded instance down, then restore

```sh
docker compose -p drill-n8n down -v --remove-orphans

RESTIC_PASSWORD=drill restored check --recipe docs/drill/n8n/recipe \
  --source restic --from <repo-export>  --report docs/drill/n8n/result-export.json
RESTIC_PASSWORD=drill restored check --recipe recipes/n8n \
  --source restic --from <repo-dotn8n>  --report docs/drill/n8n/result-dotn8n.json
```

The verdicts are in [`result.md`](result.md); the full reports are
[`result-export.txt`](result-export.txt) / [`result-export.json`](result-export.json)
and [`result-dotn8n.txt`](result-dotn8n.txt) / [`result-dotn8n.json`](result-dotn8n.json).
