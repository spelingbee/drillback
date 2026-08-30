#!/usr/bin/env bash
# The n8n leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/n8n/run.sh
#
# Deploys n8n the way its installation page says to, seeds an owner account, a workflow
# and a credential through n8n's own API, then takes TWO backups and restores each:
#
#   reading A - `n8n export:workflow --backup` + `n8n export:credentials --backup`,
#               which is the only thing n8n's documentation calls a backup;
#   reading B - the /home/node/.n8n directory the installation page says to keep on a
#               persistent volume.
#
# Neither backup is improved on here. See result.md for what each one gives back.
source "$(dirname "$0")/../lib.sh"

drill_init n8n
drill_up "$APP_DIR/compose.yaml"

# NOT /healthz, and not /rest/settings: n8n answers both from a placeholder server it
# runs while starting, which also answers POST /rest/owner/setup with 200 and creates
# nothing. /healthz/readiness is 503 until the real server is up. See result.md.
drill_wait_http http://n8n:5678/healthz/readiness 180 '200'

echo "-- seed: owner account (from inside the container, so it can be retried)"
drill_compose exec -T n8n node -e "
const payload = JSON.stringify({email:'drill@example.invalid',firstName:'Drill',lastName:'Operator',password:'Drill-Password-1'});
(async () => {
  for (let i = 0; i < 30; i++) {
    try {
      const r = await fetch('http://127.0.0.1:5678/rest/owner/setup', {
        method: 'POST', headers: {'content-type':'application/json'}, body: payload });
      const answer = await r.text();
      if (r.status === 200 && answer.includes('drill@example.invalid')) { console.log('owner account created'); return; }
      console.log('owner/setup answered', r.status, '- retrying');
    } catch (e) { console.log('owner/setup:', e.message, '- retrying'); }
    await new Promise((s) => setTimeout(s, 2000));
  }
  throw new Error('owner/setup never created the account');
})();"

echo "-- seed: log in, then create a workflow and a credential through the API"
drill_api -X POST http://n8n:5678/rest/login -H 'content-type: application/json' \
  -d '{"emailOrLdapLoginId":"drill@example.invalid","password":"Drill-Password-1"}' \
  -o /dev/null -w 'login: %{http_code}\n'
drill_api -X POST http://n8n:5678/rest/workflows -H 'content-type: application/json' \
  -d '{"name":"drill-canary-workflow","nodes":[{"parameters":{},"id":"a1b2c3d4-0000-4000-8000-000000000001","name":"When clicking Test workflow","type":"n8n-nodes-base.manualTrigger","typeVersion":1,"position":[0,0]}],"connections":{},"settings":{"executionOrder":"v1"}}' \
  -o /dev/null -w 'create-workflow: %{http_code}\n'
drill_api -X POST http://n8n:5678/rest/credentials -H 'content-type: application/json' \
  -d '{"name":"drill-canary-credential","type":"httpHeaderAuth","data":{"name":"X-Drill","value":"drill-secret-value"}}' \
  -o /dev/null -w 'create-credential: %{http_code}\n'

# ---------------------------------------------------------------- reading A: the export
echo "== backup, reading A: the documented export =="
drill_compose exec -T n8n n8n export:workflow --backup --output=/home/node/backups/latest/ 2>&1 | tail -1
drill_compose exec -T n8n n8n export:credentials --backup --output=/home/node/backups/latest/ 2>&1 | tail -1
mkdir -p "$BACKUP/export"
drill_compose cp n8n:/home/node/backups/latest/. "$(docker_path "$BACKUP/export")" >/dev/null
ls -la "$BACKUP/export"

# ---------------------------------------------------------------- reading B: the volume
echo "== backup, reading B: the .n8n directory =="
mkdir -p "$BACKUP/dotn8n"
drill_compose cp n8n:/home/node/.n8n/. "$(docker_path "$BACKUP/dotn8n")" >/dev/null
ls -la "$BACKUP/dotn8n"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-export"; mkdir -p "$REPO"
drill_restic "$BACKUP/export:/home/node/backups/latest"
REPO="$WORK/restic-dotn8n"; mkdir -p "$REPO"
drill_restic "$BACKUP/dotn8n:/home/node/.n8n"

export RESTIC_PASSWORD=$DRILL_REPO_PASSWORD
unset RESTIC_PASSWORD_FILE RESTIC_PASSWORD_COMMAND RESTIC_REPOSITORY_FILE

echo "== restore, reading A: what the documented backup gives back =="
"$REPO_ROOT/bin/restored.exe" check --recipe "$(docker_path "$APP_DIR/recipe")" \
  --source restic --from "$(docker_path "$WORK/restic-export")" \
  --report "$(docker_path "$APP_DIR/result-export.json")" 2>&1 | tee "$APP_DIR/result-export.txt"
echo "-- reading A exit: ${PIPESTATUS[0]}"

echo "== restore, reading B: what the data directory gives back =="
"$REPO_ROOT/bin/restored.exe" check --recipe "$(docker_path "$REPO_ROOT/recipes/n8n")" \
  --source restic --from "$(docker_path "$WORK/restic-dotn8n")" \
  --report "$(docker_path "$APP_DIR/result-dotn8n.json")" 2>&1 | tee "$APP_DIR/result-dotn8n.txt"
echo "-- reading B exit: ${PIPESTATUS[0]}"
