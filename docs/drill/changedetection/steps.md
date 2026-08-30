# changedetection.io - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/changedetection/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented `docker run` as compose, plus one extra
service: a one-page nginx for the watch to point at, so the drill does not measure
somebody else's website. Deviations, all recorded here:

| Deviation | Why |
|---|---|
| no published port | the drill drives it from a curl container on the project's network |
| an nginx `page` service | a target to watch that is not a real site |
| `ALLOW_IANA_RESTRICTED_ADDRESSES=true` | that target is on a private address; changedetection.io refuses those by default and names this variable in the error |

## 2. Seed through the API

The API key is generated on first start and lives in `changedetection.json`.

```text
api key: 32 characters
watch: 11f8c582-dfaa-4f7e-b3e6-c48c3199b9ea
recheck: 200
history written after 1 attempts
```

The watch's directory afterwards holds the snapshot, the compressed HTML and
`history.txt`.

## 3. Back up, as documented

Backups -> **Create backup**, through the link that button uses:

```text
request-backup: 200
backup written after 1 attempts
-rw-r--r-- 1 root root 119838 /datastore/changedetection-backup-20260830130942.zip
```

## 4. Into restic, then restore each

Reading A puts the zip alone into a snapshot at `/srv/changedetection-backups`. The
control puts `/datastore` in.

```text
  PASS  3/3 checks   <- reading A
  PASS  3/3 checks   <- control
```

Verdicts in [result.md](result.md).
