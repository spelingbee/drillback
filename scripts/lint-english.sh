#!/usr/bin/env sh
#
# Every file in this repository is in English, and CI enforces it (ADR-012).
#
# The check is mechanical: after removing the characters on the allowlist below, no
# byte outside 7-bit ASCII may remain. The allowlist is here, in one place, so adding
# to it is a reviewable diff rather than a habit.
#
#   —  em dash            –  en dash             …  ellipsis
#   ·  middle dot         §  section sign        →  right arrow
#   ⇒  double arrow       ▼  down triangle
#   ✔  check mark         ✘  ballot cross        (the report glyphs, and only those)
#   │┌┐└┘├┤┬┴─  box drawing, used in the diagrams
#
# Usage: scripts/lint-english.sh [path...]   (default: everything git tracks)

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"

strip_allowed() {
  sed \
    -e 's/—//g' -e 's/–//g' -e 's/…//g' \
    -e 's/·//g' -e 's/§//g' -e 's/→//g' \
    -e 's/⇒//g' -e 's/▼//g' \
    -e 's/✔//g' -e 's/✘//g' \
    -e 's/│//g' -e 's/┌//g' -e 's/┐//g' -e 's/└//g' -e 's/┘//g' \
    -e 's/├//g' -e 's/┤//g' -e 's/┬//g' -e 's/┴//g' -e 's/─//g'
}

if [ "$#" -gt 0 ]; then
  files=$*
else
  files=$(git ls-files)
fi

found=$(mktemp)
trap 'rm -f "$found"' EXIT

nonascii=$(printf '[\200-\377]')

for f in $files; do
  case "$f" in
  LICENSE | *.png | *.jpg | *.gz | *.zip | *.exe) continue ;;
  esac
  [ -f "$f" ] || continue

  strip_allowed <"$f" | grep -n "$nonascii" 2>/dev/null | sed "s|^|$f:|" >>"$found" || true
done

if [ -s "$found" ]; then
  cat "$found"
  printf '\nlint-english: non-English characters found. Every file here is in English\n' >&2
  printf 'lint-english: (ADR-012). If a character belongs on the allowlist, add it to\n' >&2
  printf 'lint-english: scripts/lint-english.sh in the same commit that uses it.\n' >&2
  exit 1
fi
printf 'lint-english: ok\n'
