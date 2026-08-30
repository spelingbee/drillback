#!/usr/bin/env bash
# The listmonk leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/listmonk/run.sh
#
# listmonk has no backup page. What its documentation says about backups is one warning,
# twice: "Always take a backup of the Postgres database before upgrading listmonk". So
# the drill takes that backup, and takes it again with the uploads directory beside it:
#
#   reading A - the Postgres database, which is what the documentation names;
#   reading B - the same dump plus /listmonk/uploads.
source "$(dirname "$0")/../lib.sh"

drill_init listmonk
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://app:9000/health 120 '200'

echo "-- seed: log in, create a list, a subscriber and a media upload"
drill_api -o /dev/null -w 'login: %{http_code}\n' -X POST http://app:9000/admin/login \
  -H 'content-type: application/x-www-form-urlencoded' \
  -d 'username=drilladmin&password=Drill-Password-1'
drill_api -o /w/list.json -w 'create list: %{http_code}\n' -X POST http://app:9000/api/lists \
  -H 'content-type: application/json' \
  -d '{"name":"drill-canary-list","type":"private","optin":"single","tags":["drill"]}'
LIST_ID=$(sed -e 's/.*"id"://' -e 's/,.*//' "$WORK/curl/list.json")
echo "list id: $LIST_ID"
drill_api -o /dev/null -w 'create subscriber: %{http_code}\n' -X POST http://app:9000/api/subscribers \
  -H 'content-type: application/json' \
  -d "{\"email\":\"drill-canary@example.invalid\",\"name\":\"Drill Canary\",\"status\":\"enabled\",\"lists\":[$LIST_ID]}"
cp "$REPO_ROOT/recipes/listmonk/test/drill-canary.png" "$WORK/curl/drill-canary.png"
drill_api -o /dev/null -w 'upload media: %{http_code}\n' -X POST http://app:9000/api/media \
  -F 'file=@/w/drill-canary.png;type=image/png'
echo "-- what listmonk put on disk for that upload:"
drill_compose exec -T app sh -c 'ls -la /listmonk/uploads'

echo "== backup: the Postgres database, which is what the documentation names =="
mkdir -p "$BACKUP/db" "$BACKUP/uploads" "$BACKUP/empty-uploads"
drill_compose exec -T db sh -c 'pg_dump -U listmonk -d listmonk' > "$BACKUP/db/db.sql"
ls -la "$BACKUP/db"

echo "-- and the uploads directory, for the second reading"
docker run --rm -v "${PROJECT}_listmonk-uploads:/src:ro" \
  -v "$(docker_path "$BACKUP/uploads"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/uploads"

echo "== teardown of the seeded instance =="
drill_down

# Reading A: the dump, with an empty uploads directory beside it - which is what a
# snapshot of a machine backed up "the Postgres database" way actually contains.
REPO="$WORK/restic-db-only"; mkdir -p "$REPO"
drill_restic "$BACKUP/db/db.sql:/srv/listmonk/db.sql" "$BACKUP/empty-uploads:/srv/listmonk/uploads"
echo "== restore, reading A: the database dump alone =="
drill_check "$REPO_ROOT/recipes/listmonk" "$WORK/restic-db-only" \
  "$APP_DIR/result-db-only.json" "$APP_DIR/result-db-only.txt"

REPO="$WORK/restic-all"; mkdir -p "$REPO"
drill_restic "$BACKUP/db/db.sql:/srv/listmonk/db.sql" "$BACKUP/uploads:/srv/listmonk/uploads"
echo "== restore, reading B: the dump and the uploads directory =="
drill_check "$REPO_ROOT/recipes/listmonk" "$WORK/restic-all" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"
