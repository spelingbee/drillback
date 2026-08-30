#!/usr/bin/env bash
# The Gogs leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/gogs/run.sh
#
# Deploys Gogs with the official image, installs it, creates a repository with a commit
# and uploads an avatar, then takes BOTH documented backups and restores each:
#
#   reading A - `gogs backup`, the command the CLI reference documents and the one the
#               official image's own cron job runs;
#   reading B - a copy of the /data volume, which is the only thing the image's own
#               documentation tells you to keep.
source "$(dirname "$0")/../lib.sh"

drill_init gogs
drill_up "$APP_DIR/compose.yaml"
drill_wait_http http://gogs:3000/ 120 '200|302'

echo "-- seed: install, one repository with a commit, one avatar"
drill_api http://gogs:3000/install -o /w/install.html -w 'get install: %{http_code}\n'
CSRF=$(sed -n 's/.*_csrf" value="\([^"]*\)".*/\1/p' "$WORK/curl/install.html" | head -1)
drill_api -o /dev/null -w 'post install: %{http_code}\n' -X POST http://gogs:3000/install \
  -d "_csrf=$CSRF" \
  -d 'db_type=SQLite3' -d 'db_host=127.0.0.1:3306' -d 'db_user=gogs' -d 'db_passwd=' \
  -d 'db_name=gogs' -d 'ssl_mode=disable' -d 'db_path=data/gogs.db' -d 'app_name=Gogs' \
  -d 'repo_root_path=/data/git/gogs-repositories' -d 'run_user=git' -d 'domain=gogs' \
  -d 'ssh_port=22' -d 'http_port=3000' -d 'app_url=http://gogs:3000/' \
  -d 'log_root_path=/app/gogs/log' -d 'smtp_host=' -d 'admin_name=drilladmin' \
  -d 'admin_passwd=Drill-Password-1' -d 'admin_confirm_passwd=Drill-Password-1' \
  -d 'admin_email=drill@example.invalid'

drill_api -o /dev/null -w 'login: %{http_code}\n' -X POST http://gogs:3000/user/login \
  -d 'user_name=drilladmin' -d 'password=Drill-Password-1'

drill_api http://gogs:3000/repo/create -o /w/create.html -w 'get repo form: %{http_code}\n'
CSRF=$(sed -n 's/.*_csrf" value="\([^"]*\)".*/\1/p' "$WORK/curl/create.html" | head -1)
drill_api -o /dev/null -w 'create repo: %{http_code}\n' -X POST http://gogs:3000/repo/create \
  -d "_csrf=$CSRF" -d 'user_id=1' -d 'repo_name=drill-canary-repo' \
  -d 'description=created by the restore drill' \
  -d 'gitignores=' -d 'license=' -d 'readme=Default' -d 'auto_init=on'

python -c "
import base64, sys
open(sys.argv[1], 'wb').write(base64.b64decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='))
print('avatar png written')" "$(docker_path "$WORK/curl/avatar.png")"
drill_api http://gogs:3000/user/settings/avatar -o /w/av.html -w 'get avatar form: %{http_code}\n'
CSRF=$(sed -n 's/.*_csrf" value="\([^"]*\)".*/\1/p' "$WORK/curl/av.html" | head -1)
drill_api -o /dev/null -w 'post avatar: %{http_code}\n' -X POST http://gogs:3000/user/settings/avatar \
  -F "_csrf=$CSRF" -F 'enable_custom_avatar=on' -F 'avatar=@/w/avatar.png;type=image/png'
drill_curl -f -o /dev/null -w 'repo page: %{http_code}\n' http://gogs:3000/drilladmin/drill-canary-repo

# ------------------------------------------------------- reading A: the documented command
echo "== backup, reading A: gogs backup =="
# /backup is where the image's own scheduled backup writes, and it has to belong to the
# git user before the command can write into it - which the image's backup-init.sh also
# does for itself.
drill_compose exec -T gogs sh -c 'mkdir -p /backup && chown git:git /backup'
drill_compose exec -T -u git gogs sh -c 'cd /app/gogs && ./gogs backup --target=/backup' 2>&1 | tail -3
mkdir -p "$BACKUP/archive"
drill_compose cp gogs:/backup/. "$(docker_path "$BACKUP/archive")" >/dev/null
ls -la "$BACKUP/archive"

echo "-- what is inside the archive"
drill_compose exec -T gogs sh -c 'rm -rf /tmp/x; mkdir -p /tmp/x; cd /tmp/x && unzip -o -q /backup/*.zip && find gogs-backup -maxdepth 2 | sort'

# ------------------------------------------------------- reading B: the documented volume
echo "== backup, reading B: a copy of /data =="
mkdir -p "$BACKUP/data"
drill_compose cp gogs:/data/. "$(docker_path "$BACKUP/data")" >/dev/null
du -sh "$BACKUP/data" 2>/dev/null || true

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-archive"; mkdir -p "$REPO"
drill_restic "$BACKUP/archive:/backup"
REPO="$WORK/restic-data"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/data"

echo "== restore, reading A: gogs restore --from <archive> =="
drill_check "$APP_DIR/recipe" "$WORK/restic-archive" \
  "$APP_DIR/result-archive.json" "$APP_DIR/result-archive.txt"

echo "== restore, reading B: the /data volume =="
drill_check "$REPO_ROOT/recipes/gogs" "$WORK/restic-data" \
  "$APP_DIR/result-data.json" "$APP_DIR/result-data.txt"
