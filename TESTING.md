# Testing Guide for faketandem

This document describes how to run and write tests for the faketandem pump simulator.

## Test Organization

Tests are organized into two categories:

### Unit Tests (`*_test.go`)
- Test individual components in isolation
- No external dependencies required
- Fast execution
- Located alongside source files in `pkg/` directories

### Integration Tests (`*_integration_test.go`)
- Test complete workflows with real dependencies
- May require pumpX2 installation
- Slower execution
- Test actual JPAKE authentication flows

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Unit Tests Only

```bash
go test -short ./...
```

### Run Integration Tests Only

Integration tests require the `PUMPX2_PATH` environment variable:

```bash
export PUMPX2_PATH=/path/to/pumpX2
go test -run Integration ./...
```

### Run Specific Test Package

```bash
# Test JPAKE handlers
go test ./pkg/handler

# With verbose output
go test -v ./pkg/handler

# Run specific test
go test -v ./pkg/handler -run TestJPAKEAuthenticator_FullFlow
```

### Run with Coverage

```bash
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Run Benchmarks

```bash
go test -bench=. ./pkg/handler
```

## JPAKE Tests

### Unit Tests (`pkg/handler/jpake_test.go`)

Tests individual JPAKE components without external dependencies:

- **Interface compliance**: Verifies both implementations satisfy `JPAKEAuthenticatorInterface`
- **Initialization**: Tests proper setup of authenticator instances
- **Session management**: Tests `JPAKESessionManager` operations
- **State management**: Tests round progression and completion tracking
- **Error conditions**: Tests invalid rounds, incomplete authentication, etc.

Run unit tests:
```bash
go test -v ./pkg/handler -run "^Test.*Authenticator"
```

### Integration Tests (`pkg/handler/jpake_integration_test.go`)

Tests complete JPAKE authentication flows:

#### PumpX2 JPAKE Integration Test

**Prerequisites:**
1. pumpX2 repository cloned and available
2. jpake-server changes applied to Main.java
3. pumpX2 built with `./gradlew cliparser`

**Setup:**
```bash
# Set pumpX2 path
export PUMPX2_PATH=/path/to/pumpX2

# Verify pumpX2 is built
cd $PUMPX2_PATH
./gradlew cliparser
```

**Run test:**
```bash
go test -v ./pkg/handler -run TestPumpX2JPAKEAuthenticator_FullFlow
```

**What it tests:**
- Spawns actual `jpake-server` process
- Sends mock client requests through all 4 JPAKE rounds
- Verifies server responses contain required fields
- Validates shared secret derivation
- Tests complete authentication flow

#### Go JPAKE Test

Tests the simplified Go implementation:

```bash
go test -v ./pkg/handler -run TestGoJPAKEAuthenticator_FullFlow
```

**What it tests:**
- Go-based JPAKE implementation
- All 4 rounds of authentication
- Shared secret generation
- No external dependencies required

### Session Manager Tests

Tests concurrent session handling:

```bash
go test -v ./pkg/handler -run TestJPAKESessionManager
```

## Test Environment Variables

### Required for Integration Tests

- `PUMPX2_PATH`: Path to pumpX2 repository
  ```bash
  export PUMPX2_PATH=/home/user/pumpX2
  ```

### Optional

- `LOG_LEVEL`: Set logging verbosity (debug, info, warn, error)
  ```bash
  export LOG_LEVEL=debug
  go test -v ./pkg/handler -run Integration
  ```

## Writing New Tests

### Unit Test Template

```go
package handler

import "testing"

