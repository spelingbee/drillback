#!/usr/bin/env sh
#
# Open one `help wanted` issue per P2/P3 finding from the session 4 reviews.
#
# These are contributor entry points. The brief that produced them said not to hoard
# them, and a project measured by the number of distinct external contributors with
# merged pull requests should not arrive at launch having already done every job small
# enough for a stranger to finish.
#
# Same shape as scripts/recipes-wanted.sh, for the same reasons:
#
#   - the default is a dry run, and --apply is required to create anything;
#   - --limit exists so the first batch can be five rather than thirty;
#   - it is idempotent by title, so a second run creates nothing and says so.
#
# Usage:
#   scripts/backlog-issues.sh                     # show what it would open
#   scripts/backlog-issues.sh --limit 5           # the first five, still a dry run
#   scripts/backlog-issues.sh --apply --limit 5   # actually open five
#   scripts/backlog-issues.sh --apply --repo owner/name
#
# Opening issues on a public repository is a stop point: see CLAUDE.md. Nothing here
# runs without --apply, and --apply is a human's decision. The labels have to exist
# first: run scripts/labels.sh --apply before this.

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"

APPLY=0
LIMIT=0
REPO=""

while [ "$#" -gt 0 ]; do
  case "$1" in
  --apply) APPLY=1 ;;
  --dry-run) APPLY=0 ;;
  --limit)
    shift
    [ "$#" -gt 0 ] || { echo "backlog-issues: --limit needs a number" >&2; exit 2; }
    LIMIT="$1"
    ;;
  --repo)
    shift
    [ "$#" -gt 0 ] || { echo "backlog-issues: --repo needs a value" >&2; exit 2; }
    REPO="$1"
    ;;
  -h | --help)
    sed -n '2,24p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    echo "backlog-issues: unknown argument $1" >&2
    exit 2
    ;;
  esac
  shift
done

# id|review file|extra labels|title
#
# Every body points at the finding, which carries the file:line, the reproduction and a
# proposed fix. An issue that only says what is wrong costs the first contributor an
# hour of archaeology.
ISSUES='
UX-15|ux|good first issue|--keep does not say what it kept, and prints a cleanup command that is POSIX-only
UX-17|ux|good first issue|The report is fixed at 78 columns and ignores COLUMNS
UX-16|ux|good first issue|Four flag combinations are accepted without complaint
UX-14|ux|good first issue|JSON error reports carry null arrays and an empty run block
ARCH-15|architecture|good first issue|Four discarded parameters and one dead struct field
ARCH-13|architecture|good first issue|Hint selection prioritises subject order over rule order, against SPEC 6.1
FC-11|fresh-clone|good first issue|The db/tables-empty hint is written about Gitea and printed for every recipe
UX-08|ux||recipe init tells you to run a command that exits 2, and never mentions recipe test
UX-10|ux||Raw OS and Go plumbing errors reach the user in five places
UX-11|ux||check --help dropped the Environment block and every --input example
UX-12|ux||The ASCII fallback is undocumented, unreachable by flag, and incomplete
FC-07|fresh-clone||docs/recipe-spec.md drops item-level constraints, so a recipe written from it does not validate
FC-08|fresh-clone||The --compose scaffold loses the only documentation of profiles: [test] and RESTORED_TEST_ASSETS
FC-09|fresh-clone||The generated recipe README omits an input and names the database after the service
FC-10|fresh-clone||The contribution link is missing for a commented recipe, and its fallback cp is wrong on Windows
FC-13|fresh-clone||Every documented command goes through make, with no fallback for a host without it
FC-15|fresh-clone||An interrupted run left a Docker network behind
MNT-09|maintainer||Four of the ten labels are created and never applied by anything
MNT-11|maintainer||Fifty good first issue tickets, and no way to claim one
MNT-12|maintainer||CONTRIBUTING omits the licensing, commit-message and question answers a first-timer needs
MNT-13|maintainer||recipe validate --strict passes the TEMPLATE verbatim, so the placeholder loop costs a Docker cycle
MNT-14|maintainer||The all-contributors flow cannot run, and CONTRIBUTING does not mention it
MNT-15|maintainer||A recipe-health issue does not name the file to edit, and assumes a binary nobody can install
MNT-18|maintainer||The sequential fallback in recipes.yml drops the debug log and merges all verdicts
ARCH-06|architecture||restic command lines and stderr never reach the run debug log
ARCH-14|architecture||internal/workspace does not own three paths inside the workspace
ARCH-16|architecture||The recipe format is described in four places
SEC-11|security||The vars secret warning specified in SPEC.md 9.3 is not implemented
SEC-12|security||Templates in default_path are silently not expanded, and the .. guard runs pre-render
SEC-13|security||Defence in depth: an unreachable .. check, and argv passed to compose exec without --
SEC-06|security|security|expect.glob escapes the workspace and turns the report into a host filesystem oracle
SEC-08|security|security|The JSON report embeds 200 lines of every container log
SEC-09|security|security|A sql check file: is unconstrained and opens arbitrary host paths
ARCH-09|architecture|security|A killed run leaves a full copy of the backup in the temp directory, and nothing finds it
ARCH-11|architecture||Every failure is an untyped string, which is the wall the notifiers will hit
ARCH-08|architecture||No test seam at the docker boundary: probe, runner, compose, sqlite, dir are all 0%
ARCH-07|architecture||Two lifecycle implementations that share code by copy
ARCH-10|architecture||recipe test grows the restic cache without bound
'

