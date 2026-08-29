#!/usr/bin/env sh
#
# The same Gitea round trip as demo.sh, with one thing changed: the nightly dump is
# taken from the wrong database.
#
#   pg_dump -U gitea -d postgres     instead of     pg_dump -U gitea -d gitea
#
# That is one character in a cron line. It exits 0, it produces a file, the file
# arrives in the backup every night, and it contains none of the application's tables.
# This is the failure restored exists to find.
#
# Expected result: RESTORE UNUSABLE, exit 1. Idempotent, and it cleans up after
# itself.
#
# The session brief suggested reproducing this with `pg_dump --schema=public`. That
# does not reproduce it: Gitea's tables live in the public schema, so the dump would
# contain them, and on PostgreSQL 15 and later the resulting CREATE SCHEMA public
# fails against a fresh database before any check runs. See DECISIONS.md ADR-043.

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
. ./scripts/lib.sh

require_docker
require_restic
demo_start gitea-broken
gitea_sample_stack

say "seeding a user, a repository and a commit"
gitea_seed

say "dumping the WRONG database, the way a mistyped cron line would"
gitea_dump postgres

restic_init_and_backup /srv/gitea/data /srv/gitea/db.sql

say "destroying the sample stack, so nothing but the backup is left"
sample down -v --remove-orphans --timeout 20 >/dev/null

set +e
run_check --recipe ./recipes/gitea "$@"
status=$?
set -e

say "restored exited $status"
# The script's exit code is the tool's, because that is the thing being demonstrated.
# Anything other than 1 means the demo did not reproduce the failure it exists for.
[ "$status" -eq 1 ] || die "expected exit 1 (RESTORE UNUSABLE), got $status"
exit 1
