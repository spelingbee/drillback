#!/usr/bin/env bash
# The Gotify leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/gotify/run.sh
#
# Gotify's documentation has no backup page. What it has is a configuration reference
# naming `data/gotify.db` and an installation page mounting /app/data, so the drill tests
# the two readings that produces:
#
#   reading A - the data directory;
#   reading B - "the database", meaning gotify.db.
source "$(dirname "$0")/../lib.sh"

drill_init gotify
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://gotify:80/health 120 '200'

echo "-- seed: an application with an uploaded icon, and one message"
mkdir -p "$WORK/curl"
cp "$REPO_ROOT/recipes/gotify/test/drill-canary.png" "$WORK/curl/drill-canary.png"
drill_api -u 'drilladmin:Drill-Password-1' -o /w/app.json -w 'create application: %{http_code}\n' \
  -X POST http://gotify:80/application -H 'content-type: application/json' \
  -d '{"name":"drill-canary-app","description":"created by the restore drill"}'
TOKEN=$(sed -e 's/.*"token":"//' -e 's/".*//' "$WORK/curl/app.json")
APP_ID=$(sed -e 's/.*"id"://' -e 's/,.*//' "$WORK/curl/app.json")
echo "application $APP_ID, token ${TOKEN:0:8}..."
drill_api -u 'drilladmin:Drill-Password-1' -o /dev/null -w 'upload icon: %{http_code}\n' \
  -X POST "http://gotify:80/application/$APP_ID/image" \
  -F 'file=@/w/drill-canary.png;type=image/png'
drill_curl -o /dev/null -w 'send message: %{http_code}\n' -X POST "http://gotify:80/message?token=$TOKEN" \
  -H 'content-type: application/json' \
  -d '{"title":"drill-canary-message","message":"this line proves the restore","priority":5}'
drill_compose exec -T gotify sh -c 'ls -la /app/data; echo "--- images:"; ls -la /app/data/images'

echo "== backup: the data directory =="
mkdir -p "$BACKUP/data" "$BACKUP/db-only"
docker run --rm -v "${PROJECT}_gotify-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data"
cp "$BACKUP/data/gotify.db" "$BACKUP/db-only/"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-all"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/app/data"
echo "== restore, reading A: the whole data directory =="
drill_check "$REPO_ROOT/recipes/gotify" "$WORK/restic-all" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-db-only"; mkdir -p "$REPO"
drill_restic "$BACKUP/db-only:/app/data"
echo "== restore, reading B: gotify.db on its own =="
drill_check "$REPO_ROOT/recipes/gotify" "$WORK/restic-db-only" \
  "$APP_DIR/result-db-only.json" "$APP_DIR/result-db-only.txt"
