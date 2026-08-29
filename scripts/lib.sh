# Shared helpers for the demo scripts. Sourced, never executed.
#
# The demos stand in for a user's production system: they build a small stack with
# docker compose, put real data in it, back that data up with restic, tear the stack
# down, and then hand the backup to restored. Nothing here is part of the tool.

set -eu

# Git Bash rewrites arguments that look like absolute paths before docker sees them.
case "$(uname -s)" in
MINGW* | MSYS*)
  export MSYS_NO_PATHCONV=1
  export MSYS2_ARG_CONV_EXCL='*'
  ;;
esac

# hostpath renders a path the way the docker daemon expects to receive it.
hostpath() {
  case "$(uname -s)" in
  MINGW* | MSYS*) cygpath -m "$1" ;;
  *) printf '%s' "$1" ;;
  esac
}

say() { printf '\n== %s\n' "$*"; }

die() {
  printf 'demo: %s\n' "$*" >&2
  exit 1
}

require_docker() {
  command -v docker >/dev/null 2>&1 || die "docker is not on PATH"
  docker info >/dev/null 2>&1 || die "the docker daemon is not reachable"
  docker compose version >/dev/null 2>&1 || die "docker compose is not available"
}

require_restic() {
  command -v restic >/dev/null 2>&1 ||
    die "restic is not on PATH; restored needs it to read the repository"
}

# The images the demos use, pinned so a demo that worked yesterday works today.
IMAGE_RESTIC=restic/restic:0.19.1
IMAGE_SQLITE=keinos/sqlite3:3.46.0
IMAGE_CURL=curlimages/curl:8.16.0

# The password for the throwaway repository the demo creates and then deletes. It is
# not a secret: the repository exists for about a minute inside a temporary directory.
DEMO_RESTIC_PASSWORD=restored-demo

# demo_start makes a scratch directory and arranges for everything to be removed
# again, whatever happens. $DEMO is the scratch root, $SRV is the tree that stands in
# for the user's filesystem, and $REPO is the restic repository.
demo_start() {
  DEMO_NAME=$1
  BIN=${RESTORED_BIN:-./bin/restored}
  [ -x "$BIN" ] || BIN="$BIN.exe"
  [ -x "$BIN" ] || die "no restored binary at ${RESTORED_BIN:-./bin/restored}; run make build"

  # The scratch root is kept in the host's own path form from the start. Git Bash
  # understands C:/... perfectly well, and it means every path handed to docker or to
  # restored is already the one they expect.
  #
  # RESTORED_DEMO_DIR pins it, so captured output does not churn on a random name
  # every time scripts/capture-demo.sh runs.
  if [ -n "${RESTORED_DEMO_DIR:-}" ]; then
    rm -rf "$RESTORED_DEMO_DIR"
    mkdir -p "$RESTORED_DEMO_DIR"
    DEMO=$(hostpath "$RESTORED_DEMO_DIR")
  else
    DEMO=$(hostpath "$(mktemp -d 2>/dev/null || mktemp -d -t restored-demo)")
  fi
  SRV="$DEMO/srv"
  REPO="$DEMO/repo"
  PROJECT="restored-demo-$DEMO_NAME-$$"
  COMPOSE="$DEMO/compose.yaml"
  mkdir -p "$SRV" "$REPO"
  trap demo_cleanup EXIT INT TERM
  say "demo workspace $DEMO"
}

demo_cleanup() {
  status=$?
  if [ -n "${COMPOSE:-}" ] && [ -f "$COMPOSE" ]; then
    sample down -v --remove-orphans --timeout 20 >/dev/null 2>&1 || true
  fi
  if [ -n "${DEMO:-}" ] && [ -d "$DEMO" ]; then
    rm -rf "$DEMO" 2>/dev/null || true
  fi
  return $status
}

sample() { docker compose -p "$PROJECT" -f "$(hostpath "$COMPOSE")" "$@"; }

sample_network() {
  docker network ls --filter "label=com.docker.compose.project=$PROJECT" --format '{{.Name}}' | head -1
}

