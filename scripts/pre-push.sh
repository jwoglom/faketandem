#!/bin/bash

# Pre-push hook to run golangci-lint
# This automatically fixes issues where possible, then checks for remaining issues

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

# First, try to auto-fix issues
echo "🔧 Auto-fixing issues..."
$GOLANGCI_LINT run --fix --timeout=5m &> /dev/null

# Check if any files were modified by --fix
if ! git diff --quiet; then
    echo "✏️  Auto-fixed some issues. Changes:"
    git diff --stat
    echo ""
    echo "📝 Staging auto-fixed changes..."
    git add -u
fi

# Now run the linter to check for remaining issues
echo "🔍 Checking for remaining issues..."
$GOLANGCI_LINT run --timeout=5m

# Check the exit code
if [ $? -ne 0 ]; then
    echo ""
    echo "❌ golangci-lint found issues that cannot be auto-fixed!"
    echo ""
    echo "Please fix the remaining issues manually, or:"
    echo "  - Review the changes and commit them"
    echo "  - Skip this check with: git push --no-verify"
    echo ""
    exit 1
fi

echo "✅ golangci-lint passed!"
exit 0
