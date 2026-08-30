#!/usr/bin/env sh
#
# Open one "Recipe: <app>" issue per line of docs/recipes-wanted.txt.
#
# Fifty issues is a lot of noise to make in one afternoon, and it is not reversible in
# any way that looks good. So:
#
#   - the default is a dry run, and --apply is required to create anything;
#   - --limit exists so the first batch can be five rather than fifty;
#   - it is idempotent by title, so a second run creates nothing and says so.
#
# Usage:
#   scripts/recipes-wanted.sh                       # show what it would open
#   scripts/recipes-wanted.sh --limit 5             # the first five, still a dry run
#   scripts/recipes-wanted.sh --apply --limit 5     # actually open five
#   scripts/recipes-wanted.sh --apply --repo owner/name
#
# Opening issues on a public repository is a stop point: see CLAUDE.md. Nothing here
# runs without --apply, and --apply is a human's decision.

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"

LIST=docs/recipes-wanted.txt
APPLY=0
LIMIT=0
REPO=""

while [ "$#" -gt 0 ]; do
  case "$1" in
  --apply) APPLY=1 ;;
  --dry-run) APPLY=0 ;;
  --limit)
    shift
    [ "$#" -gt 0 ] || {
      echo "recipes-wanted: --limit needs a number" >&2
      exit 2
    }
    LIMIT="$1"
    ;;
  --repo)
    shift
    [ "$#" -gt 0 ] || {
      echo "recipes-wanted: --repo needs a value" >&2
      exit 2
    }
    REPO="$1"
    ;;
  --list)
    shift
    LIST="$1"
    ;;
  -h | --help)
    sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    echo "recipes-wanted: unknown argument $1" >&2
    exit 2
    ;;
  esac
  shift
done

command -v gh >/dev/null 2>&1 || {
  echo "recipes-wanted: gh is not on PATH" >&2
  exit 2
}
[ -f "$LIST" ] || {
  echo "recipes-wanted: no such list $LIST" >&2
  exit 2
}

if [ -n "$REPO" ]; then
  REPO_ARGS="--repo $REPO"
else
  REPO_ARGS=""
fi

# One API call, not one per app: fifty title searches is fifty round trips and a very
# good way to meet a rate limit.
existing=""
if [ "$APPLY" -eq 1 ] || [ -n "${RECIPES_WANTED_CHECK_EXISTING:-}" ]; then
  # shellcheck disable=SC2086
  existing=$(gh issue list $REPO_ARGS --state all --limit 500 --json title --jq '.[].title' 2>/dev/null || true)
fi

count=0
skipped=0
printf 'list:   %s\n' "$LIST"
printf 'mode:   %s\n\n' "$([ "$APPLY" -eq 1 ] && echo "APPLY - issues will be created" || echo "dry run")"

# The body deliberately does two things: it says what makes this application's restore
# interesting, and it links the ten-minute guide. Somebody arriving from a "good first
# issue" search should be able to get from the issue to a passing round trip without
# opening anything else.
while IFS='|' read -r name stars repo description; do
  case "$name" in
  '#'* | '') continue ;;
  esac
  name=$(printf '%s' "$name" | sed 's/^ *//; s/ *$//')
  repo=$(printf '%s' "$repo" | sed 's/^ *//; s/ *$//')
  description=$(printf '%s' "$description" | sed 's/^ *//; s/ *$//')
  stars=$(printf '%s' "$stars" | sed 's/^ *//; s/ *$//')
  [ -n "$name" ] || continue

  title="Recipe: $name"

  if printf '%s\n' "$existing" | grep -Fxq "$title"; then
    printf 'skip    %s (an issue with that title already exists)\n' "$title"
    skipped=$((skipped + 1))
    continue
  fi

  if [ "$LIMIT" -gt 0 ] && [ "$count" -ge "$LIMIT" ]; then
    printf '\nstopping at --limit %s\n' "$LIMIT"
    break
  fi

  body=$(
    cat <<BODY
**$name** has no recipe yet. https://github.com/$repo - $description

A recipe teaches \`restored\` how to stand this application up from a backup and how to
tell whether the restore actually worked. It is two YAML files, it needs no Go, and if
you run $name yourself you are the best-placed person to write it.

**Start here:** [Add a recipe in 10 minutes](https://github.com/spelingbee/drillback/blob/main/CONTRIBUTING.md#add-a-recipe-in-10-minutes)

If you already have a \`docker-compose.yml\` for it, the first draft is one command:

\`\`\`sh
drillback recipe init $name --compose ~/docker/$name/docker-compose.yml
\`\`\`

That turns your volumes into inputs, recognises a PostgreSQL or SQLite service, and
leaves a TODO everywhere the answer is yours.

### The one hard question

**What would prove the restore worked?** Name something a real installation of $name
has and a fresh one does not: a row in a particular table, a file in a particular
directory, an API listing that comes back empty on day one. "The home page loads" is
not it - that works against an empty database, which is exactly the failure this tool
exists to catch.

\`drillback recipe test\` enforces that mechanically: it starts your stack with empty
inputs and rejects the recipe if every check still passes.

### What CI will do

Run the round trip and report one verdict, inside fifteen minutes. A pull request that
touches only \`recipes/$name/**\` needs nothing else to be green.

---

Not sure it fits? Say so here - some applications keep no state worth restoring, and
some cannot round trip inside the budget. Finding that out and writing it down is
itself a useful contribution.

<sub>Opened from \`docs/recipes-wanted.txt\` ($stars stars at the time that list was gathered).</sub>
BODY
  )

  if [ "$APPLY" -eq 0 ]; then
    printf 'would open  %s\n' "$title"
  else
    # shellcheck disable=SC2086
    if gh issue create $REPO_ARGS \
      --title "$title" \
      --label recipe --label "good first issue" \
      --body "$body"; then
      printf 'opened  %s\n' "$title"
    else
      printf 'FAILED  %s\n' "$title" >&2
    fi
  fi
  count=$((count + 1))
done <"$LIST"

printf '\n%s issue(s), %s already existed\n' "$count" "$skipped"
if [ "$APPLY" -eq 0 ]; then
  echo
  echo "This was a dry run. Nothing was created."
  echo "Run scripts/labels.sh --apply first: gh refuses a label that does not exist."
fi
