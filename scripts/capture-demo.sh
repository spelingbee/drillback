#!/usr/bin/env sh
#
# Capture what the tool really prints, into docs/demo/*.txt.
#
# Nothing in docs/demo/ is ever written by hand. Everything the README shows as
# terminal output comes from here, from a real run against a real backup. The only
# processing is mechanical: the demo's own progress lines are dropped and the report
# that follows the "== restored check" marker is kept verbatim.

set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"

mkdir -p docs/demo

# A pinned scratch directory, so re-capturing changes the numbers and not the paths.
RESTORED_DEMO_DIR=${RESTORED_DEMO_DIR:-${TMPDIR:-/tmp}/restored-demo}
export RESTORED_DEMO_DIR

capture() {
  script=$1
  dest=$2
  expected=$3
  printf 'capturing %s -> %s\n' "$script" "$dest"

  set +e
  out=$("$script" 2>&1)
  status=$?
  set -e
  if [ "$status" -ne "$expected" ]; then
    printf '%s\n' "$out" >&2
    printf 'capture-demo: %s exited %s, expected %s\n' "$script" "$status" "$expected" >&2
    exit 1
  fi

  printf '%s\n' "$out" |
    sed -n '/^== restored check$/,$p' |
    sed '1d' |
    sed '/^== restored exited /d' |
    sed -e :a -e '/^[[:space:]]*$/{$d;N;ba' -e '}' >"$dest"
}

capture ./scripts/demo.sh docs/demo/pass.txt 0
capture ./scripts/demo-broken.sh docs/demo/fail.txt 1
capture ./scripts/demo-kuma.sh docs/demo/kuma.txt 0

# The README is not written by hand either: the captured text is spliced between the
# markers that name each file.
embed() {
  file=$1
  awk -v f="$file" '
    BEGIN { while ((getline line < f) > 0) body = body line "\n" }
    $0 == "<!-- BEGIN " f " -->" { print; print "```text"; printf "%s", body; print "```"; skip = 1; next }
    $0 == "<!-- END " f " -->"   { skip = 0 }
    !skip { print }
  ' README.md >README.tmp && mv README.tmp README.md
}

embed docs/demo/pass.txt
embed docs/demo/fail.txt

printf '\nwrote:\n'
for f in docs/demo/pass.txt docs/demo/fail.txt docs/demo/kuma.txt; do
  printf '  %-22s %s lines\n' "$f" "$(wc -l <"$f" | tr -d ' ')"
done
