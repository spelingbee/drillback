#!/usr/bin/env bash
# The File Browser leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/filebrowser/run.sh
#
# File Browser's documentation never uses the word backup. What it says is that there
# are three volumes - /srv, /database and /config - "where the data, database and
# configuration will be stored". So the drill tests the two readings a person actually
# has:
#
#   reading A - all three volumes;
#   reading B - the files under /srv (and /config), without the database.
source "$(dirname "$0")/../lib.sh"

drill_init filebrowser
drill_up "$APP_DIR/compose.yaml"
sleep 4

echo "-- first-boot log, which is the only place the admin password is ever printed:"
drill_compose logs filebrowser 2>&1 | grep -E "initialized with randomly generated password|Listening|wound down|archived" || true

echo "-- seed: sign in as the bootstrapped admin, upload a file, share it"
PASSWORD=$(drill_compose logs filebrowser 2>&1 |
  sed -n 's/.*randomly generated password: //p' | tr -d '\r' | head -1)
TOKEN=$(drill_curl -f -X POST http://filebrowser:80/api/login \
  -H 'content-type: application/json' \
  -d "{\"username\":\"admin\",\"password\":\"$PASSWORD\",\"recaptcha\":\"\"}")
echo "login: token of ${#TOKEN} characters"
drill_curl -f -o /dev/null -w 'upload: %{http_code}\n' \
  -X POST "http://filebrowser:80/api/resources/drill-canary.txt?override=true" \
  -H "X-Auth: $TOKEN" \
  --data-binary 'drill canary file: this line proves the restore'
drill_curl -f -X POST "http://filebrowser:80/api/share/drill-canary.txt" \
  -H "X-Auth: $TOKEN" -H 'content-type: application/json' \
  -d '{"password":"","expires":"","unit":"hours"}' -w '\nshare: %{http_code}\n'

echo "== backup: the three documented volumes =="
for v in data database config; do
  mkdir -p "$BACKUP/$v"
done
docker run --rm -v "${PROJECT}_filebrowser_data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
docker run --rm -v "${PROJECT}_filebrowser_database:/src:ro" \
  -v "$(docker_path "$BACKUP/database"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
docker run --rm -v "${PROJECT}_filebrowser_config:/src:ro" \
  -v "$(docker_path "$BACKUP/config"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data" "$BACKUP/database" "$BACKUP/config"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-all"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/srv" "$BACKUP/database:/database" "$BACKUP/config:/config"
echo "== restore, reading A: all three volumes =="
drill_check "$REPO_ROOT/recipes/filebrowser" "$WORK/restic-all" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

# ----------------------------------------------------------------------- reading B
# The files and the configuration, without the database. This is the shape of backup a
# person ends up with when they back up "their files" - which, for a file manager, is a
# very easy thing to believe you have done.
echo "== reading B: the files and the configuration, without the database =="
mkdir -p "$BACKUP/empty-database"
REPO="$WORK/restic-no-db"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/srv" "$BACKUP/empty-database:/database" "$BACKUP/config:/config"
drill_check "$REPO_ROOT/recipes/filebrowser" "$WORK/restic-no-db" \
  "$APP_DIR/result-no-db.json" "$APP_DIR/result-no-db.txt"
