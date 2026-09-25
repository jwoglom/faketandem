package handler

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// The tests in this file cover what happens when pumpX2's jpake-server
// abandons a handshake part-way through. They use a stand-in shell script
// instead of the real jar, both so they run everywhere (no java, no jar) and
// so the failure is deterministic rather than a 1-in-256 dice roll -- see
// diagnoseJPAKEServerError for what the real one is.

// jpakeServerErrorEnvelope is the exact line pumpX2's jpake-server prints, and
// then exits on, when its round-2 message comes out a byte short: the
// underlying IllegalArgumentException reaches it wrapped in an
// InvocationTargetException, whose getMessage() is null.
const jpakeServerErrorEnvelope = `{"error":"Exception during server JPAKE authentication: null"}`

// jpake1aEnvelope / jpake1bEnvelope are real jpake-server output envelopes,
// enough for convertServerResponseToParams to do its job.
const jpake1aEnvelope = `{"messageName":"Jpake1aResponse","txId":"0","messageParams":[0,[65,4,-73,-44,-10,-109,-70,-31,65,118,56,51,-121]],"characteristicName":"AUTHORIZATION","packets":["09002100a700004104b7d4f693bae14176383387"],"characteristic":"7b83fff9-9f77-4e5c-8064-aae2c24838b9"}`

const jpake1bEnvelope = `{"messageName":"Jpake1bResponse","txId":"1","messageParams":[0,[65,4,11,22,33,44,55,66,77,88,99,110,121]],"characteristicName":"AUTHORIZATION","packets":["09012201a700004104b7d4f693bae14176383387"],"characteristic":"7b83fff9-9f77-4e5c-8064-aae2c24838b9"}`

// writeFakeJPAKEServer writes an executable script that plays jpake-server,
// and returns its path. body is the script's shell source; it is invoked with
// whatever arguments the authenticator would have passed to java, and ignores
// them.
func writeFakeJPAKEServer(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fake-jpake-server.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil { //nolint:gosec // test-only helper script, must be executable
		t.Fatalf("failed to write fake jpake-server: %v", err)
	}
	return path
}

// TestJPAKEServerProcessFailsFastOnErrorEnvelope: an {"error": ...} line is
// jpake-server's last word, so expect must fail on it immediately rather than
// discarding it as unrecognized chatter and then sitting out its full timeout.
// That difference is what turns a pairing attempt that hangs until the
// driver's own 30-90 second timeout into one that fails at once.
func TestJPAKEServerProcessFailsFastOnErrorEnvelope(t *testing.T) {
	// Prints the error envelope and then hangs around, exactly like a JVM that
	// has not quite exited yet: nothing about the failure may depend on the
	// process dying, or on stdout closing.
	script := writeFakeJPAKEServer(t, "echo '"+jpakeServerErrorEnvelope+"'\nsleep 60\n")

	server, err := startJPAKEServer(script, nil)
	if err != nil {
		t.Fatalf("failed to start fake jpake-server: %v", err)
	}
	defer server.close()

	start := time.Now()
	// Deliberately a longer timeout than the handshake's own: the point is
	// that the error is noticed without waiting for any timeout at all.
	_, err = server.expect(regexp.MustCompile(`JPAKE_2:\s*(\{.*\})`), 2*jpakeServerTimeout)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expect succeeded against a jpake-server that reported an error")
	}
	if !errors.Is(err, ErrJPAKEServerFailed) {
		t.Errorf("expected ErrJPAKEServerFailed, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("expect took %s to notice the error envelope; it must not wait out the timeout", elapsed)
	}
	if !strings.Contains(err.Error(), "Exception during server JPAKE authentication: null") {
		t.Errorf("error should quote what jpake-server said, got: %v", err)
	}
	if !strings.Contains(err.Error(), "1 handshake in 256") {
		t.Errorf("error should name the known upstream cause, got: %v", err)
	}
}

