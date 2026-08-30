#!/usr/bin/env sh
#
# drillback installer.
#
#   curl -fsSL https://raw.githubusercontent.com/spelingbee/drillback/main/install.sh | sh
#
# What it does, in order: work out the OS and the architecture, download the matching
# release archive and the release checksum file, verify the archive against the
# checksum, unpack it into a temporary directory, and move the binary into place.
#
# It installs to ~/.local/bin by default and never needs root to do it. --system
# installs to /usr/local/bin, which usually does. Running the whole script as root
# without --system is refused, because it produces a binary in /root/.local/bin that
# the user who asked for it cannot run.
#
# Flags:
#   --version vX.Y.Z   install this release instead of the latest
#   --system           install to /usr/local/bin (or --dir) rather than ~/.local/bin
#   --dir PATH         install to PATH
#   --no-verify        skip checksum verification. Do not use this.
#   --help
set -eu

REPO="spelingbee/drillback"
BINARY="drillback"
VERSION=""
DIR=""
SYSTEM=0
VERIFY=1

say()  { printf '%s\n' "$*"; }
warn() { printf '%s\n' "$*" >&2; }
die()  { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Install $BINARY.

Usage: install.sh [--version vX.Y.Z] [--system] [--dir PATH] [--no-verify]

  --version vX.Y.Z  install a specific release. Default: the latest release.
  --system          install to /usr/local/bin instead of \$HOME/.local/bin.
  --dir PATH        install to PATH instead of either default.
  --no-verify       do not verify the SHA-256 checksum. Not recommended.
  --help            print this and exit.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version) [ $# -ge 2 ] || die "--version needs a value, for example --version v0.1.0"
               VERSION="$2"; shift 2 ;;
    --version=*) VERSION="${1#--version=}"; shift ;;
    --system)  SYSTEM=1; shift ;;
    --dir)     [ $# -ge 2 ] || die "--dir needs a path"
               DIR="$2"; shift 2 ;;
    --dir=*)   DIR="${1#--dir=}"; shift ;;
    --no-verify) VERIFY=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown option $1. Run install.sh --help." ;;
  esac
done

# Running as root without --system installs into root's home, which is not what
# anyone means. Ask for the flag rather than guessing.
if [ "$(id -u 2>/dev/null || echo 0)" = "0" ] && [ "$SYSTEM" -eq 0 ] && [ -z "$DIR" ]; then
  die "refusing to run as root without --system.
  Either run it as your own user, which installs to \$HOME/.local/bin,
  or run it as root with --system, which installs to /usr/local/bin."
fi

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required and was not found in PATH."; }
need uname
need tar

if command -v curl >/dev/null 2>&1; then
  fetch()      { curl -fsSL "$1" -o "$2"; }
  fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch()      { wget -qO "$2" "$1"; }
  fetch_stdout() { wget -qO - "$1"; }
else
  die "either curl or wget is required and neither was found in PATH."
fi

os_raw="$(uname -s)"
case "$os_raw" in
  Linux)   OS=linux ;;
  Darwin)  OS=darwin ;;
  MINGW*|MSYS*|CYGWIN*)
    die "this script does not install on Windows.
  Download the .zip from https://github.com/$REPO/releases and unpack it,
  or run: go install github.com/$REPO/cmd/$BINARY@latest" ;;
  *) die "unsupported operating system: $os_raw.
  Build from source instead: go install github.com/$REPO/cmd/$BINARY@latest" ;;
esac

arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported architecture: $arch_raw. $BINARY ships amd64 and arm64 only.
  Build from source instead: go install github.com/$REPO/cmd/$BINARY@latest" ;;
esac

if [ -z "$VERSION" ]; then
  say "Looking up the latest release of $REPO..."
  VERSION="$(fetch_stdout "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name"[ ]*:[ ]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$VERSION" ] || die "could not determine the latest release of $REPO.
  The repository may have no releases yet, or the GitHub API may be rate-limiting you.
  Pass one explicitly: install.sh --version v0.1.0"
fi
case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac
NUM="${VERSION#v}"

if [ -n "$DIR" ]; then
  TARGET="$DIR"
elif [ "$SYSTEM" -eq 1 ]; then
  TARGET="/usr/local/bin"
else
  TARGET="$HOME/.local/bin"
fi

ARCHIVE="${BINARY}_${NUM}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"

TMP="$(mktemp -d 2>/dev/null || mktemp -d -t drillback)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT INT TERM

say "Downloading $ARCHIVE ($VERSION, $OS/$ARCH)..."
fetch "$BASE/$ARCHIVE" "$TMP/$ARCHIVE" || die "could not download $BASE/$ARCHIVE
  Check that $VERSION exists and ships a $OS/$ARCH build:
  https://github.com/$REPO/releases/tag/$VERSION"

if [ "$VERIFY" -eq 1 ]; then
  if command -v sha256sum >/dev/null 2>&1; then
    sum() { sha256sum "$1" | cut -d' ' -f1; }
  elif command -v shasum >/dev/null 2>&1; then
    sum() { shasum -a 256 "$1" | cut -d' ' -f1; }
  else
    die "neither sha256sum nor shasum was found, so the download cannot be verified.
  Install one of them, or re-run with --no-verify if you accept the risk."
  fi

  say "Verifying the checksum..."
  fetch "$BASE/checksums.txt" "$TMP/checksums.txt" \
    || die "could not download $BASE/checksums.txt, so the download cannot be verified."

  want="$(grep " \*\{0,1\}$ARCHIVE\$" "$TMP/checksums.txt" | cut -d' ' -f1 | head -n 1)"
  [ -n "$want" ] || die "checksums.txt does not list $ARCHIVE. Refusing to install an unverified binary."
  got="$(sum "$TMP/$ARCHIVE")"
  if [ "$want" != "$got" ]; then
    die "CHECKSUM MISMATCH for $ARCHIVE.
  expected $want
  got      $got
  The download is corrupt or has been tampered with. Nothing was installed."
  fi
  say "Checksum OK."
else
  warn "Skipping checksum verification because --no-verify was passed."
fi

tar -xzf "$TMP/$ARCHIVE" -C "$TMP" || die "could not unpack $ARCHIVE."
[ -f "$TMP/$BINARY" ] || die "$ARCHIVE did not contain a $BINARY binary. This is a packaging bug; please report it at https://github.com/$REPO/issues"

mkdir -p "$TARGET" 2>/dev/null || die "could not create $TARGET.
  Either choose a writable directory with --dir PATH, or re-run with sudo and --system."
chmod +x "$TMP/$BINARY"

if ! mv "$TMP/$BINARY" "$TARGET/$BINARY" 2>/dev/null; then
  die "could not write $TARGET/$BINARY.
  Either choose a writable directory with --dir PATH, or re-run as: sudo sh install.sh --system"
fi

say ""
say "Installed $TARGET/$BINARY"
"$TARGET/$BINARY" version || true

case ":$PATH:" in
  *":$TARGET:"*) ;;
  *)
    say ""
    warn "$TARGET is not on your PATH. Add it:"
    warn "    export PATH=\"$TARGET:\$PATH\""
    warn "and put that line in your shell profile."
    ;;
esac

say ""
say "Next: $BINARY check --recipe gitea --source restic --from /path/to/repo"
say "See https://github.com/$REPO#quick-start"
