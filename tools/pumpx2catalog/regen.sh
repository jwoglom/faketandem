#!/usr/bin/env bash
# Regenerates pkg/pumpx2/messages/catalog_gen.go from a pumpX2 cliparser jar:
#   tools/pumpx2catalog/regen.sh path/to/cliparser.jar
# Needs a JDK (javac, java) and Go. After regenerating, run the parity tests against
# the same jar (FAKETANDEM_TEST_CLIPARSER_JAR) and refresh the golden vectors.
set -euo pipefail

jar="${1:?usage: regen.sh path/to/cliparser.jar}"
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT

javac -nowarn -cp "$jar" -d "$out" "$here/CatalogDump.java"
java -cp "$jar:$out" CatalogDump 2>/dev/null > "$out/catalog.json"
(cd "$root" && go run "$here/gencatalog.go" < "$out/catalog.json" > "$out/catalog_gen.go")
cp "$out/catalog_gen.go" "$root/pkg/pumpx2/messages/catalog_gen.go"
echo "wrote $root/pkg/pumpx2/messages/catalog_gen.go"
