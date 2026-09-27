package pumpx2

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// The test binary doubles as the "java" these runners exec: with FAKE_CLIPARSER
// set it acts as a cliparser jar instead of running tests. Its arguments are
// "-jar <path> <command...>".
func TestMain(m *testing.M) {
	if mode := os.Getenv("FAKE_CLIPARSER"); mode != "" {
		os.Exit(fakeCliparser(mode, os.Args[3:]))
	}
	os.Exit(m.Run())
}

func fakeCliparser(mode string, args []string) int {
	if args[0] != "serve" {
		// One-shot, as JarRunner calls it.
		fmt.Printf("one-shot %s\n", strings.Join(args, " "))
		return 0
	}
	if mode == "no-serve" {
		fmt.Fprintln(os.Stderr, "Nothing to do.")
		return 0
	}
	if startsFile := os.Getenv("FAKE_CLIPARSER_STARTS"); startsFile != "" {
		f, err := os.OpenFile(startsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.WriteString("start\n")
			_ = f.Close()
		}
	}
	scanner := bufio.NewScanner(os.Stdin)
	served := 0
	for scanner.Scan() {
		if mode == "crash-after-one" && served == 1 {
			return 1
		}
		var request serveRequest
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			return 2
		}
		stdout := fmt.Sprintf("served %s characteristic=%s\n", strings.Join(request.Args, " "), request.Env["PUMPX2_CHARACTERISTIC"])
		reply, _ := json.Marshal(serveReply{ID: request.ID, OK: true, Stdout: stdout})
		fmt.Println(string(reply))
		served++
	}
	return 0
}

func fakeServeRunner(t *testing.T, mode string) *ServeRunner {
	t.Helper()
	t.Setenv("FAKE_CLIPARSER", mode)
	runner := NewServeRunner("fake.jar", os.Args[0])
	t.Cleanup(func() { _ = runner.Close() })
	return runner
}

func TestServeRunnerAnswersEveryCallFromOneProcess(t *testing.T) {
	starts := t.TempDir() + "/starts"
	t.Setenv("FAKE_CLIPARSER_STARTS", starts)
	runner := fakeServeRunner(t, "serve")

	for i := 0; i < 3; i++ {
		output, err := runner.Parse("CURRENT_STATUS", []string{"0001", "0203"})
		if err != nil {
			t.Fatalf("parse %d: %v", i, err)
		}
		if output != "served parse 0001 0203 characteristic=CURRENT_STATUS\n" {
			t.Fatalf("parse %d output %q", i, output)
		}
	}
	output, err := runner.Encode(7, "ApiVersionRequest", nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if output != "served encode 7 ApiVersionRequest {} characteristic=\n" {
		t.Fatalf("encode output %q", output)
	}

	if got := countLines(t, starts); got != 1 {
		t.Fatalf("started %d processes, want 1", got)
	}
}

func TestServeRunnerRestartsAProcessThatDiesAndRetries(t *testing.T) {
	starts := t.TempDir() + "/starts"
	t.Setenv("FAKE_CLIPARSER_STARTS", starts)
	runner := fakeServeRunner(t, "crash-after-one")

	for i := 0; i < 2; i++ {
		if _, err := runner.Parse("", []string{"00"}); err != nil {
			t.Fatalf("parse %d: %v", i, err)
		}
	}

	if got := countLines(t, starts); got != 2 {
		t.Fatalf("started %d processes, want 2", got)
	}
}

func TestServeRunnerFallsBackToAJVMPerCallWithoutServe(t *testing.T) {
	runner := fakeServeRunner(t, "no-serve")

	for i := 0; i < 2; i++ {
		output, err := runner.Parse("", []string{"00", "11"})
		if err != nil {
			t.Fatalf("parse %d: %v", i, err)
		}
		if output != "one-shot parse 00 11\n" {
			t.Fatalf("parse %d output %q", i, output)
		}
	}
	if !runner.unsupported {
		t.Fatal("serve was not marked unsupported")
	}
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Count(string(data), "\n")
}

// Against a real cliparser jar: through serve if the jar has it, through the
// JarRunner fallback if not. Skipped without FAKETANDEM_TEST_CLIPARSER_JAR.
func TestServeRunner_Parse_RealJpake1aRequest(t *testing.T) {
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping real jar integration test")
	}
	runner := NewServeRunner(jarPath, "java")
	t.Cleanup(func() { _ = runner.Close() })

	for i := 0; i < 2; i++ {
		output, err := runner.Parse("AUTHORIZATION", realJpake1aRawFragments)
		if err != nil {
			t.Fatalf("parse %d: %v", i, err)
		}
		if name, _ := parseCliparserOutput(output); name != "Jpake1aRequest" {
			t.Fatalf("parse %d: message name %q (output: %s)", i, name, output)
		}
	}
	t.Logf("serve unsupported by this jar: %v", runner.unsupported)
}
