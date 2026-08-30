#!/usr/bin/env bash
# The Mealie leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/mealie/run.sh
#
# Mealie has a backup page, and it names two things. One is the integrated backup
# feature; the other is a tip that says copying /app/data with the container stopped is
# "the best way to backup your data". The drill tests the second, which is the one the
# page recommends, and records the first as untested and why.
source "$(dirname "$0")/../lib.sh"

drill_init mealie
drill_up "$APP_DIR/compose.yaml"
for i in $(seq 1 60); do
  code=$(drill_curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://mealie:9000/api/app/about 2>/dev/null || true)
  [ "$code" = "200" ] && { echo "-- ready after ${i} attempts"; break; }
  sleep 3
done

echo "-- seed: log in as the account Mealie creates for itself, and add a recipe"
TOKEN=$(drill_curl -f -X POST http://mealie:9000/api/auth/token \
  -H 'content-type: application/x-www-form-urlencoded' \
  -d 'username=changeme@example.com&password=MyPassword' |
  sed -e 's/.*"access_token":"//' -e 's/".*//')
echo "token of ${#TOKEN} characters"
drill_curl -f -o /dev/null -w 'create recipe: %{http_code}\n' -X POST http://mealie:9000/api/recipes \
  -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"drill canary recipe"}'

echo "== backup: the data directory, with the container stopped =="
# The backup page: "You can easily perform entire site backups by stopping the
# container, and backing up this folder with your chosen tool. This is the best way to
# backup your data."
drill_compose stop
mkdir -p "$BACKUP/data"
docker run --rm -v "${PROJECT}_mealie-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-data"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/app/data"
echo "== restore: the documented best way =="
drill_check "$REPO_ROOT/recipes/mealie" "$WORK/restic-data" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"
