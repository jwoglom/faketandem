#!/usr/bin/env bash
# Builds one release tarball: a static faketandem binary for GOOS/GOARCH, with the web UI beside
# it, as .github/workflows/release.yml publishes them.
#
#   scripts/build-release.sh <goos> <goarch> <version> [outdir]
#
# The tarball is <outdir>/faketandem-<version>-<goos>-<goarch>.tar.gz (32-bit arm: -armv7 or
# -armv6, after GOARM, 7 unless set) and holds `faketandem` and `ui/` at its root, so it unpacks
# straight into a directory the binary serves the UI from (the `ui` directory beside the
# executable). CGO is off: the Linux BLE transport is pure Go, and every other platform builds the
# virtual transport only, so the binary needs nothing installed.
set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "usage: $0 <goos> <goarch> <version> [outdir]" >&2
  exit 2
fi

goos="$1"
goarch="$2"
version="$3"
outdir="${4:-dist}"
root="$(cd "$(dirname "$0")/.." && pwd)"
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT

export GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0
arch="$goarch"
if [[ "$goarch" == "arm" ]]; then
  export GOARM="${GOARM:-7}"
  arch="armv${GOARM}"
fi
name="faketandem-${version}-${goos}-${arch}"

binary="faketandem"
if [[ "$goos" == "windows" ]]; then
  binary="faketandem.exe"
fi

(cd "$root" && go build -trimpath -ldflags "-s -w -X main.version=${version}" -o "$stage/$binary" .)
cp -R "$root/ui" "$stage/ui"

mkdir -p "$outdir"
tar -C "$stage" -czf "$outdir/$name.tar.gz" "$binary" ui
echo "$outdir/$name.tar.gz"
