#!/usr/bin/env bash
# The FreshRSS leg of the restore drill. Run from the repository root:
#
#   bash docs/drill/freshrss/run.sh
#
# FreshRSS has a backup page that says exactly what to keep and what to skip, so the
# drill tests both halves of that sentence:
#
#   reading A - ./data and ./extensions, which the page calls required and recommended;
#   reading B - the same, with cache/ left out, which the page says FreshRSS rebuilds.
source "$(dirname "$0")/../lib.sh"

drill_init freshrss
drill_up "$APP_DIR/compose.yaml"
for i in $(seq 1 60); do
  code=$(drill_curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://freshrss/i/ 2>/dev/null || true)
  [ "$code" = "200" ] && { echo "-- ready after ${i} attempts"; break; }
  sleep 3
done

echo "-- seed: subscribe to the feed next door and fetch it"
drill_compose exec -T freshrss sh -c 'cd /var/www/FreshRSS && cat > /tmp/drill.opml <<XML
<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0"><head><title>drill</title></head><body>
<outline text="drill-canary-feed" title="drill-canary-feed" type="rss" xmlUrl="http://feed/feed.xml" htmlUrl="http://feed/"/>
</body></opml>
XML
./cli/import-for-user.php --user drilladmin --filename /tmp/drill.opml 2>&1 | tail -2
./cli/actualize-user.php --user drilladmin 2>&1 | tail -2
ls -la /var/www/FreshRSS/data/users/drilladmin/'

echo "== backup: ./data and ./extensions =="
mkdir -p "$BACKUP/data" "$BACKUP/extensions" "$BACKUP/data-no-cache"
docker run --rm -v "${PROJECT}_freshrss-data:/src:ro" \
  -v "$(docker_path "$BACKUP/data"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
docker run --rm -v "${PROJECT}_freshrss-extensions:/src:ro" \
  -v "$(docker_path "$BACKUP/extensions"):/dst" alpine:3.20 sh -c 'cp -a /src/. /dst/'
du -sh "$BACKUP/data" "$BACKUP/data/cache" "$BACKUP/extensions" 2>/dev/null || true

# Reading B: the page says "You can skip cache/; FreshRSS rebuilds it."
docker run --rm -v "$(docker_path "$BACKUP/data"):/src:ro" \
  -v "$(docker_path "$BACKUP/data-no-cache"):/dst" alpine:3.20 \
  sh -c 'cd /src && tar cf - --exclude=./cache . | (cd /dst && tar xf -)'
du -sh "$BACKUP/data-no-cache"

echo "== teardown of the seeded instance =="
drill_down

REPO="$WORK/restic-all"; mkdir -p "$REPO"
drill_restic "$BACKUP/data:/var/www/FreshRSS/data" "$BACKUP/extensions:/var/www/FreshRSS/extensions"
echo "== restore, reading A: data and extensions =="
drill_check "$REPO_ROOT/recipes/freshrss" "$WORK/restic-all" \
  "$APP_DIR/result.json" "$APP_DIR/result.txt"

REPO="$WORK/restic-no-cache"; mkdir -p "$REPO"
drill_restic "$BACKUP/data-no-cache:/var/www/FreshRSS/data" "$BACKUP/extensions:/var/www/FreshRSS/extensions"
echo "== restore, reading B: the same without cache/ =="
drill_check "$REPO_ROOT/recipes/freshrss" "$WORK/restic-no-cache" \
  "$APP_DIR/result-no-cache.json" "$APP_DIR/result-no-cache.txt"
