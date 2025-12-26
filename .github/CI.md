# Continuous Integration (CI) Documentation

This document describes the CI/CD pipeline for the faketandem project.

## Overview

The CI pipeline runs on every push and pull request to ensure code quality and functionality. It consists of four main jobs:

1. **Test** - Unit tests with coverage
2. **Integration Tests** - Tests with real pumpX2 integration
3. **Build** - Verify the binary builds correctly
4. **Lint** - Code quality and style checks

## Jobs Description

### 1. Test Job

**Purpose:** Run unit tests and generate coverage reports

**Matrix Strategy:**
- Tested on Go versions: 1.21, 1.22
- Ensures compatibility across Go versions

**Steps:**
1. Checkout code
2. Set up Go with caching
3. Download and verify dependencies
4. Run `go vet` for static analysis
5. Run unit tests with race detection and coverage
6. Upload coverage to Codecov
7. Run benchmarks (informational)

**Commands:**
```bash
go vet ./...
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go test -bench=. -benchmem ./pkg/handler
```

**Success Criteria:**
- All tests pass
- No race conditions detected
- `go vet` reports no issues

### 2. Integration Tests Job

**Purpose:** Test with actual pumpX2 jpake-server integration

**Steps:**
1. Checkout faketandem repository
2. Checkout pumpX2 repository (dev branch)
3. Set up Go 1.22 and Java 17
4. Build pumpX2 cliparser with Gradle
5. Verify cliparser.jar exists
6. Run integration tests with PUMPX2_PATH set

**Environment:**
- `PUMPX2_PATH`: Set to pumpX2 checkout location
- Timeout: 5 minutes

**Commands:**
```bash
cd pumpX2
./gradlew cliparser --console=plain -q

cd faketandem
export PUMPX2_PATH=${{ github.workspace }}/pumpX2
go test -v -timeout 5m ./pkg/handler -run Integration
```

**Success Criteria:**
- pumpX2 builds successfully
- cliparser.jar is created
- Integration tests pass

**Note:** This job checks out pumpX2 from the dev branch where jpake-server changes are available.

### 3. Build Job

**Purpose:** Verify the binary builds successfully

**Steps:**
1. Checkout code
2. Set up Go 1.22
3. Build binary: `go build -v -o faketandem .`
4. Verify binary exists and runs
5. Upload binary as artifact

**Artifacts:**
- Binary name: `faketandem-{OS}-{commit-sha}`
- Retention: 7 days
- Available for download from GitHub Actions UI

**Success Criteria:**
- Binary builds without errors
- Binary executable exists
- Help text displays correctly

### 4. Lint Job

**Purpose:** Enforce code quality and style standards

**Linters Enabled:**
- gofmt - Code formatting
- govet - Static analysis
- errcheck - Unchecked errors
- staticcheck - Advanced static analysis
- unused - Unused code detection
- gosimple - Simplification suggestions
- ineffassign - Ineffective assignments
- gocyclo - Cyclomatic complexity
- misspell - Spelling errors
- revive - Golint replacement
- goimports - Import organization

**Configuration:** `.golangci.yml`

**Success Criteria:**
- No linting issues found
- Code follows Go conventions

## Status Check Job

**Purpose:** Aggregate results and determine overall CI status

