#!/usr/bin/env bash
# The Trilium leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/trilium/run.sh
#
# Trilium has a real backup page, and the thing it documents is Trilium's own backup:
# copies of the database written into a `backup` directory once a day, once a week, once
# a month, and on demand. So:
#
#   reading A - backup/backup-now.db, restored by the alternative procedure the page
#               spells out (stop, delete document.db and its -wal/-shm, copy, chmod);
#   reading B - the whole data directory, as a control.
source "$(dirname "$0")/../lib.sh"

drill_init trilium
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://trilium:8080/api/health-check 180 '200'

echo "-- seed: initialise the database, set a password, create a note"
drill_api -o /dev/null -w 'new-document: %{http_code}\n' -X POST http://trilium:8080/api/setup/new-document \
  -H 'content-type: application/json' -d '{}'
sleep 3
drill_api -o /dev/null -w 'set-password: %{http_code}\n' -X POST http://trilium:8080/set-password \
  -H 'content-type: application/json' \
  -d '{"password1":"Drill-Password-1","password2":"Drill-Password-1"}'
TOKEN=$(drill_curl -X POST http://trilium:8080/etapi/auth/login \
  -H 'content-type: application/json' -d '{"password":"Drill-Password-1"}' |
  sed -e 's/.*"authToken":"//' -e 's/".*//')
echo "etapi token: ${#TOKEN} characters"
drill_curl -o /dev/null -w 'create-note: %{http_code}\n' -X POST http://trilium:8080/etapi/create-note \
  -H "Authorization: $TOKEN" -H 'content-type: application/json' \
  -d '{"parentNoteId":"root","title":"drill-canary-note","type":"text","content":"drill canary note: this line proves the restore"}'

echo "== backup: Trilium's own, which is what its backup page is about =="
# Settings -> Backup -> Backup Now, through the API the same button uses.
drill_curl -o /dev/null -w 'backup now: %{http_code}\n' -X PUT http://trilium:8080/etapi/backup/now \
  -H "Authorization: $TOKEN"
drill_compose exec -T trilium sh -c 'ls -la /home/node/trilium-data /home/node/trilium-data/backup'

mkdir -p "$BACKUP/backup" "$BACKUP/data"
docker run --rm -v "${PROJECT}_trilium-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
cp -a "$BACKUP/data/backup/." "$BACKUP/backup/"
ls -la "$BACKUP/backup"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-backup"; mkdir -p "$REPO"
drill_restic "$BACKUP/backup:/home/node/trilium-data/backup"
echo "== restore, reading A: the documented procedure over Trilium's own backup =="
drill_check "$APP_DIR/recipe" "$WORK/restic-backup" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-data"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/home/node/trilium-data"
echo "== control: the whole data directory =="
drill_check "$REPO_ROOT/recipes/trilium" "$WORK/restic-data" \
  "$APP_DIR/result-data.json" "$APP_DIR/result-data.txt"
