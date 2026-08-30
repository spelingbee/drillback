#!/usr/bin/env bash
# Shared helpers for the restore drill (docs/drill/).
#
# Every app folder under docs/drill/ has a run.sh that sources this file. The job of
# the drill is to follow an application's own backup documentation literally, put what
# that documentation produces into a restic repository, and then hand the repository to
# `restored check`. Nothing here improves on the documented backup; that is the point.
#
# Usage, from the repository root:
#
#   bash docs/drill/<app>/run.sh
#
# Environment:
#   DRILL_WORK   scratch directory for deployments, backups and restic repositories
#                (default: $TMPDIR/restored-drill)

set -euo pipefail

# Git Bash rewrites container paths into Windows ones before the daemon sees them.
export MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*'

DRILL_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$DRILL_LIB_DIR/../.." && pwd)
DRILL_WORK=${DRILL_WORK:-"${TMPDIR:-/tmp}/restored-drill"}

RESTIC_IMAGE=restic/restic:0.19.1
CURL_IMAGE=curlimages/curl:8.10.1
DRILL_REPO_PASSWORD=drill

# Windows has no compiler for a host restic, and the container is what makes a snapshot
# record /var/opt/memos rather than a path inside this scratch directory.
# Git Bash paths (/c/My/...) are not what docker.exe means by /c/My/...; cygpath -m
# renders the same location as C:/My/... , which both the daemon and compose accept.
docker_path() { cygpath -m "$1" 2>/dev/null || printf '%s' "$1"; }

drill_init() {
  APP=$1
  APP_DIR="$DRILL_LIB_DIR/$APP"
  PROJECT="drill-$APP"
  WORK="$DRILL_WORK/$APP"
  BACKUP="$WORK/backup"      # what the documented backup produces
  REPO="$WORK/restic"        # the restic repository handed to `restored check`
  # A previous run of this app leaves a compose project and its volumes behind if it
  # was interrupted. The drill starts from nothing, every time.
  docker compose -p "$PROJECT" down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$WORK"
  mkdir -p "$BACKUP" "$REPO"
  echo "== drill: $APP =="
  echo "-- work: $WORK"
}

# drill_up <compose file> [args...] - bring the application up in its own project.
drill_up() {
  local file=$1; shift
  DRILL_COMPOSE=$(docker_path "$file")
  docker compose -p "$PROJECT" -f "$DRILL_COMPOSE" up -d "$@"
}

drill_compose() { docker compose -p "$PROJECT" -f "$DRILL_COMPOSE" "$@"; }

drill_down() {
  docker compose -p "$PROJECT" -f "$DRILL_COMPOSE" down -v --remove-orphans 2>&1 | tail -3 || true
}

# drill_net - the network the application's compose project created.
drill_net() { printf '%s_default' "$PROJECT"; }

# drill_curl <curl args...> - curl from inside the application's network, so the drill
# never needs a published port on the host.
drill_curl() {
  docker run --rm --network "$(drill_net)" "$CURL_IMAGE" -sS "$@"
}

# drill_wait_http <url> [tries] [expected-status-regex] - wait for the app to answer.
drill_wait_http() {
  local url=$1 tries=${2:-90} want=${3:-'2..|3..'} code
  for i in $(seq 1 "$tries"); do
    code=$(docker run --rm --network "$(drill_net)" "$CURL_IMAGE" \
      -s -o /dev/null -w '%{http_code}' --max-time 10 "$url" 2>/dev/null || true)
    if [[ $code =~ ^($want)$ ]]; then
      echo "-- ready after ${i}s-ish: $url -> $code"
      return 0
    fi
    sleep 2
  done
  echo "!! never became ready: $url (last code: ${code:-none})" >&2
  return 1
}

# drill_restic <hostpath>:<absolute path in the snapshot> ...
#
# Creates the throwaway repository and puts the documented backup into it at the paths
# a user's own machine would have. Runs restic in a container for the same reason the
# round-trip harness does (ADR-051): the snapshot has to record /srv/gogs/data, not a
# path inside this scratch directory.
drill_restic() {
  local binds=(-v "$(docker_path "$REPO"):/repo") paths=()
  local pair host abs
  for pair in "$@"; do
    host=${pair%%:*}; abs=${pair#*:}
    binds+=(-v "$(docker_path "$host"):$abs:ro")
    paths+=("$abs")
  done
  docker run --rm -e RESTIC_PASSWORD="$DRILL_REPO_PASSWORD" \
    -v "$(docker_path "$REPO"):/repo" "$RESTIC_IMAGE" \
    --no-cache --repo /repo init --repository-version 2 >/dev/null
  docker run --rm -e RESTIC_PASSWORD="$DRILL_REPO_PASSWORD" \
    "${binds[@]}" "$RESTIC_IMAGE" \
    --no-cache --repo /repo backup --tag drill --host drill "${paths[@]}" | tail -5
}

# drill_check [extra restored flags...] - run the drill's verdict.
drill_check() {
  export RESTIC_PASSWORD="$DRILL_REPO_PASSWORD"
  unset RESTIC_PASSWORD_FILE RESTIC_PASSWORD_COMMAND RESTIC_REPOSITORY_FILE || true
  set +e
  "$REPO_ROOT/bin/restored.exe" check \
    --recipe "$APP_DIR/recipe" \
    --source restic --from "$REPO" \
    --report "$APP_DIR/result.json" \
    "$@" 2>&1 | tee "$APP_DIR/result.txt"
  DRILL_VERDICT=${PIPESTATUS[0]}
  set -e
  echo "-- restored check exit: $DRILL_VERDICT"
  return 0
}

# drill_api <curl args...> - curl from inside the application's network, keeping a
# cookie jar between calls so a session-authenticated API can be driven.
drill_api() {
  mkdir -p "$WORK/curl"
  docker run --rm --network "$(drill_net)" \
    -v "$(docker_path "$WORK/curl"):/w" -w /w "$CURL_IMAGE" \
    -sS -c /w/cookies -b /w/cookies "$@"
}

# drill_json <file> <python expression over the parsed document `d`>
drill_json() { python -c "import json,sys;d=json.load(open(sys.argv[1]));print($2)" "$1"; }

# drill_check <recipe dir> <restic repo> <report.json> <report.txt>
#
# Runs the verdict and prints the exit code without ending the script: a FAIL is a
# result to record, not a reason to stop, and there is usually a second reading to test
# after the first one has failed.
drill_check() {
  export RESTIC_PASSWORD=$DRILL_REPO_PASSWORD
  unset RESTIC_PASSWORD_FILE RESTIC_PASSWORD_COMMAND RESTIC_REPOSITORY_FILE || true
  set +e
  "$REPO_ROOT/bin/restored.exe" check \
    --recipe "$(docker_path "$1")" \
    --source restic --from "$(docker_path "$2")" \
    --report "$(docker_path "$3")" 2>&1 | tee "$4"
  local code=${PIPESTATUS[0]}
  set -e
  echo "-- restored check exit: $code"
}
