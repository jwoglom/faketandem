# Scripts

This directory contains utility scripts for the faketandem project.

## Git Hooks

### Pre-push Hook

The `pre-push.sh` script automatically fixes linting issues where possible and checks for remaining issues before pushing code.

#### How It Works

1. **Auto-fix**: Runs `golangci-lint run --fix` to automatically fix formatting and other auto-fixable issues
2. **Stage changes**: If files were modified, they're automatically staged with `git add -u`
3. **Verify**: Runs `golangci-lint run` again to check for remaining issues
4. **Block or allow**: Blocks the push if non-fixable issues remain

#### Installation

The hook should be automatically set up when you clone the repository. If not, run:

```bash
ln -sf ../../scripts/pre-push.sh .git/hooks/pre-push
chmod +x scripts/pre-push.sh
```

#### Usage

The hook runs automatically before every `git push`. It will:

- ✅ **Auto-fix** formatting issues (gofmt, goimports, etc.)
- ✅ **Stage** the fixed files automatically
- ❌ **Block** the push if non-fixable issues remain (errcheck, gocyclo, etc.)

**To skip the check** (not recommended):
```bash
git push --no-verify
```

**To manually run the same process**:
```bash
golangci-lint run --fix
git add -u
git commit --amend --no-edit  # if you want to include fixes in last commit
```

#### Requirements

- `golangci-lint` must be installed. Install with:
  ```bash
  # macOS
  brew install golangci-lint
  
  # Linux
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2
  ```

## Other Scripts

### packet.py

Python script for packet analysis and testing.

**Requirements:**
```bash
pip install -r requirements.txt
```
