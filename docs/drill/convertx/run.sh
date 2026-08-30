#!/usr/bin/env bash
# The ConvertX leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/convertx/run.sh
#
# ConvertX's README never mentions backups. What it does is mount one volume, so the
# drill tests the two readings that produces:
#
#   reading A - the /app/data directory;
#   reading B - "the database", meaning mydb.sqlite.
source "$(dirname "$0")/../lib.sh"

drill_init convertx
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://convertx:3000/ 120 '302'

echo "-- seed: create the first account, then upload a file for conversion"
mkdir -p "$WORK/curl"
cp "$REPO_ROOT/recipes/convertx/test/drill-canary.png" "$WORK/curl/drill-canary.png"
drill_api -o /dev/null -w 'register: %{http_code}\n' -X POST http://convertx:3000/register \
  -d 'email=drill@example.invalid&password=Drill-Password-1'
# The job is created when the page is loaded; the upload lands in uploads/<user>/<job>/.
drill_api -o /dev/null -w 'home: %{http_code}\n' http://convertx:3000/
drill_api -o /dev/null -w 'upload: %{http_code}\n' -X POST http://convertx:3000/upload \
  -F 'file=@/w/drill-canary.png;type=image/png'
drill_api -o /dev/null -w 'convert: %{http_code}\n' -X POST http://convertx:3000/convert \
  -F 'file_names=drill-canary.png' -F 'convert_to=jpg,imagemagick' || true
drill_compose exec -T convertx sh -c 'ls -la /app/data; echo "--- files:"; find /app/data/uploads /app/data/output -type f 2>/dev/null'

echo "== backup: the data directory =="
mkdir -p "$BACKUP/data" "$BACKUP/db-only"
docker run --rm -v "${PROJECT}_convertx-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data"
cp "$BACKUP/data/mydb.sqlite" "$BACKUP/db-only/"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-all"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/app/data"
echo "== restore, reading A: the whole data directory =="
drill_check "$REPO_ROOT/recipes/convertx" "$WORK/restic-all" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-db-only"; mkdir -p "$REPO"
drill_restic "$BACKUP/db-only:/app/data"
echo "== restore, reading B: mydb.sqlite on its own =="
drill_check "$REPO_ROOT/recipes/convertx" "$WORK/restic-db-only" \
  "$APP_DIR/result-db-only.json" "$APP_DIR/result-db-only.txt"
