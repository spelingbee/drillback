#!/usr/bin/env bash
# The SiYuan leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/siyuan/run.sh
#
# SiYuan's README has no backup section. What it has is a Docker section that mounts one
# thing - the workspace - so the drill tests the two readings that produces:
#
#   reading A - the whole workspace;
#   reading B - workspace/data, which is where the notes are.
source "$(dirname "$0")/../lib.sh"

drill_init siyuan
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://siyuan:6806/ 180 '401'

echo "-- seed: a notebook and a document, through SiYuan's API"
TOKEN=$(drill_compose exec -T siyuan sh -c \
  'grep -o "\"token\": *\"[^\"]*\"" /siyuan/workspace/conf/conf.json | head -1 | sed "s/.*: *\"//; s/\"//"' | tr -d '\r')
echo "api token: ${#TOKEN} characters"
NOTEBOOK=$(drill_curl -f -X POST http://siyuan:6806/api/notebook/createNotebook \
  -H "Authorization: Token $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"drill-canary-notebook"}' | sed -e 's/.*"id": *"//' -e 's/".*//')
echo "notebook: $NOTEBOOK"
drill_curl -f -o /dev/null -w 'create document: %{http_code}\n' \
  -X POST http://siyuan:6806/api/filetree/createDocWithMd \
  -H "Authorization: Token $TOKEN" -H 'content-type: application/json' \
  -d "{\"notebook\":\"$NOTEBOOK\",\"path\":\"/drill-canary-doc\",\"markdown\":\"drill canary note: this line proves the restore\"}"
drill_compose exec -T siyuan sh -c 'ls -la /siyuan/workspace; grep -rl "this line proves the restore" /siyuan/workspace/data'

echo "== backup: the workspace directory =="
mkdir -p "$BACKUP/workspace" "$BACKUP/data-only"
docker run --rm -v "${PROJECT}_siyuan-workspace:/src:ro" \
  -v "$(docker_path "$BACKUP/workspace"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
du -sh "$BACKUP/workspace" "$BACKUP"/workspace/* 2>/dev/null || true

# Reading B: the notes, without the configuration beside them.
mkdir -p "$BACKUP/data-only/data"
cp -a "$BACKUP/workspace/data/." "$BACKUP/data-only/data/"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-workspace"; mkdir -p "$REPO"
drill_restic "$BACKUP/workspace:/siyuan/workspace"
echo "== restore, reading A: the whole workspace =="
drill_check "$REPO_ROOT/recipes/siyuan" "$WORK/restic-workspace" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-data-only"; mkdir -p "$REPO"
drill_restic "$BACKUP/data-only:/siyuan/workspace"
echo "== restore, reading B: workspace/data, without conf =="
drill_check "$REPO_ROOT/recipes/siyuan" "$WORK/restic-data-only" \
  "$APP_DIR/result-data-only.json" "$APP_DIR/result-data-only.txt"
