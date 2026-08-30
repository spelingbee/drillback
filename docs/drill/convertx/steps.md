# ConvertX - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/convertx/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the README's compose file with the image pinned to
v0.18.0. Deviations, both recorded here:

| Deviation | Why |
|---|---|
| no published port | the drill drives ConvertX from a curl container on the project's network |
| `HTTP_ALLOWED=true` | the README's own warning: needed when the service is not reached over localhost or https, which is every request the drill makes |

## 2. Seed through ConvertX's own pages

The first account is created through the setup page, which is what "visit
`http://localhost:3000` and create your account" means. The home page has to be loaded
before the upload: ConvertX creates the job there, and the upload lands in
`uploads/<user>/<job>/`.

```text
register: 302
home: 200
upload: 200
convert: 500
```

The `convert: 500` is honest and is left in - see *Not tested* in
[result.md](result.md). What the drill carries forward is the uploaded file and its job
row:

```text
/app/data/uploads/1/1/drill-canary.png
```

## 3. Back up

```text
-rw-r--r-- 1 root root 20480 mydb.sqlite
-rw-r--r-- 1 root root 32768 mydb.sqlite-shm
-rw-r--r-- 1 root root 16512 mydb.sqlite-wal
```

Reading A is the directory. Reading B is `mydb.sqlite` copied out of it on its own.

## 4. Into restic at /app/data, then restore each

```sh
restored check --recipe recipes/convertx --source restic --from <repo-all>
restored check --recipe recipes/convertx --source restic --from <repo-db-only>
```

```text
  PASS  6/6 checks               <- the data directory
  RESTORE UNUSABLE  2/6 checks   <- mydb.sqlite alone
```

Same recipe both times. Verdicts in [result.md](result.md).
