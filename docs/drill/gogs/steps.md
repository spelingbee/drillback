# Gogs - exactly what was run

Everything below is in [`run.sh`](run.sh), which is the file that was executed:

```sh
bash docs/drill/gogs/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented `docker run` written as compose: the
`gogs/gogs` image with one volume on `/data`. One deviation, recorded here: no published
ports, because the drill drives Gogs from a curl container on the project's own network
and never uses SSH.

## 2. Seed through the forms a person uses

Gogs has no CLI that can install an instance or create a repository, so the drill drives
the same requests a browser makes: the installer, a login, the new-repository form with
`auto_init` so there is a commit on disk, and an avatar upload.

```text
get install: 200
post install: 500
login: 302
get repo form: 200
create repo: 302
avatar png written
post avatar: 302
repo page: 200
```

Two notes on that output, both true and both worth writing down:

- `post install: 500` is what the installer answers while it is finishing; the
  instance is installed regardless - `/install` answers 404 afterwards and the admin
  account is in the database. The drill checks the outcome (`repo page: 200`) rather
  than trusting a status code.
- the avatar is a 70-byte 1x1 PNG. It is there to be a file that lives outside both the
  database and the repositories.

## 3. Back up, reading A: the documented command

`/backup` is where the image's own scheduled backup writes, and `backup-init.sh` chowns
it to `git` before running; the drill does the same two things.

```sh
docker compose -p drill-gogs exec -T gogs sh -c 'mkdir -p /backup && chown git:git /backup'
docker compose -p drill-gogs exec -T -u git gogs sh -c 'cd /app/gogs && ./gogs backup --target=/backup'
```

```text
[TRACE] Skipping "data" directory in custom directory
[ INFO] Dumping repositories in "/data/git/gogs-repositories"
[ INFO] Repositories dumped to: /tmp/gogs-backup-.../repositories.zip
[ INFO] Backup succeed! Archive is located at: /backup/gogs-backup-20260830120140.zip
```

```text
-rw-r--r-- 1 kadyr 197609 23054 gogs-backup-20260830120140.zip
```

and what is in it:

```text
gogs-backup/metadata.ini
gogs-backup/repositories.zip
gogs-backup/custom/conf
gogs-backup/custom/log
gogs-backup/data/avatars
gogs-backup/db/<38 tables>.json
```

## 4. Back up, reading B: the documented volume

```sh
docker compose -p drill-gogs cp gogs:/data/. <backup>/data/
```

```text
466K  <backup>/data
```

## 5. Into restic, at the paths a real machine would have

```text
processed 1 files, 22.512 KiB in 0:00     # the archive, at /backup
snapshot 1a464fbc saved
processed 40 files, 405.512 KiB in 0:00   # the volume, at /data
snapshot 81fd3b45 saved
```

## 6. Tear down, then restore each

```sh
docker compose -p drill-gogs down -v --remove-orphans

restored check --recipe docs/drill/gogs/recipe --source restic --from <repo-archive>
restored check --recipe recipes/gogs          --source restic --from <repo-data>
```

The first recipe's `restore` service runs the documented command and nothing else:

```sh
cd /app/gogs
gosu git ./gogs restore --from=/backup/$(cd /backup && ls *.zip | head -1)
```

Its failure is left in the service log rather than aborting the run, so the drill can
still ask what state the instance was left in. Verdicts in [result.md](result.md).

## 7. What was established by hand, outside `run.sh`

Three follow-ups, each a single `docker run` against the same archive, to find out
whether the failure in step 6 was the whole story. The commands and their output are
quoted in [result.md](result.md) under *Root cause*; in summary they establish that
passing `--config`, making `/app/gogs` writable, and moving `TMPDIR` onto the volume
each move the failure without removing it.
