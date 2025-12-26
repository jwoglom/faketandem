#!/bin/sh
# Wrapper script to run pumpX2 cliparser with gradle
# Usage: cliparser-gradle.sh <pumpx2_path> [cliparser args...]
#
# This script changes to the pumpX2 directory and runs gradlew with all
# arguments passed through via --args

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <pumpx2_path> [cliparser args...]" >&2
    exit 1
fi

PUMPX2_PATH="$1"
shift

cd "$PUMPX2_PATH"

# Pass all remaining arguments to cliparser via --args
exec ./gradlew cliparser -q --console=plain --args="$*"