func TestMyComponent(t *testing.T) {
    // Setup
    component := NewMyComponent()

    // Execute
    result, err := component.DoSomething()

    // Verify
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if result != expectedValue {
        t.Errorf("Expected %v, got %v", expectedValue, result)
    }
}
```

### Integration Test Template

```go
func TestMyFeature_Integration(t *testing.T) {
    // Skip if required environment not available
    requiredPath := os.Getenv("REQUIRED_PATH")
    if requiredPath == "" {
        t.Skip("Skipping: REQUIRED_PATH not set")
    }

    // Setup
    // ... create components with real dependencies

    // Execute full workflow
    // ...

    // Verify end-to-end behavior
    // ...
}
```

## Common Test Scenarios

### Testing JPAKE Round Processing

```go
func TestJPAKERound(t *testing.T) {
    auth := NewJPAKEAuthenticator("123456", bridge)

    // Mock client data
    clientData := map[string]interface{}{
        "messageName": "Jpake1aRequest",
        "centralChallengeHash": "abc123...",
    }

    // Process round
    response, err := auth.ProcessRound(1, clientData)
    if err != nil {
        t.Fatalf("Round failed: %v", err)
    }

    // Verify response structure
    if _, ok := response["centralChallengeHash"]; !ok {
        t.Error("Missing required field")
    }
}
```

### Testing Error Conditions

```go
func TestErrorCondition(t *testing.T) {
    auth := NewJPAKEAuthenticator("123456", bridge)

    // Test invalid input
    _, err := auth.ProcessRound(99, nil)
    if err == nil {
        t.Error("Expected error for invalid round")
    }
}
```

### Testing Concurrent Sessions

```go
func TestConcurrentSessions(t *testing.T) {
    manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java")

    // Create multiple sessions concurrently
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            sessionID := fmt.Sprintf("session-%d", id)
            auth := manager.GetOrCreate(sessionID, "123456", bridge)
            if auth == nil {
                t.Error("Failed to create session")
            }
        }(i)
    }
    wg.Wait()
}
```

## Continuous Integration

For CI environments, use:

```bash
#!/bin/bash
set -e

# Run unit tests (no external dependencies)
echo "Running unit tests..."
go test -short -v ./...

# Run integration tests if pumpX2 available
if [ -d "$PUMPX2_PATH" ]; then
    echo "Running integration tests..."
    go test -v ./... -run Integration
else
    echo "Skipping integration tests (PUMPX2_PATH not set)"
fi

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Debugging Tests

### Enable Debug Logging

```go
import log "github.com/sirupsen/logrus"

func TestDebug(t *testing.T) {
    log.SetLevel(log.DebugLevel)
    // Your test code
}
```

### Run Single Test with Verbose Output

```bash
go test -v ./pkg/handler -run TestSpecificTest
```

### Print Test Data

```go
t.Logf("Debug info: %+v", someData)
```

## Performance Testing

### Run Benchmarks

```bash
# All benchmarks
go test -bench=. ./...

# Specific benchmark
go test -bench=BenchmarkJPAKE ./pkg/handler

# With memory profiling
go test -bench=. -benchmem ./pkg/handler

# Save results
go test -bench=. ./... > bench.txt
```

### Compare Benchmark Results

```bash
# Install benchcmp
go install golang.org/x/tools/cmd/benchcmp@latest

# Run before changes
go test -bench=. ./... > old.txt

# Make changes
# ...

# Run after changes
go test -bench=. ./... > new.txt

# Compare
benchcmp old.txt new.txt
```

## Troubleshooting

### Integration Test Fails: "pumpX2 directory does not exist"

Ensure `PUMPX2_PATH` is set correctly:
```bash
export PUMPX2_PATH=/path/to/pumpX2
echo $PUMPX2_PATH  # Verify it's set
ls -la $PUMPX2_PATH  # Verify directory exists
```

### Integration Test Fails: "failed to spawn JPAKE process"

Ensure pumpX2 is built:
```bash
cd $PUMPX2_PATH
./gradlew cliparser
ls -la cliparser/build/libs/cliparser.jar  # Verify JAR exists
```

### Test Timeout

Increase timeout for slow tests:
```bash
go test -timeout 30s ./pkg/handler -run Integration
```

### Import Cycle Errors

- Move test helpers to `_test.go` files
- Use separate test packages when needed: `package handler_test`

## Best Practices

1. **Test naming**: Use descriptive names that explain what is being tested
   - Good: `TestJPAKEAuthenticator_InvalidRound`
   - Bad: `TestJPAKE1`

2. **Table-driven tests**: Use for testing multiple scenarios
   ```go
   tests := []struct {
       name string
       input int
       want string
   }{
       {"case 1", 1, "one"},
       {"case 2", 2, "two"},
   }
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           // test code
       })
   }
   ```

3. **Test helpers**: Extract common setup into helper functions

4. **Cleanup**: Use `defer` for cleanup
   ```go
   auth := NewJPAKEAuthenticator(...)
   defer auth.Close()
   ```

5. **Skip appropriately**: Skip tests that require unavailable resources
   ```go
   if os.Getenv("REQUIRED_VAR") == "" {
       t.Skip("Skipping: REQUIRED_VAR not set")
   }
   ```
