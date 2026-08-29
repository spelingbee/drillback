#!/usr/bin/env sh
#
# The Uptime Kuma round trip: stand up a real Uptime Kuma, put a user, a monitor and
# a heartbeat in it, back the data directory up with restic, destroy the stack, and
# ask restored whether the backup would come back.
#
# Needs docker and restic. Idempotent, and it cleans up after itself.

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
. ./scripts/lib.sh

require_docker
require_restic
demo_start uptime-kuma

DATA="$SRV/uptime-kuma/data"
SEED="$DEMO/seed"
mkdir -p "$DATA" "$SEED"
cp recipes/uptime-kuma/test/seed.sql "$SEED/seed.sql"

cat >"$COMPOSE" <<EOF
# The user's production stack, as far as this demo is concerned. It is deliberately
# not the recipe's compose.yaml: the recipe describes how to bring a backup up, and
# this describes the system the backup was taken from.
services:
  kuma:
    image: louislam/uptime-kuma:1.23.16-alpine
    volumes:
      - "$(hostpath "$DATA"):/app/data"
    networks: [demo]

  seeder:
    image: $IMAGE_SQLITE
    # The image runs as its own unprivileged user, which cannot write to a bind mount
    # whose files belong to the application's uid. The seeder is scaffolding, not the
    # thing under test, so it runs as root.
    user: "0:0"
    volumes:
      - "$(hostpath "$DATA"):/app/data"
      - "$(hostpath "$SEED"):/seed:ro"
    command: ["sleep", "infinity"]
    networks: [demo]

networks:
  demo:
    internal: true
EOF

say "starting a sample Uptime Kuma"
sample up -d --quiet-pull
wait_http http://kuma:3001/

say "stopping Uptime Kuma, so the backup is taken from a quiet database"
sample stop kuma >/dev/null

say "seeding a user, a monitor and a heartbeat"
sample exec -T seeder sh -c 'sqlite3 /app/data/kuma.db < /seed/seed.sql'

restic_init_and_backup /srv/uptime-kuma/data

say "destroying the sample stack, so nothing but the backup is left"
sample down -v --remove-orphans --timeout 20 >/dev/null

run_check --recipe ./recipes/uptime-kuma "$@"
