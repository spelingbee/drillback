# Beszel - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/beszel/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented hub compose file. Deviations: no
published port, `APP_URL` pointing at the service name rather than localhost, and the
documented `USER_EMAIL` / `USER_PASSWORD` variables so the drill has an account whose
password it knows.

## 2. Seed through Beszel's own API

```text
signed in as bq7oejkmho5yey7, token of 224 characters
create system: 200
```

## 3. Back up

```text
-rw-r--r-- 1 kadyr 197609   4096 data.db
-rw-r--r-- 1 kadyr 197609  32768 data.db-shm
-rw-r--r-- 1 kadyr 197609 716912 data.db-wal
```

Reading A is that directory. Reading B is `data.db` copied out of it on its own.

## 4. Into restic at /beszel_data, then restore each

```sh
restored check --recipe recipes/beszel --source restic --from <repo-all>
restored check --recipe recipes/beszel --source restic --from <repo-db-only>
```

```text
  PASS  5/5 checks               <- reading A
  RESTORE UNUSABLE  4/5 checks   <- reading B
```

Same recipe both times. Verdicts in [result.md](result.md).