// TestJPAKEServerProcessErrorIncludesStderr: jpake-server prints the Java
// stack trace behind an aborted handshake to stderr and nowhere else, so an
// error that omits it leaves nothing to diagnose from.
func TestJPAKEServerProcessErrorIncludesStderr(t *testing.T) {
	script := writeFakeJPAKEServer(t, "echo 'java.lang.reflect.InvocationTargetException' >&2\nsleep 0.2\nexit 3\n")

	server, err := startJPAKEServer(script, nil)
	if err != nil {
		t.Fatalf("failed to start fake jpake-server: %v", err)
	}
	defer server.close()

	_, err = server.expect(regexp.MustCompile(`JPAKE_1A:\s*(\{.*\})`), jpakeServerTimeout)
	if err == nil {
		t.Fatal("expect succeeded against a jpake-server that printed nothing and exited")
	}
	if !errors.Is(err, ErrJPAKEServerFailed) {
		t.Errorf("expected ErrJPAKEServerFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "InvocationTargetException") {
		t.Errorf("error should carry the jpake-server stderr tail, got: %v", err)
	}
}

// TestDiagnoseJPAKEServerError checks that the "null" exception is named only
// where it is actually diagnosed -- while waiting for round 2 -- and not
// pinned on every unrelated failure.
func TestDiagnoseJPAKEServerError(t *testing.T) {
	round2 := jpakeEnvelopeRegex("JPAKE_2")
	round4 := jpakeEnvelopeRegex("JPAKE_4")

	if got := diagnoseJPAKEServerError("Exception during server JPAKE authentication: null", round2); got == "" {
		t.Error("expected a diagnosis for the round-2 null exception")
	}
	if got := diagnoseJPAKEServerError("Exception during server JPAKE authentication: null", round4); got != "" {
		t.Errorf("did not expect a round-2 diagnosis while waiting for round 4, got: %s", got)
	}
	if got := diagnoseJPAKEServerError("Expected Jpake1bRequest, got: null", round2); got != "" {
		t.Errorf("did not expect a diagnosis for an unrelated error, got: %s", got)
	}
}

// TestPumpX2JPAKEAuthenticatorAbortsHandshakeOnServerFailure drives the real
// authenticator through the exact sequence the flake produces: rounds 1a and
// 1b answered normally, then the error envelope instead of JPAKE_2. The
// handshake must fail immediately, the subprocess must be gone, and every
// later round must keep failing instead of quietly starting a second server
// half-way through a handshake the client has already given up on.
func TestPumpX2JPAKEAuthenticatorAbortsHandshakeOnServerFailure(t *testing.T) {
	script := writeFakeJPAKEServer(t, strings.Join([]string{
		"echo 'JPAKE_1A: " + jpake1aEnvelope + "'",
		"read _client1a",
		"echo 'JPAKE_1B: " + jpake1bEnvelope + "'",
		"read _client1b",
		"echo 'java.lang.reflect.InvocationTargetException' >&2",
		"echo '" + jpakeServerErrorEnvelope + "'",
		"",
	}, "\n"))

	auth := NewPumpX2JPAKEAuthenticator("123456", nil, "", "jar", "", script, "unused.jar")
	defer func() { _ = auth.Close() }()

	requestData := func(name string) map[string]interface{} {
		return map[string]interface{}{
			"messageName":   name,
			"rawPacketsHex": []string{"09002000a700004104db40769d34d99ed523e489"},
		}
	}

	if _, err := auth.ProcessRound(1, requestData("Jpake1aRequest")); err != nil {
		t.Fatalf("round 1a failed against the fake server: %v", err)
	}

	pid := auth.server.cmd.Process.Pid

	start := time.Now()
	_, err := auth.ProcessRound(1, requestData("Jpake1bRequest"))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("round 1b succeeded even though jpake-server reported an error instead of JPAKE_2")
	}
	if !errors.Is(err, ErrJPAKEServerFailed) {
		t.Errorf("expected ErrJPAKEServerFailed, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("round 1b took %s to fail; it must not wait out the 30s read timeout", elapsed)
	}

	if auth.server != nil {
		t.Error("the failed jpake-server subprocess was not released")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := syscall.Kill(pid, 0); err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("fake jpake-server pid %d still running 5s after the handshake failed", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// A later round must not resurrect the handshake by starting a second
	// server: the client is talking to values only the dead one knew.
	if _, err := auth.ProcessRound(2, requestData("Jpake2Request")); !errors.Is(err, ErrJPAKEServerFailed) {
		t.Errorf("expected the authenticator to stay failed, got %v", err)
	}
	if auth.server != nil {
		t.Error("a later round started a replacement jpake-server for an abandoned handshake")
	}
}

// failingAuthenticator is a JPAKEAuthenticatorInterface whose rounds always
// fail the way a dead jpake-server's do.
type failingAuthenticator struct{}

func (failingAuthenticator) ProcessRound(int, map[string]interface{}) (map[string]interface{}, error) {
	return nil, ErrJPAKEServerFailed
}
func (failingAuthenticator) GetSharedSecret() ([]byte, error) { return nil, errors.New("not complete") }
func (failingAuthenticator) GetLongTermSecret() ([]byte, error) {
	return nil, errors.New("not complete")
}
func (failingAuthenticator) IsComplete() bool { return false }

// TestJPAKEHandlerDropsSessionOnRoundFailure: session IDs are a constant, so a
// failed authenticator left in the session map would be handed straight to the
// client's next pairing attempt -- and a failed one refuses every round
// forever. One unlucky handshake must not make the emulator unpairable.
func TestJPAKEHandlerDropsSessionOnRoundFailure(t *testing.T) {
	manager := NewJPAKESessionManager("pumpx2", "/tmp/pumpx2", "jar", "", "java", "unused.jar", state.NewPumpState())
	manager.authenticators["default"] = failingAuthenticator{}

	h := NewJPAKEHandler(nil, manager, "Jpake1bRequest", 1)
	_, err := h.HandleMessage(&pumpx2.ParsedMessage{
		MessageType:   "Jpake1bRequest",
		TxID:          1,
		Cargo:         map[string]interface{}{},
		RawPacketsHex: []string{"09012201a700004104181c34a4a79473cdb40ef1"},
	}, state.NewPumpState())

	if !errors.Is(err, ErrJPAKEServerFailed) {
		t.Fatalf("expected the round failure to surface as ErrJPAKEServerFailed, got %v", err)
	}
	if _, stillThere := manager.authenticators["default"]; stillThere {
		t.Error("the failed authenticator was left in the session map; the next pairing attempt would inherit it")
	}
}
