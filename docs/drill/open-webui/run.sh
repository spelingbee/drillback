#!/usr/bin/env bash
# The Open WebUI leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/open-webui/run.sh
#
# Deploys Open WebUI the way its quick start says to, signs up an account and saves a
# chat through its own API, then takes the backup its own backup page describes - the
# whole of /app/backend/data, copied with the stack stopped - and restores it.
source "$(dirname "$0")/../lib.sh"

drill_init open-webui
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://open-webui:8080/health 180 '200'

echo "-- seed: sign up an account and save a chat, through Open WebUI's own API"
drill_compose exec -T open-webui python3 -c "
import json, urllib.request
BASE = 'http://127.0.0.1:8080'

def post(path, body, token=None):
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode(),
                                 headers={'content-type': 'application/json'})
    if token:
        req.add_header('authorization', 'Bearer ' + token)
    with urllib.request.urlopen(req) as r:
        return json.load(r)

user = post('/api/v1/auths/signup', {'name': 'Drill Operator',
    'email': 'drill@example.invalid', 'password': 'Drill-Password-1'})
chat = post('/api/v1/chats/new', {'chat': {
    'title': 'drill-canary-chat', 'models': ['drill-model'],
    'messages': [{'id': 'drill-msg-1', 'role': 'user', 'content': 'drill canary message'}]}},
    user['token'])
print('signed up', user['email'], '- saved chat', chat['id'], chat['title'])
"

# The backup page's own scripts stop the stack before they copy:
#   docker-compose -f \"\$COMPOSE_FILE\" down
# so the copy below is taken with nothing writing to the database.
echo "== backup: the documented data directory, stack stopped =="
drill_compose stop
mkdir -p "$BACKUP/data"
docker run --rm -v "${PROJECT}_open-webui:/src:ro" -v "$(docker_path "$BACKUP/data"):/dst" \
  alpine:3.20 sh -c 'cp -a /src/. /dst/'
ls -la "$BACKUP/data"
du -sh "$BACKUP/data" 2>/dev/null || true

echo "== teardown of the seeded instance =="
drill_down

drill_restic "$BACKUP/data:/opt/open-webui"

echo "== restore =="
drill_check "$REPO_ROOT/recipes/open-webui" "$REPO" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"
