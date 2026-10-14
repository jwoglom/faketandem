# Claude-Specific Guidelines

This document contains specific instructions for Claude (Anthropic's AI assistant) when working on the faketandem codebase.

## Before Making Any Code Changes

**Always run the linter first to check current state:**

```bash
golangci-lint run --timeout=5m
```

If there are existing issues, fix them before making your changes.

## After Making Code Changes

**Always auto-fix and verify:**

```bash
# 1. Auto-fix formatting and simple issues
golangci-lint run --fix --timeout=5m

# 2. Stage the fixed files
git add -u

# 3. Check for remaining issues
golangci-lint run --timeout=5m
```

## Linting Command Reference

### Basic Commands

```bash
# Check all code
golangci-lint run --timeout=5m

# Auto-fix issues
golangci-lint run --fix --timeout=5m

# Check specific directory
golangci-lint run --timeout=5m ./pkg/handler/

# Check specific file
golangci-lint run --timeout=5m ./pkg/handler/router.go
```

### When to Use --fix

Use `--fix` for:
- ✅ Formatting issues (gofmt, goimports)
- ✅ Import organization
- ✅ Unnecessary type conversions
- ✅ Simple naming convention fixes

**Do NOT rely on --fix for:**
- ❌ Unchecked errors (requires manual error handling)
- ❌ High complexity (requires refactoring)
- ❌ Logic errors

## Common Linting Issues and How to Fix

### 1. Unchecked Errors (errcheck)

**Error Message:**
```
Error return value of `someFunc` is not checked (errcheck)
```

**Fix Pattern:**
```go
// Before
someFunc()

// After
if err := someFunc(); err != nil {
    log.Warnf("Failed to call someFunc: %v", err)
}
```

**For functions where errors can be safely ignored:**
```go
// Add a comment explaining why
_ = someFunc() // Safe to ignore: this is a cleanup operation
```

### 2. High Cyclomatic Complexity (gocyclo)

**Error Message:**
```
cyclomatic complexity 17 of func `funcName` is high (> 15) (gocyclo)
```

**Fix Pattern - Extract Helper Functions:**
```go
// Before
func complexFunc() {
    if condition1 {
        // 10 lines
    }
    if condition2 {
        // 10 lines
    }
    if condition3 {
        // 10 lines
    }
}

// After
func complexFunc() {
    handleCondition1()
    handleCondition2()
    handleCondition3()
}

func handleCondition1() {
    if condition1 {
        // 10 lines
    }
}

func handleCondition2() {
    if condition2 {
        // 10 lines
    }
}

func handleCondition3() {
    if condition3 {
        // 10 lines
    }
}
```

### 3. Naming Conventions (revive)

**Error Message:**
```
var-naming: type ApiVersionHandler should be APIVersionHandler (revive)
```

**Fix Pattern:**
```go
// Before
type ApiVersionHandler struct {}
func NewApiVersionHandler() *ApiVersionHandler {}
type HttpServer struct {}

// After
type APIVersionHandler struct {}
func NewAPIVersionHandler() *APIVersionHandler {}
type HTTPServer struct {}
```

**Common Acronyms:**
- API (not Api)
- HTTP (not Http)
- URL (not Url)
- ID (not Id)
- JSON (not Json)
- XML (not Xml)

**Avoid Stuttering:**
```go
// Before
type HandlerResponse struct {}  // in package handler

// After
type Response struct {}  // in package handler
```

### 4. Formatting Issues (gofmt, goimports)

**Error Message:**
```
File is not properly formatted (gofmt)
```

**Fix:**
```bash
# Let golangci-lint fix it
golangci-lint run --fix

# Or use gofmt/goimports directly
gofmt -w pkg/handler/myfile.go
goimports -w pkg/handler/myfile.go
```

## Workflow for Claude

When making changes to the codebase, follow this workflow:

### 1. Initial Check
```bash
golangci-lint run --timeout=5m
```

Note any existing issues.

### 2. Make Your Changes

Edit the necessary files.

### 3. Auto-Fix
```bash
golangci-lint run --fix --timeout=5m
```

This will automatically fix:
- Formatting
- Import organization
- Simple naming issues

### 4. Check Results
```bash
golangci-lint run --timeout=5m
```

If there are remaining issues, they need manual fixes.

### 5. Manual Fixes

For remaining issues (typically errcheck and gocyclo):
- Add error checking
- Refactor complex functions
- Fix any logic issues

### 6. Final Verification
```bash
golangci-lint run --timeout=5m
```

Should show no issues.

### 7. Commit
```bash
git add -A
git commit -m "Your descriptive commit message"
```

### 8. Push

The pre-push hook will automatically:
- Run `golangci-lint run --fix`
- Stage any fixed files
- Verify no issues remain
- Block push if issues exist

## Tool Calls for Claude

When using tools in the Cursor/Claude environment:

### Check Linting
```xml
<Shell command="cd /Users/james/repos/faketandem && golangci-lint run --timeout=5m" />
```

### Auto-Fix
```xml
<Shell command="cd /Users/james/repos/faketandem && golangci-lint run --fix --timeout=5m" />
```

### Check Specific Package
```xml
<Shell command="cd /Users/james/repos/faketandem && golangci-lint run --timeout=5m ./pkg/handler/" />
```

## Testing

After fixing linting issues, always run tests:

```bash
# Quick test
go test -v ./...

# Full test with race detection
go test -v -race ./...

# Specific package
go test -v ./pkg/handler/
```

## Common Pitfalls

### 1. Don't Ignore Real Errors

```go
// Bad - silently ignoring important errors
_ = db.Close()

// Good - at least log it
if err := db.Close(); err != nil {
    log.Warnf("Failed to close database: %v", err)
}
```

### 2. Don't Over-Extract

```go
// Bad - too granular
func processData() {
    step1()
    step2()
    step3()
}
func step1() { x := 1 }
func step2() { y := 2 }
func step3() { z := 3 }

// Good - meaningful extraction
func processData() {
    data := loadData()
    result := transformData(data)
    saveResult(result)
}
```

### 3. Don't Break Existing Tests

After refactoring:
```bash
go test -v ./...
```

Must still pass.

## Configuration Files

The linting configuration is in `.golangci.yml`:
- Version: 2
- Timeout: 5m
- Min complexity: 15
- Enabled linters: govet, errcheck, staticcheck, unused, ineffassign, gocyclo, misspell, revive, unconvert, unparam

## CI Pipeline

GitHub Actions runs:
```bash
golangci-lint run --timeout=5m
```

Using golangci-lint v2.7.2 (action v6).

Must pass before merge.

## Quick Command Summary

```bash
# Full workflow
golangci-lint run --fix && golangci-lint run && go test -v ./...

# Just lint
golangci-lint run

# Just fix
golangci-lint run --fix

# Just test
go test -v ./...

# Full CI simulation
golangci-lint run && go vet ./... && go test -race ./...
```

## When in Doubt

1. Run `golangci-lint run --fix` first
2. Check what remains with `golangci-lint run`
3. Fix manually if needed
4. Test with `go test -v ./...`
5. Commit and push

The pre-push hook will catch any missed issues!

## Cursor Cloud specific instructions

### Project overview
faketandem is a Tandem insulin pump BLE emulator written in Go. It has a single binary entry point at the repo root (`main.go`), an HTTP/WebSocket API on `:8080`, and a static web UI in `ui/`.

### pumpX2 is optional
Messages are parsed and encoded by `pkg/pumpx2/messages`, a Go port of the [pumpX2](https://github.com/jwoglom/pumpX2) message library as its cliparser jar drives it, and pairing runs `pkg/jpake`, a Go port of pumpX2's EC-JPAKE. Neither the app nor its tests need pumpX2 or a JVM. pumpX2 is used only by `-message-codec pumpx2`/`compare` and `-jpake-mode pumpx2`, and by the tests that check the port against it (`FAKETANDEM_TEST_CLIPARSER_JAR`, below). The update script may still clone pumpX2 to `/workspace/pumpX2` (branch `dev`) for those.

The port must stay byte-for-byte equal to the jar. Don't edit `catalog_gen.go` by hand: regenerate it with `tools/pumpx2catalog/regen.sh <jar>`, port any class `TestEveryClassIsPorted` names in the matching `msg_*.go` file, and re-record the jar's answers with `go test ./pkg/pumpx2/messages -run MatchesJar -update-golden` (with `FAKETANDEM_TEST_CLIPARSER_JAR` set). A port failure is fixed in the port, never by editing the recorded answers.

### BLE hardware limitation
The binary calls `log.Fatalf` when BLE adapter initialization fails (`bluetooth_linux.go`). In the Cloud Agent VM there is no Bluetooth adapter, so the app exits after initializing all non-hardware components (config, message codec, pump state, protocol). Use `-transport virtual` to run it anyway. This is expected — the app is designed for Raspberry Pi 3+. All unit tests and linting still work fully without BLE hardware.

### Key commands
| Task | Command |
|---|---|
| Build | `go build -o faketandem .` |
| Lint | `golangci-lint run --timeout=5m` |
| Lint fix | `golangci-lint run --fix --timeout=5m` |
| All tests (unit + integration) | `go test -v -race -timeout 5m ./...` |
| All tests, also checked against pumpX2 | `FAKETANDEM_TEST_CLIPARSER_JAR=<cliparser.jar> go test -timeout 20m ./...` |
| Run app | `./faketandem -transport virtual -q` |

### Go dependencies
All Go dependencies are vendored in `vendor/`. No `go mod download` is needed.

### Tests against pumpX2: FAKETANDEM_TEST_CLIPARSER_JAR
With `FAKETANDEM_TEST_CLIPARSER_JAR` set to a built cliparser jar, the parity cases in `pkg/pumpx2/messages` and `pkg/pumpx2` run live against it (without it they run against the recorded answers), the `*_jar_test.go` handler and harness tests run through it as well as the Go port, and the tests of pumpX2's own JPAKE run. `PUMPX2_PATH` is only needed for the gradle-mode `TestPumpX2JPAKEAuthenticator_FullFlow`.

### Pre-push hook
`scripts/pre-push.sh` runs `golangci-lint --fix` then verifies no issues remain. It exits gracefully if `golangci-lint` is not installed. When pushing with `--no-verify`, the hook is skipped.