**Logic:**
- ✅ **Required for success:** Test, Build, Lint
- ⚠️ **Optional:** Integration Tests (failures don't block CI)

**Why Integration Tests are Optional:**
- Requires external pumpX2 repository
- May fail due to network issues
- pumpX2 dev branch changes may break tests
- Unit tests provide sufficient confidence

## Triggering CI

### Automatic Triggers

**Push Events:**
```yaml
branches:
  - main
  - claude/**
```

**Pull Request Events:**
```yaml
branches:
  - main
```

### Manual Triggers

You can manually trigger workflows from the GitHub Actions UI:
1. Go to Actions tab
2. Select "CI" workflow
3. Click "Run workflow"
4. Select branch

## Local CI Verification

### Run All Checks Locally

```bash
# 1. Run unit tests
go test -v -race ./...

# 2. Run vet
go vet ./...

# 3. Run linter (requires golangci-lint)
golangci-lint run

# 4. Build
go build -v .

# 5. Run integration tests (requires pumpX2)
export PUMPX2_PATH=/path/to/pumpX2
go test -v ./pkg/handler -run Integration
```

### Install golangci-lint

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or using Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Caching Strategy

### Go Module Cache
```yaml
path: |
  ~/.cache/go-build
  ~/go/pkg/mod
key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}
```

**Benefits:**
- Faster dependency downloads
- Reduced network usage
- Faster builds (cached Go build artifacts)

### Gradle Cache
```yaml
path: |
  ~/.gradle/caches
  ~/.gradle/wrapper
key: ${{ runner.os }}-gradle-${{ hashFiles('**/*.gradle*', '**/gradle-wrapper.properties') }}
```

**Benefits:**
- Faster pumpX2 builds
- Reduced Gradle plugin downloads

## Coverage Reporting

**Tool:** Codecov

**Configuration:**
- Uploaded after unit tests complete
- Includes race detection data
- Atomic coverage mode for accuracy

**Badge:**
Add to README.md:
```markdown
[![codecov](https://codecov.io/gh/jwoglom/faketandem/branch/main/graph/badge.svg)](https://codecov.io/gh/jwoglom/faketandem)
```

**Viewing Reports:**
1. Go to https://codecov.io/gh/jwoglom/faketandem
2. View coverage by file, package, or commit
3. Track coverage trends over time

## Troubleshooting

### Test Failures

**Check logs:**
1. Go to Actions tab
2. Click on failed run
3. Expand "Run unit tests" step
4. Review failure output

**Common issues:**
- Race conditions: Review concurrent code
- Flaky tests: Add retries or fix timing issues
- Missing mocks: Update test fixtures

### Integration Test Failures

**Check pumpX2 build:**
1. Expand "Build pumpX2 cliparser" step
2. Verify Gradle build succeeded
3. Check "Verify cliparser JAR" step

**Common issues:**
- pumpX2 dev branch changes: Update faketandem code
- Gradle network issues: Retry workflow
- Java version mismatch: Update setup-java action

### Lint Failures

**Fix locally:**
```bash
# Auto-fix formatting
gofmt -w .

# Auto-fix imports
goimports -w .

# Run linter
golangci-lint run --fix
```

**Common issues:**
- Unused imports: Remove them
- Ineffective assignments: Fix the code
- Misspellings: Fix typos
- Complex functions: Refactor to reduce complexity

### Build Failures

**Check Go version:**
- Ensure code is compatible with Go 1.21+
- Check for deprecated API usage

**Check dependencies:**
```bash
go mod verify
go mod tidy
```

### Cache Issues

**Clear caches:**
1. Go to repository Settings
2. Click Actions → Caches
3. Delete specific caches
4. Re-run workflow

## Performance Optimization

### Current Timings (Approximate)

- Test Job: ~2-3 minutes
- Integration Tests: ~5-7 minutes (includes pumpX2 build)
- Build Job: ~1-2 minutes
- Lint Job: ~2-3 minutes

**Total:** ~10-15 minutes

### Optimization Opportunities

1. **Parallel Execution:** Jobs run in parallel
2. **Caching:** Reduces download and build times
3. **Matrix Strategy:** Tests multiple Go versions efficiently
4. **Artifact Retention:** 7 days (saves storage)

## Best Practices

### Before Pushing

1. Run tests locally: `go test ./...`
2. Run linter: `golangci-lint run`
3. Ensure code builds: `go build`
4. Check formatting: `gofmt -l .`

### Writing Tests

1. Keep tests fast (unit tests < 1s each)
2. Use t.Parallel() for concurrent tests
3. Mock external dependencies
4. Skip integration tests appropriately

### Pull Requests

1. Ensure CI passes before requesting review
2. Address lint warnings
3. Maintain or improve coverage
4. Update tests for new features

## GitHub Status Checks

**Required Checks:**
- Test (all Go versions)
- Build
- Lint

**Optional Checks:**
- Integration Tests

**Branch Protection:**
Configure in repository settings to require:
- CI status check passing
- Code review approval
- Up-to-date with base branch

## Secrets and Variables

**No secrets required** for current CI configuration.

If needed in future:
1. Go to Settings → Secrets and variables → Actions
2. Add repository secrets
3. Reference in workflow: `${{ secrets.SECRET_NAME }}`

## Extending CI

### Adding New Jobs

```yaml
new-job:
  name: New Job
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - name: Do something
      run: echo "Hello"
```

### Adding New Linters

Edit `.golangci.yml`:
```yaml
linters:
  enable:
    - newlinter
```

### Adding Deployment

```yaml
deploy:
  name: Deploy
  runs-on: ubuntu-latest
  needs: [test, build, lint]
  if: github.ref == 'refs/heads/main'
  steps:
    - name: Deploy to production
      run: ./deploy.sh
```

## Support

For issues with CI:
1. Check this documentation
2. Review workflow logs
3. Open an issue with CI logs attached
4. Tag with `ci` label
