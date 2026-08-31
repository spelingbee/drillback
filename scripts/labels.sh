#!/usr/bin/env sh
#
# Create the labels this project's workflows and templates refer to.
#
# Idempotent: a label that exists is updated to the colour and description below
# rather than reported as an error, so running this twice is the same as running it
# once. The issue templates and recipe-health.yml apply these labels, and applying a
# label that does not exist makes gh fail, so this has to run before either does.
#
# Usage:
#   scripts/labels.sh              # show what would change (default)
#   scripts/labels.sh --apply      # create and update them
#   scripts/labels.sh --apply --repo owner/name
#
# Creating labels changes a repository. The default is therefore a dry run, and the
# repository is the one `gh` infers from the checkout unless --repo says otherwise.

set -eu

APPLY=0
REPO=""

while [ "$#" -gt 0 ]; do
  case "$1" in
  --apply) APPLY=1 ;;
  --dry-run) APPLY=0 ;;
  --repo)
    shift
    [ "$#" -gt 0 ] || {
      echo "labels: --repo needs a value" >&2
      exit 2
    }
    REPO="$1"
    ;;
  -h | --help)
    sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    echo "labels: unknown argument $1" >&2
    exit 2
    ;;
  esac
  shift
done

command -v gh >/dev/null 2>&1 || {
  echo "labels: gh is not on PATH" >&2
  exit 2
}

if [ -n "$REPO" ]; then
  set -- --repo "$REPO"
else
  set --
fi

# name|colour|description
#
# The colours are chosen so the two that matter most to a newcomer - `good first
# issue` and `recipe` - are the ones that stand out in a list.
LABELS='
recipe|0e8a16|A recipe for one application: two YAML files and no Go
good first issue|7057ff|A good place to start. Small, self-contained, and reviewed quickly
help wanted|008672|Nobody is working on this and somebody should
recipe-broken|b60205|An existing recipe stopped working, usually because upstream moved
source|1d76db|Reading a backup: restic, borg, kopia, a tarball
notifier|5319e7|Telling somebody the drill failed
hint|fbca04|A rule in docs/hints.yaml that turns an error into an explanation
check-type|c2e0c6|The expect vocabulary a recipe asserts with
security|B60205|Changes a trust boundary; wants a design note before code
bug|d73a4a|drillback did something wrong
enhancement|a2eeef|Something drillback should be able to do and cannot
'

printf '%s\n' "$LABELS" | while IFS='|' read -r name colour description; do
  [ -n "$name" ] || continue
  if [ "$APPLY" -eq 0 ]; then
    printf 'would create or update  %-18s #%s  %s\n' "$name" "$colour" "$description"
    continue
  fi
  # --force updates a label that already exists instead of failing, which is what
  # makes this safe to run again.
  if gh label create "$name" --color "$colour" --description "$description" --force "$@"; then
    printf 'ok  %s\n' "$name"
  else
    printf 'FAILED  %s\n' "$name" >&2
  fi
done

if [ "$APPLY" -eq 0 ]; then
  echo
  echo "This was a dry run. Nothing was created. Re-run with --apply."
fi
