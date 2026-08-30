#!/usr/bin/env sh
#
# The number this project is trying to move: distinct people, other than the owner and
# other than a bot, with a pull request merged in the trailing 365 days.
#
# Not stars. Not merged pull requests - the same person opening ten is one person, and
# the thing being measured is whether a stranger can arrive, contribute, and leave with
# something merged. Everything in CONTRIBUTING.md, the round-trip harness, and the
# ten-minute guide exists to make this number larger.
#
# Usage:
#   scripts/contributors.sh                     # trailing 365 days
#   scripts/contributors.sh --days 90
#   scripts/contributors.sh --repo owner/name
#   scripts/contributors.sh --json              # for a dashboard
#
# Read-only. It creates nothing and changes nothing.

set -eu

DAYS=365
REPO=""
JSON=0

while [ "$#" -gt 0 ]; do
  case "$1" in
  --days)
    shift
    [ "$#" -gt 0 ] || {
      echo "contributors: --days needs a number" >&2
      exit 2
    }
    DAYS="$1"
    ;;
  --repo)
    shift
    [ "$#" -gt 0 ] || {
      echo "contributors: --repo needs a value" >&2
      exit 2
    }
    REPO="$1"
    ;;
  --json) JSON=1 ;;
  -h | --help)
    sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    echo "contributors: unknown argument $1" >&2
    exit 2
    ;;
  esac
  shift
done

command -v gh >/dev/null 2>&1 || {
  echo "contributors: gh is not on PATH" >&2
  exit 2
}
if [ -z "$REPO" ]; then
  REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
fi
OWNER=${REPO%%/*}

# date -d is GNU, -v is BSD. Both are common enough that it is worth handling each
# rather than requiring one.
if since=$(date -u -d "-${DAYS} days" +%Y-%m-%d 2>/dev/null); then
  :
elif since=$(date -u -v-"${DAYS}"d +%Y-%m-%d 2>/dev/null); then
  :
else
  echo "contributors: cannot compute a date ${DAYS} days ago with this date(1)" >&2
  exit 2
fi

# The owner is excluded because the metric is about other people, and bots are
# excluded because dependabot is not a contributor in the sense being counted.
authors=$(gh pr list --repo "$REPO" --state merged --limit 1000 \
  --json author,mergedAt,number,title \
  --jq "[.[] | select(.mergedAt >= \"${since}\")] | .[] | \"\(.author.login)\t\(.mergedAt[0:10])\t#\(.number) \(.title)\"" |
  grep -v "^${OWNER}	" |
  grep -viE '^(app/|dependabot|renovate|github-actions|.*\[bot\])' || true)

if [ -z "$authors" ]; then
  if [ "$JSON" -eq 1 ]; then
    printf '{"repo":"%s","since":"%s","contributors":0,"merged_pull_requests":0,"people":[]}\n' "$REPO" "$since"
  else
    printf 'repo:   %s\nsince:  %s\n\nNo external contributor has had a pull request merged yet.\n\n' "$REPO" "$since"
    printf 'That is the number this project exists to move. docs/recipes-wanted.txt and\n'
    printf 'scripts/recipes-wanted.sh are the tools for it.\n'
  fi
  exit 0
fi

people=$(printf '%s\n' "$authors" | cut -f1 | sort -u)
distinct=$(printf '%s\n' "$people" | wc -l | tr -d ' ')
merged=$(printf '%s\n' "$authors" | wc -l | tr -d ' ')

if [ "$JSON" -eq 1 ]; then
  # gh carries its own --jq, but composing an object needs the real thing, and this
  # is the only branch that does.
  command -v jq >/dev/null 2>&1 || {
    echo "contributors: --json needs jq on PATH" >&2
    exit 2
  }
  printf '%s\n' "$people" |
    jq -R -s -c --arg repo "$REPO" --arg since "$since" --argjson merged "$merged" \
      '{repo:$repo, since:$since, contributors:(split("\n")|map(select(length>0))|length),
        merged_pull_requests:$merged, people:(split("\n")|map(select(length>0)))}'
  exit 0
fi

printf 'repo:   %s\nsince:  %s\n\n' "$REPO" "$since"
printf '%s\n' "$authors" | sort -k2 | awk -F'\t' '{printf "  %-20s %s  %s\n", $1, $2, $3}'
printf '\n  %s distinct external contributor(s), %s merged pull request(s)\n' "$distinct" "$merged"
