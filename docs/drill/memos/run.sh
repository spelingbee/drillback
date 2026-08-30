#!/usr/bin/env bash
# The Memos leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/memos/run.sh
#
# Deploys Memos with its own documented compose file, creates the host account and a
# memo through its own API, then takes the backup its documentation describes - the data
# directory, "the database and any local assets" - and restores it.
#
# The copy is taken with the instance running, because nothing in the Memos
# documentation says to stop it first. That is the point of the exercise.
source "$(dirname "$0")/../lib.sh"

drill_init memos
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://memos:5230/healthz 120 '200'

echo "-- seed: the host account and one memo, through the Memos API"
drill_curl -f -X POST http://memos:5230/api/v1/users \
  -H 'content-type: application/json' \
  -d '{"username":"drill","password":"Drill-Password-1","role":"HOST"}' \
  -o /dev/null -w 'create-user: %{http_code}\n'
TOKEN=$(drill_curl -f -X POST http://memos:5230/api/v1/auth/signin \
  -H 'content-type: application/json' \
  -d '{"passwordCredentials":{"username":"drill","password":"Drill-Password-1"}}' \
  | sed -e 's/.*"accessToken": *"//' -e 's/".*//')
echo "signin: token of ${#TOKEN} characters"
drill_curl -f -X POST http://memos:5230/api/v1/memos \
  -H "authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"content":"drill-canary-memo: this line proves the restore","visibility":"PRIVATE"}' \
  -o /dev/null -w 'create-memo: %{http_code}\n'

echo "== backup: the data directory, exactly as documented =="
mkdir -p "$BACKUP/data"
docker run --rm -v "${PROJECT}_memos-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data"

echo "== teardown of the seeded instance =="
drill_down

drill_restic "$BACKUP/data:/var/opt/memos"

echo "== restore =="
export RESTIC_PASSWORD=$DRILL_REPO_PASSWORD
unset RESTIC_PASSWORD_FILE RESTIC_PASSWORD_COMMAND RESTIC_REPOSITORY_FILE
"$REPO_ROOT/bin/restored.exe" check --recipe "$(docker_path "$REPO_ROOT/recipes/memos")" \
  --source restic --from "$(docker_path "$REPO")" \
  --report "$(docker_path "$APP_DIR/result.json")" 2>&1 | tee "$APP_DIR/result.txt"
echo "-- exit: ${PIPESTATUS[0]}"

# --------------------------------------------------------------------- reading B
# "back up both the database and any local assets" can be read as "back up the
# database file". On this instance memos_prod.db was 4 KiB and memos_prod.db-wal was
# 160 KiB, so that reading is worth its own verdict. Same recipe, same snapshot path -
# the only difference is which files are in it.
echo "== reading B: the database file alone, without its -wal companion =="
mkdir -p "$BACKUP/db-only"
cp "$BACKUP/data/memos_prod.db" "$BACKUP/db-only/"
ls -la "$BACKUP/db-only"
REPO="$WORK/restic-db-only"; mkdir -p "$REPO"
drill_restic "$BACKUP/db-only:/var/opt/memos"
"$REPO_ROOT/bin/restored.exe" check --recipe "$(docker_path "$REPO_ROOT/recipes/memos")" \
  --source restic --from "$(docker_path "$REPO")" \
  --report "$(docker_path "$APP_DIR/result-db-only.json")" 2>&1 | tee "$APP_DIR/result-db-only.txt"
echo "-- reading B exit: ${PIPESTATUS[0]}"
