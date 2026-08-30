# Navidrome - exactly what was run

Everything below is in [`run.sh`](run.sh):

```sh
bash docs/drill/navidrome/run.sh
```

## 1. Deploy

[`compose.yaml`](compose.yaml) is the documented deployment plus the backup page's own
environment-variable example: `ND_BACKUP_PATH`, `ND_BACKUP_SCHEDULE`, `ND_BACKUP_COUNT`,
with `/data`, `/music` and `/backup` mounted.

Two deviations, recorded here:

| Deviation | Why |
|---|---|
| no published port | the drill drives Navidrome from a curl container on the project's network |
| `ND_SCANSCHEDULE: "@every 10s"` | not a default; the drill needs the seeded track in the library before the backup is taken |

The music is one file: `recipes/navidrome/test/drill-canary.mp3`, a one-second sine wave
generated with ffmpeg and tagged *Drill Operator / Restore Drill / Drill Canary Track*.

## 2. Seed through Navidrome's own API

```sh
curl -X POST http://navidrome:4533/auth/createAdmin \
  -d '{"username":"drilladmin","password":"Drill-Password-1"}'
curl "http://navidrome:4533/rest/createPlaylist?u=drilladmin&p=Drill-Password-1&v=1.16.1&c=restored-drill&f=json&name=drill-canary-playlist"
```

```text
createAdmin: 200
library scanned after 1 attempts
createPlaylist: 200
```

## 3. Back up, as documented

```sh
docker compose -p drill-navidrome run --rm navidrome backup create
```

```text
level=info msg="Backup complete" elapsed=12ms path=/backup/navidrome_backup_2026.08.30_12.33.06.db
```

A single 684 KB SQLite file. A copy of the whole `/data` folder is taken as well, and
used only as a control.

## 4. Into restic

The backup directory goes in at `/backup`, which is the path the backup page's own
environment-variable example uses; the control goes in at `/data`.

## 5. Tear down, then restore three ways

```sh
# A: the documented commands
restored check --recipe docs/drill/navidrome/recipe --source restic --from <repo-backup>

# A': the same, with the two undocumented conditions met
restored check --recipe docs/drill/navidrome/recipe-bootstrapped --source restic --from <repo-backup>

# control: a copy of /data
restored check --recipe recipes/navidrome --source restic --from <repo-data>
```

```text
  RESTORE UNUSABLE  1/4 checks   <- A
  RESTORE UNUSABLE  1/4 checks   <- A'
  PASS  6/6 checks               <- control
```

## 6. What was established by hand, outside run.sh

Two probes, quoted in [result.md](result.md):

- `navidrome backup restore -b <absolute path>` fails with an empty `backup path=`,
  while `ND_BACKUP_PATH=/backup ... -b <file name>` reports `Restore complete`;
- after that `Restore complete`, `POST /auth/createAdmin` answers 200 - Navidrome only
  offers to create the first administrator when it has no users - and a sign-in as the
  account that was in the backup answers 401.
