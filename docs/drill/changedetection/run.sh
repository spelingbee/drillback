#!/usr/bin/env bash
# The changedetection.io leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/changedetection/run.sh
#
# changedetection.io has a Backups page that writes a zip, and a wiki page about putting
# that zip back. So:
#
#   reading A - the zip the Backups page writes, restored by unzipping it into
#               /datastore as the wiki page says;
#   reading B - a copy of /datastore, as a control.
source "$(dirname "$0")/../lib.sh"

drill_init changedetection
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://changedetection:5000/ 180 '200'

echo "-- seed: one watch on the page next door, checked once so it has history"
KEY=$(drill_compose exec -T changedetection sh -c \
  'grep -o "\"api_access_token\": *\"[^\"]*\"" /datastore/changedetection.json | head -1 | sed "s/.*: *\"//; s/\"//"' | tr -d '\r')
echo "api key: ${#KEY} characters"
UUID=$(drill_curl -f -X POST http://changedetection:5000/api/v1/watch -H "x-api-key: $KEY" \
  -H 'content-type: application/json' \
  -d '{"url":"http://page/","title":"drill-canary-watch"}' |
  sed -e 's/.*"uuid": *"//' -e 's/".*//')
echo "watch: $UUID"
drill_curl -f -o /dev/null -w 'recheck: %{http_code}\n' -H "x-api-key: $KEY" \
  "http://changedetection:5000/api/v1/watch/$UUID?recheck=1"
for i in $(seq 1 30); do
  if drill_curl -H "x-api-key: $KEY" "http://changedetection:5000/api/v1/watch/$UUID/history" |
      grep -q datastore; then
    echo "history written after ${i} attempts"; break
  fi
  sleep 2
done
drill_compose exec -T changedetection sh -c "ls -la /datastore/$UUID"

echo "== backup: the Backups page's own zip =="
# Backups -> Create backup, through the link that button uses.
drill_curl -f -o /dev/null -w 'request-backup: %{http_code}\n' -L http://changedetection:5000/backups/request-backup
for i in $(seq 1 30); do
  if drill_compose exec -T changedetection sh -c 'ls /datastore/changedetection-backup-*.zip' >/dev/null 2>&1; then
    echo "backup written after ${i} attempts"; break
  fi
  sleep 2
done
drill_compose exec -T changedetection sh -c 'ls -la /datastore/changedetection-backup-*.zip'

mkdir -p "$BACKUP/zip" "$BACKUP/datastore"
docker run --rm -v "${PROJECT}_changedetection-data:/src:ro" \
  -v "$(docker_path "$BACKUP/datastore"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
cp "$BACKUP"/datastore/changedetection-backup-*.zip "$BACKUP/zip/"
ls -la "$BACKUP/zip"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-zip"; mkdir -p "$REPO"
drill_restic "$BACKUP/zip:/srv/changedetection-backups"
echo "== restore, reading A: unzip the backup into /datastore =="
drill_check "$APP_DIR/recipe" "$WORK/restic-zip" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-datastore"; mkdir -p "$REPO"
drill_restic "$BACKUP/datastore:/datastore"
echo "== control: a copy of /datastore =="
drill_check "$REPO_ROOT/recipes/changedetection" "$WORK/restic-datastore" \
  "$APP_DIR/result-datastore.json" "$APP_DIR/result-datastore.txt"
