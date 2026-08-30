# Trilium - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/trilium/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented Docker deployment with the data
directory on a volume. Deviation: no published port.

## 2. Seed through Trilium's own setup and API

A fresh Trilium has no schema at all - it serves a setup page and waits to be told
whether this is a new document or a sync target.

```text
new-document: 204
set-password: 302
etapi token: 57 characters
create-note: 201
```

## 3. Back up, as documented

Settings -> Backup -> **Backup Now**, through the API that button uses:

```text
backup now: 204
-rw-r--r-- 1 node node 3231744 backup-now.db
```

Trilium writes it into `backup/`, inside the data directory.

## 4. Into restic, then restore

Reading A puts `/home/node/trilium-data/backup` into a snapshot on its own. The control
puts the whole data directory in.

```sh
restored check --recipe docs/drill/trilium/recipe --source restic --from <repo-backup>
restored check --recipe recipes/trilium          --source restic --from <repo-data>
```

The first recipe's restore service is the documented procedure, line for line:

```sh
rm -f document.db document.db-wal document.db-shm
cp /backup/backup-now.db document.db
chmod 600 document.db
```

```text
  PASS  3/3 checks   <- reading A
  PASS  4/4 checks   <- control
```

Verdicts in [result.md](result.md).
