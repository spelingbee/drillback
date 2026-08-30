#!/usr/bin/env bash
# The Beszel leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/beszel/run.sh
#
# Beszel's documentation has no backup page. What it has is a hub installation page that
# mounts one directory, so the drill tests the two readings that produces:
#
#   reading A - the /beszel_data directory;
#   reading B - "the database", meaning data.db.
source "$(dirname "$0")/../lib.sh"

drill_init beszel
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://beszel:8090/ 120 '200'

echo "-- seed: the first account and one monitored system"
# The hub creates this account itself, from the documented USER_EMAIL / USER_PASSWORD.
drill_curl -f -X POST http://beszel:8090/api/collections/users/auth-with-password \
  -H 'content-type: application/json' \
  -d '{"identity":"drill@example.invalid","password":"Drill-Password-1"}' > "$WORK/auth.json"
TOKEN=$(sed -e 's/.*"token":"//' -e 's/".*//' "$WORK/auth.json")
USER_ID=$(sed -e 's/.*"record":{//' -e 's/}.*//' "$WORK/auth.json" | tr ',' '\n' |
  sed -n 's/^"id":"\([^"]*\)".*/\1/p' | head -1)
echo "signed in as $USER_ID, token of ${#TOKEN} characters"
drill_curl -f -o /dev/null -w 'create system: %{http_code}\n' \
  -X POST http://beszel:8090/api/collections/systems/records \
  -H "Authorization: $TOKEN" -H 'content-type: application/json' \
  -d "{\"name\":\"drill-canary-system\",\"host\":\"192.0.2.10\",\"port\":\"45876\",\"status\":\"pending\",\"users\":[\"$USER_ID\"]}"

echo "== backup: the data directory =="
mkdir -p "$BACKUP/data" "$BACKUP/db-only"
docker run --rm -v "${PROJECT}_beszel-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data"
cp "$BACKUP/data/data.db" "$BACKUP/db-only/"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-all"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/beszel_data"
echo "== restore, reading A: the whole data directory =="
drill_check "$REPO_ROOT/recipes/beszel" "$WORK/restic-all" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-db-only"; mkdir -p "$REPO"
drill_restic "$BACKUP/db-only:/beszel_data"
echo "== restore, reading B: data.db on its own =="
drill_check "$REPO_ROOT/recipes/beszel" "$WORK/restic-db-only" \
  "$APP_DIR/result-db-only.json" "$APP_DIR/result-db-only.txt"
