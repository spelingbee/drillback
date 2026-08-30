#!/usr/bin/env sh
#
# The Gitea round trip, end to end, on a machine that has docker and restic and
# nothing else prepared:
#
#   1. stand up a real Gitea and PostgreSQL,
#   2. put a user, a repository and a commit in them,
#   3. dump the database and back the data directory and the dump up with restic,
#   4. destroy the stack, so nothing but the backup is left,
#   5. ask drillback whether that backup would actually come back.
#
# Expected result: PASS, exit 0. Idempotent, and it cleans up after itself.

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
. ./scripts/lib.sh

require_docker
require_restic
demo_start gitea
gitea_sample_stack

say "seeding a user, a repository and a commit"
gitea_seed

say "dumping the database the way a nightly backup script would"
gitea_dump gitea

restic_init_and_backup /srv/gitea/data /srv/gitea/db.sql

say "destroying the sample stack, so nothing but the backup is left"
sample down -v --remove-orphans --timeout 20 >/dev/null

run_check --recipe ./recipes/gitea "$@"
