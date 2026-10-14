#!/usr/bin/env bash
# Regenerates pkg/pumpx2/messages/parity/testdata/catalog.json, what the jar says about every message
# class, from a pumpX2 cliparser jar:
#   tools/pumpx2catalog/regen.sh path/to/cliparser.jar
# Needs a JDK (javac, java) and python3. Then re-record the jar's answers with
#   FAKETANDEM_TEST_CLIPARSER_JAR=path/to/cliparser.jar go test ./pkg/pumpx2/messages/parity -update-golden
# and fix the Go message types until go test ./pkg/pumpx2/... passes.
set -euo pipefail

jar="${1:?usage: regen.sh path/to/cliparser.jar}"
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT

javac -nowarn -cp "$jar" -d "$out" "$here/CatalogDump.java"
java -cp "$jar:$out" CatalogDump 2>/dev/null | python3 -m json.tool --sort-keys --indent 1 > "$out/catalog.json"
cp "$out/catalog.json" "$root/pkg/pumpx2/messages/parity/testdata/catalog.json"
echo "wrote $root/pkg/pumpx2/messages/parity/testdata/catalog.json"
