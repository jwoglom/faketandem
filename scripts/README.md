# Scripts

This directory contains utility scripts for the faketandem project.

## Git Hooks

### Pre-push Hook

The `pre-push.sh` script runs `golangci-lint` before pushing code to ensure no linting issues are introduced.

#### Installation

The hook should be automatically set up when you clone the repository. If not, run:

```bash
ln -sf ../../scripts/pre-push.sh .git/hooks/pre-push
chmod +x scripts/pre-push.sh
```

#### Usage

The hook runs automatically before every `git push`. If linting issues are found:

1. **Fix automatically** (where possible):
   ```bash
   golangci-lint run --fix
   ```

2. **Skip the check** (not recommended):
   ```bash
   git push --no-verify
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