# wait_http polls a URL from a helper container on the sample stack's own network,
# because the sample stack publishes no ports either.
wait_http() {
  url=$1
  tries=${2:-90}
  net=$(sample_network)
  i=0
  while [ "$i" -lt "$tries" ]; do
    code=$(docker run --rm --network "$net" --entrypoint "" "$IMAGE_CURL" \
      curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$url" 2>/dev/null || echo 000)
    case "$code" in
    2* | 3*) return 0 ;;
    esac
    i=$((i + 1))
    sleep 2
  done
  die "timed out waiting for $url (last status ${code:-none})"
}

# restic_in_container runs restic against $REPO with $SRV mounted at /srv, so the
# snapshot carries clean POSIX paths that match the recipe defaults on every host.
restic_in_container() {
  docker run --rm \
    -v "$(hostpath "$REPO"):/repo" \
    -v "$(hostpath "$SRV"):/srv:ro" \
    -e RESTIC_REPOSITORY=/repo \
    -e RESTIC_PASSWORD="$DEMO_RESTIC_PASSWORD" \
    "$IMAGE_RESTIC" "$@"
}

restic_init_and_backup() {
  say "backing $* up into a throwaway restic repository"
  restic_in_container init >/dev/null
  restic_in_container backup --host demo-host --tag "$DEMO_NAME" "$@" >/dev/null
}

run_check() {
  say "restored check"
  RESTIC_PASSWORD="$DEMO_RESTIC_PASSWORD" "$BIN" check --from "$REPO" "$@"
}

# ---- the Gitea sample stack ------------------------------------------------
#
# Shared by demo.sh and demo-broken.sh, which differ only in how the database is
# dumped. It is deliberately not the recipe's compose.yaml: the recipe describes how
# to bring a backup up, and this describes the system the backup was taken from.

GITEA_USER=drilluser
GITEA_PASSWORD=drill-pass-123
GITEA_DB=gitea

helper_curl() {
  docker run --rm --network "$(sample_network)" --entrypoint "" "$IMAGE_CURL" curl "$@"
}

gitea_sample_stack() {
  DATA="$SRV/gitea/data"
  mkdir -p "$DATA"

  cat >"$COMPOSE" <<EOF
services:
  db:
    image: postgres:16.4-alpine
    environment:
      POSTGRES_DB: $GITEA_DB
      POSTGRES_USER: $GITEA_DB
      POSTGRES_PASSWORD: demo-throwaway
    volumes:
      # Where the dump is written, so the bytes never pass through the host shell.
      - "$(hostpath "$SRV/gitea"):/dump"
    networks: [demo]

  gitea:
    image: gitea/gitea:1.22.6
    depends_on:
      db:
        condition: service_started
    environment:
      USER_UID: "1000"
      USER_GID: "1000"
      GITEA__database__DB_TYPE: postgres
      GITEA__database__HOST: db:5432
      GITEA__database__NAME: $GITEA_DB
      GITEA__database__USER: $GITEA_DB
      GITEA__database__PASSWD: demo-throwaway
      GITEA__server__ROOT_URL: http://gitea:3000/
      GITEA__server__OFFLINE_MODE: "true"
      GITEA__security__INSTALL_LOCK: "true"
      GITEA__cron__ENABLED: "false"
    volumes:
      - "$(hostpath "$DATA"):/data"
    networks: [demo]

networks:
  demo:
    internal: true
EOF

  say "starting a sample Gitea and PostgreSQL"
  sample up -d --quiet-pull
  wait_http http://gitea:3000/api/healthz 120
}

gitea_seed() {
  sample exec -T -u git gitea gitea admin user create \
    --username "$GITEA_USER" \
    --password "$GITEA_PASSWORD" \
    --email drill@example.invalid \
    --admin --must-change-password=false >/dev/null

  code=$(helper_curl -sS -o /dev/null -w '%{http_code}' --max-time 60 \
    -u "$GITEA_USER:$GITEA_PASSWORD" \
    -H 'Content-Type: application/json' \
    -X POST -d '{"name":"drill-repo","auto_init":true,"private":false}' \
    http://gitea:3000/api/v1/user/repos)
  [ "$code" = "201" ] || die "creating the sample repository returned HTTP $code"
}

# gitea_dump writes the dump the backup will carry. The database it is told to dump
# is the whole difference between demo.sh and demo-broken.sh.
gitea_dump() {
  database=$1
  sample stop gitea >/dev/null
  sample exec -T db pg_dump -U "$GITEA_DB" -d "$database" -f /dump/db.sql
  sample exec -T db sh -c 'ls -l /dump/db.sql'
}
