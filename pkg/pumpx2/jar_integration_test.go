package pumpx2

import (
	"os"
	"testing"
)

// TestJarRunner_Parse_RealJpake1aRequest exercises the real cliparser jar (not
// just our own output parser) against a captured real-device JPAKE message, to
// guard against regressions in the input framing (raw, unstripped, whitespace-
// joined fragments) that the CLI's "parse" command actually requires. Skipped
// unless FAKETANDEM_TEST_CLIPARSER_JAR points at a real cliparser jar, since CI
// doesn't have one available.
func TestJarRunner_Parse_RealJpake1aRequest(t *testing.T) {
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping real jar integration test")
	}

	runner := NewJarRunner(jarPath, "java")
	output, err := runner.Parse("authentication", realJpake1aRawFragments)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	name, cargo := parseCliparserOutput(output)
	if name != "Jpake1aRequest" {
		t.Fatalf("expected message name Jpake1aRequest, got %q (output: %s)", name, output)
	}
	if cargo["appInstanceId"] != 0 {
		t.Errorf("expected appInstanceId=0, got %v", cargo["appInstanceId"])
	}
	if _, ok := cargo["centralChallenge"].(string); !ok {
		t.Errorf("expected centralChallenge to be a hex string, got %v (%T)", cargo["centralChallenge"], cargo["centralChallenge"])
	}

	opcode, txID, ok := opcodeAndTxIDFromFirstFragment(realJpake1aRawFragments)
	if !ok || opcode != 32 || txID != 4 {
		t.Errorf("expected opcode=32 txID=4, got opcode=%d txID=%d ok=%v", opcode, txID, ok)
	}
}
