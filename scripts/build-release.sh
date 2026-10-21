#!/usr/bin/env bash
# Builds one release asset: a static faketandem binary for GOOS/GOARCH, as
# .github/workflows/release.yml publishes them.
#
#   scripts/build-release.sh <goos> <goarch> <version> [outdir]
#
# The binary is <outdir>/faketandem-<version>-<goos>-<goarch> (32-bit arm: -armv7 or -armv6, after
# GOARM, 7 unless set; .exe on windows), and is all a release ships: the web UI is built in, and
# CGO is off, so the Linux BLE transport (pure Go) and the virtual transport everywhere else need
# nothing installed.
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

export GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0
arch="$goarch"
if [[ "$goarch" == "arm" ]]; then
  export GOARM="${GOARM:-7}"
  arch="armv${GOARM}"
fi
name="faketandem-${version}-${goos}-${arch}"
if [[ "$goos" == "windows" ]]; then
  name="$name.exe"
fi

mkdir -p "$outdir"
out="$(cd "$outdir" && pwd)/$name"
(cd "$root" && go build -trimpath -ldflags "-s -w -X main.version=${version}" -o "$out" .)
echo "$outdir/$name"
