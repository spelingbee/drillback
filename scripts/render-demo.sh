#!/usr/bin/env sh
#
# Render docs/demo/demo.gif from docs/demo/demo.tape.
#
# The GIF in the README is a recording of a real run: this script runs the same
# scripts/demo.sh and scripts/demo-broken.sh that `make demo` runs, inside a terminal
# vhs is recording. Nothing is staged and nothing is typed by hand. See CLAUDE.md,
# "Never hand-write demo output" - a GIF is output too.
#
# Needs: vhs, ttyd, ffmpeg, a reachable docker daemon, restic. On a Mac:
#   brew install vhs
# On Linux, see https://github.com/charmbracelet/vhs#installation
#
# If you would rather not install four things, this is how the committed GIF was
# actually rendered - charm's vhs image with the docker CLI, the compose plugin and
# restic added, driving the host daemon through a mounted socket:
#
#   cat > /tmp/vhs.Dockerfile <<EOF
#   FROM ghcr.io/charmbracelet/vhs:latest
#   RUN apt-get update -qq && apt-get install -y --no-install-recommends #         docker-cli docker-compose restic ca-certificates
#   EOF
#   docker build -t restored-vhs -f /tmp/vhs.Dockerfile /tmp
#
#   # The workspace must be a path the daemon and the container both resolve, for the
#   # reason docs/docker.md calls the same-path rule.
#   sudo mkdir -p /var/lib/restored-demo
#   docker run --rm -u 0 #     -v /var/run/docker.sock:/var/run/docker.sock #     -v /var/lib/restored-demo:/var/lib/restored-demo #     -v "$PWD:/work" -e TMPDIR=/var/lib/restored-demo -w /work #     --entrypoint sh restored-vhs -c 'vhs docs/demo/demo.tape'
#
# It takes several minutes. It stands up Gitea and PostgreSQL twice, takes two real
# restic backups, throws both stacks away, and restores them.
set -eu
cd "$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"

die() { printf 'render-demo.sh: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

have vhs    || die "vhs is not installed. See https://github.com/charmbracelet/vhs#installation"
have ttyd   || die "ttyd is not installed, and vhs cannot record without it."
have ffmpeg || die "ffmpeg is not installed, and vhs cannot encode without it."
have restic || die "restic is not installed. The demo takes a real backup with it."
have docker || die "docker is not installed."
docker info >/dev/null 2>&1 || die "the docker daemon is not reachable. Start it and try again."

[ -x bin/restored ] || {
  printf 'building bin/restored first...\n'
  go build -o bin/restored ./cmd/restored
}

printf 'Rendering docs/demo/demo.gif. This runs two real restore drills and takes a few minutes.\n'
vhs docs/demo/demo.tape

[ -f docs/demo/demo.gif ] || die "vhs finished but docs/demo/demo.gif was not written."

# A GIF that is suspiciously small usually means the tape raced ahead of the run and
# recorded an empty terminal. Better to fail here than to commit a blank demo.
size=$(wc -c < docs/demo/demo.gif)
[ "$size" -gt 100000 ] || die "docs/demo/demo.gif is only $size bytes, which means the
  recording is almost certainly empty. Check that scripts/demo.sh succeeds on its own
  first, then re-run this."

printf 'Wrote docs/demo/demo.gif (%s bytes).\n' "$size"
printf 'Watch it before committing it. If either run did not reach a verdict, the tape\n'
printf 'timed out and the GIF is wrong.\n'
