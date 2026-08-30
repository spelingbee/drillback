#!/usr/bin/env bash
# The Navidrome leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/navidrome/run.sh
#
# Navidrome has a real backup page with a real scope note, so there is only one reading
# to test:
#
#   reading A - `navidrome backup create`, restored with `navidrome backup restore`,
#               which is what the backup page documents in both directions.
#
# A second snapshot of the /data folder is taken as a control, so that "the documented
# backup came back" can be told apart from "this drill's seeding worked".
source "$(dirname "$0")/../lib.sh"

drill_init navidrome

# One small generated MP3 - a 1-second sine with tags - so the library has something in
# it. It is the recipe's own test asset; ffmpeg is not needed to re-run this.
docker volume create "${PROJECT}_navidrome-music" >/dev/null
docker run --rm -v "${PROJECT}_navidrome-music:/music" \
  -v "$(docker_path "$REPO_ROOT/recipes/navidrome/test"):/seed:ro" \
  alpine:3.20 sh -c 'cp /seed/drill-canary.mp3 /music/ && ls -la /music'

drill_up "$APP_DIR/compose.yaml"
sleep 8

echo "-- seed: the admin account, a scanned library, a playlist"
drill_curl -f -o /dev/null -w 'createAdmin: %{http_code}\n' -X POST http://navidrome:4533/auth/createAdmin \
  -H 'content-type: application/json' \
  -d '{"username":"drilladmin","password":"Drill-Password-1"}'
Q="u=drilladmin&p=Drill-Password-1&v=1.16.1&c=drillback-drill&f=json"
for i in $(seq 1 30); do
  if drill_curl "http://navidrome:4533/rest/search3?$Q&query=Drill" | grep -q '"artist"'; then
    echo "library scanned after ${i} attempts"; break
  fi
  sleep 2
done
drill_curl -f -o /dev/null -w 'createPlaylist: %{http_code}\n' \
  "http://navidrome:4533/rest/createPlaylist?$Q&name=drill-canary-playlist"
drill_curl "http://navidrome:4533/rest/getPlaylists?$Q" | head -c 200
echo

echo "== backup: the documented command =="
# The backup page's docker instruction, verbatim in shape:
#   sudo docker compose run <service_name> backup create
drill_compose run --rm navidrome backup create 2>&1 | tail -2
mkdir -p "$BACKUP/backup" "$BACKUP/data"
docker run --rm -v "${PROJECT}_navidrome-backup:/src:ro" \
  -v "$(docker_path "$BACKUP/backup"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/backup"

echo "-- control: a copy of the whole /data folder, for comparison only"
docker run --rm -v "${PROJECT}_navidrome-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
du -sh "$BACKUP/data" "$BACKUP/backup"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-backup"; mkdir -p "$REPO"
drill_restic "$BACKUP/backup:/backup"
echo "== restore, reading A: navidrome backup restore =="
drill_check "$APP_DIR/recipe" "$WORK/restic-backup" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

echo "== restore, reading A-prime: the same file, with the two conditions the =="
echo "== documentation does not mention: a database already there, and       =="
echo "== ND_BACKUP_PATH set with --backup-file as a name inside it           =="
drill_check "$APP_DIR/recipe-bootstrapped" "$WORK/restic-backup"   "$APP_DIR/result-bootstrapped.json" "$APP_DIR/result-bootstrapped.txt"

REPO="$WORK/restic-data"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/data"
echo "== control: the same instance restored from a copy of /data =="
drill_check "$REPO_ROOT/recipes/navidrome" "$WORK/restic-data" \
  "$APP_DIR/result-data.json" "$APP_DIR/result-data.txt"
