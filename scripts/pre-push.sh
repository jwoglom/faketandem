#!/bin/bash

# Pre-push hook to run golangci-lint
# This prevents pushing code with linting issues

echo "Running golangci-lint before push..."

# Check if golangci-lint is available
if ! command -v golangci-lint &> /dev/null; then
    # Try the local installation
    if [ -f "/usr/local/bin/golangci-lint" ]; then
        GOLANGCI_LINT="/usr/local/bin/golangci-lint"
    else
        echo "⚠️  golangci-lint not found. Skipping lint check."
        echo "   Install it with: brew install golangci-lint"
        exit 0
    fi
else
    GOLANGCI_LINT="golangci-lint"
fi

# Run the linter
$GOLANGCI_LINT run --timeout=5m

# Check the exit code
if [ $? -ne 0 ]; then
    echo ""
    echo "❌ golangci-lint found issues!"
    echo ""
    echo "To fix automatically (where possible), run:"
    echo "  golangci-lint run --fix"
    echo ""
    echo "To skip this check, use:"
    echo "  git push --no-verify"
    echo ""
    exit 1
fi

echo "✅ golangci-lint passed!"
exit 0