command -v gh >/dev/null 2>&1 || {
  echo "backlog-issues: the gh CLI is required" >&2
  exit 2
}
gh_args=""
[ -n "$REPO" ] && gh_args="--repo $REPO"

existing=""
if [ "$APPLY" -eq 1 ]; then
  # shellcheck disable=SC2086
  existing=$(gh issue list $gh_args --state all --limit 500 --json title --jq '.[].title' 2>/dev/null || true)
fi

count=0
created=0
skipped=0

printf '%s\n' "$ISSUES" | while IFS='|' read -r id review labels title; do
  [ -n "$id" ] || continue
  count=$((count + 1))
  if [ "$LIMIT" -gt 0 ] && [ "$count" -gt "$LIMIT" ]; then
    break
  fi

  full="$id: $title"
  body="Found by the independent reviews before the first public release.

**The finding, with the file:line, a reproduction and a proposed fix, is in
[\`docs/review/$review.md\`](https://github.com/spelingbee/restored/blob/main/docs/review/$review.md)
under \`$id\`.** Read that first; it is more use than this issue body.

This is a good thing to pick up. If any part of it is unclear, or the finding turns out
to be wrong, say so on this issue - a reviewer being wrong is a normal outcome and
closing this as \"not a bug, and here is why\" is a real contribution.

See [CONTRIBUTING.md](https://github.com/spelingbee/restored/blob/main/CONTRIBUTING.md)
to get set up. \`go test ./...\` is green with neither Docker nor restic installed."

  label_args="--label 'help wanted'"
  [ -n "$labels" ] && label_args="$label_args --label '$labels'"

  if [ "$APPLY" -eq 0 ]; then
    printf 'would open: %s\n            labels: help wanted%s\n' \
      "$full" "$([ -n "$labels" ] && printf ', %s' "$labels")"
    continue
  fi

  if printf '%s\n' "$existing" | grep -Fxq "$full"; then
    printf 'exists:     %s\n' "$full"
    skipped=$((skipped + 1))
    continue
  fi

  # shellcheck disable=SC2086
  if eval gh issue create $gh_args --title "\"\$full\"" --body "\"\$body\"" $label_args >/dev/null; then
    printf 'opened:     %s\n' "$full"
    created=$((created + 1))
  else
    printf 'FAILED:     %s\n' "$full" >&2
  fi
done

if [ "$APPLY" -eq 0 ]; then
  echo
  echo "That was a dry run. Nothing was created."
  echo "Run scripts/labels.sh --apply first, then this with --apply."
  echo "Opening issues on a public repository is a stop point: see CLAUDE.md."
fi
