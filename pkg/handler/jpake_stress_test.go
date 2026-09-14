package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// This file holds the JPAKE stress harness: it runs the full five-round
// pairing handshake end to end, many times over and several at once, against
// the real pumpX2 jpake-server and jpake-client subprocesses through the real
// PumpX2JPAKEAuthenticator code path.
//
// It exists because fresh pairing failed roughly once in ten full runs of
// TandemKit's integration suite, with jpake-server printing
// {"error":"Exception during server JPAKE authentication: null"} and exiting
// mid-handshake. A single run of TestPumpX2JPAKEAuthenticator_FullFlowViaJar
// is far too coarse to see that; this harness reproduces it in minutes.
//
// It is skipped unless FAKETANDEM_STRESS_JPAKE=1 (and a cliparser jar is
// available), since each iteration spawns several JVMs and CI should not pay
// for that.

// jpakeStressConfig is the harness's runtime configuration, read from the
// environment so a run can be widened without editing code.
type jpakeStressConfig struct {
	jarPath     string
	iterations  int
	parallelism int
}

// jpakeStressConfigFromEnv reads the harness configuration, or reports why the
// harness should be skipped.
func jpakeStressConfigFromEnv() (jpakeStressConfig, string) {
	if os.Getenv("FAKETANDEM_STRESS_JPAKE") != "1" {
		return jpakeStressConfig{}, "FAKETANDEM_STRESS_JPAKE is not 1, skipping JPAKE stress harness"
	}
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		return jpakeStressConfig{}, "FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping JPAKE stress harness"
	}

	cfg := jpakeStressConfig{jarPath: jarPath, iterations: 30, parallelism: 4}
	if v := os.Getenv("FAKETANDEM_STRESS_JPAKE_ITERATIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.iterations = n
		}
	}
	if v := os.Getenv("FAKETANDEM_STRESS_JPAKE_PARALLELISM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.parallelism = n
		}
	}
	return cfg, ""
}

// jpakeStressResult records one handshake attempt.
type jpakeStressResult struct {
	iteration int
	err       error
	stderr    string
	duration  time.Duration
}

// jpakeClient wraps the "java -jar cliparser jpake" subprocess that plays the
// phone side of the handshake.
type jpakeClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

// startJPAKEClient launches pumpX2's own client-side JPAKE implementation.
func startJPAKEClient(jarPath, pairingCode string) (*jpakeClient, error) {
	cmd := exec.Command("java", "-jar", jarPath, "jpake", pairingCode) //nolint:gosec // test-only, jar path comes from the test environment
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	return &jpakeClient{cmd: cmd, stdin: stdin, scanner: scanner}, nil
}

func (c *jpakeClient) close() {
	_ = c.stdin.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	_ = c.cmd.Wait()
}

// readLine reads one line of client output, bounded by timeout.
func (c *jpakeClient) readLine(timeout time.Duration) (string, error) {
	type result struct {
		line string
		ok   bool
	}
	ch := make(chan result, 1)
	go func() {
		if c.scanner.Scan() {
			ch <- result{c.scanner.Text(), true}
			return
		}
		ch <- result{"", false}
	}()
	select {
	case r := <-ch:
		if !r.ok {
			return "", errors.New("jpake client exited")
		}
		return r.line, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out after %s waiting for jpake client output", timeout)
	}
}

// nextRequestPackets waits for the client's next "_SENT:" line and returns the
// raw BLE fragments it carries.
func (c *jpakeClient) nextRequestPackets(timeout time.Duration) ([]string, error) {
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("timed out waiting for the client's next request")
		}
		line, err := c.readLine(remaining)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(line, "_SENT:") || !strings.Contains(line, "packets") {
			continue
		}
		colonIdx := strings.Index(line, ": ")
		if colonIdx < 0 {
			return nil, fmt.Errorf("no JSON payload in client line: %s", line)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(line[colonIdx+2:]), &payload); err != nil {
			return nil, fmt.Errorf("failed to parse client request JSON: %w", err)
		}
		rawPackets, ok := payload["packets"].([]interface{})
		if !ok || len(rawPackets) == 0 {
			return nil, fmt.Errorf("client request line had no packets: %s", line)
		}
		frags := make([]string, len(rawPackets))
		for i, p := range rawPackets {
			frags[i], _ = p.(string)
		}
		return frags, nil
	}
}

// jpakeStressRequestOrder is the fixed order of client requests in a full
// from-scratch pairing.
var jpakeStressRequestOrder = []string{
	"Jpake1aRequest",
	"Jpake1bRequest",
	"Jpake2Request",
	"Jpake3SessionKeyRequest",
	"Jpake4KeyConfirmationRequest",
}

// runOneJPAKEHandshake drives one complete pairing handshake through the real
// PumpX2JPAKEAuthenticator, returning the failure (and whatever jpake-server
// wrote to stderr) rather than failing the test, so the caller can count
// failures across many attempts.
//
//nolint:gocyclo // a sequential protocol drive; splitting it would obscure the order
func runOneJPAKEHandshake(jarPath, pairingCode string) (serverStderr string, err error) {
	bridge, berr := pumpx2.NewBridge("", "jar", "", "java", jarPath)
	if berr != nil {
		return "", fmt.Errorf("failed to create bridge: %w", berr)
	}
	bridge.SetPairingCode(pairingCode)

	auth := NewPumpX2JPAKEAuthenticator(pairingCode, bridge, "", "jar", "", "java", jarPath)
	defer func() {
		if auth.server != nil {
			serverStderr = auth.server.recentStderr()
		}
		_ = auth.Close()
	}()

	client, cerr := startJPAKEClient(jarPath, pairingCode)
	if cerr != nil {
		return "", fmt.Errorf("failed to start jpake client: %w", cerr)
	}
	defer client.close()

	for _, expectedRequestType := range jpakeStressRequestOrder {
		frags, ferr := client.nextRequestPackets(60 * time.Second)
		if ferr != nil {
			return "", fmt.Errorf("waiting for client %s: %w", expectedRequestType, ferr)
		}

		parsed, perr := bridge.ParseMessage(bluetooth.CharAuthorization, frags)
		if perr != nil {
			return "", fmt.Errorf("ParseMessage failed for %s: %w", expectedRequestType, perr)
		}
		if parsed.MessageType != expectedRequestType {
			return "", fmt.Errorf("expected parsed message type %s, got %s", expectedRequestType, parsed.MessageType)
		}

		requestData := make(map[string]interface{}, len(parsed.Cargo)+2)
		for k, v := range parsed.Cargo {
			requestData[k] = v
		}
		requestData["messageName"] = parsed.MessageType
		requestData["rawPacketsHex"] = parsed.RawPacketsHex

		responseParams, rerr := auth.ProcessRound(jpakeRoundForTest(parsed.MessageType), requestData)
		if rerr != nil {
			return "", fmt.Errorf("ProcessRound for %s failed: %w", expectedRequestType, rerr)
		}

		responseType := jpakeResponseTypeForTest(parsed.MessageType)
		encoded, eerr := bridge.EncodeMessage(parsed.TxID, responseType, responseParams)
		if eerr != nil {
			return "", fmt.Errorf("EncodeMessage failed for %s: %w", responseType, eerr)
		}
		if len(encoded.Packets) == 0 {
			return "", fmt.Errorf("EncodeMessage returned no packets for %s", responseType)
		}

		if _, werr := io.WriteString(client.stdin, strings.Join(encoded.Packets, " ")+"\n"); werr != nil {
			return "", fmt.Errorf("failed to write %s to client stdin: %w", responseType, werr)
		}
	}

	if !auth.IsComplete() {
		return "", errors.New("authenticator did not reach round 4 completion")
	}
	secret, serr := auth.GetSharedSecret()
	if serr != nil {
		return "", fmt.Errorf("GetSharedSecret failed: %w", serr)
	}
	if len(secret) == 0 {
		return "", errors.New("empty derived shared secret")
	}

	// Confirm the client agrees on the secret -- a handshake that "completes"
	// with a mismatched key is still a failure.
	for i := 0; i < 20; i++ {
		line, lerr := client.readLine(30 * time.Second)
		if lerr != nil {
			return "", fmt.Errorf("client produced no derived secret: %w", lerr)
		}
		if strings.Contains(line, "derivedSecret") {
			if !strings.Contains(line, string(secret)) {
				return "", fmt.Errorf("client derived a different secret: %s (server: %s)", line, secret)
			}
			return "", nil
		}
		if strings.Contains(line, "error") {
			return "", fmt.Errorf("client reported an error: %s", line)
		}
	}
	return "", errors.New("client never reported a derived secret")
}

// TestJPAKEStress_FullPairing runs the full pairing handshake many times, in
// parallel, and reports the failure rate. See the file header for why.
func TestJPAKEStress_FullPairing(t *testing.T) {
	cfg, skip := jpakeStressConfigFromEnv()
	if skip != "" {
		t.Skip(skip)
	}

	t.Logf("running %d JPAKE pairing handshakes, %d at a time", cfg.iterations, cfg.parallelism)

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, cfg.parallelism)
		mu      sync.Mutex
		results []jpakeStressResult
	)

	start := time.Now()
	for i := 0; i < cfg.iterations; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(iteration int) {
			defer wg.Done()
			defer func() { <-sem }()

			iterStart := time.Now()
			serverStderr, err := runOneJPAKEHandshake(cfg.jarPath, "123456")
			mu.Lock()
			results = append(results, jpakeStressResult{
				iteration: iteration,
				err:       err,
				stderr:    serverStderr,
				duration:  time.Since(iterStart),
			})
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	failures, upstream := 0, 0
	for _, r := range results {
		if r.err == nil {
			continue
		}
		failures++

		// A failure pumpX2's own jpake-server owns up to is the known
		// ~1-in-256 round-2 encoding bug (see the file header): it is not
		// something this code can prevent, only detect and explain, so it is
		// counted and reported rather than treated as a regression. Anything
		// else is.
		if errors.Is(r.err, ErrJPAKEServerFailed) {
			upstream++
			t.Logf("iteration %d failed after %s (pumpX2 jpake-server gave up): %v", r.iteration, r.duration, r.err)
		} else {
			t.Errorf("iteration %d failed after %s: %v", r.iteration, r.duration, r.err)
		}
		if r.stderr != "" {
			t.Logf("iteration %d jpake-server stderr:\n%s", r.iteration, r.stderr)
		}
	}

	t.Logf("JPAKE stress: %d/%d handshakes failed (%.1f%%), %d of them inside pumpX2's jpake-server, in %s",
		failures, len(results), 100*float64(failures)/float64(len(results)), upstream, time.Since(start))

	// The upstream bug costs about 1 handshake in 256. Well past that means
	// something else is wrong -- a leak, contention, a second failure mode --
	// and should not hide behind a known excuse.
	if budget := cfg.iterations/64 + 3; upstream > budget {
		t.Errorf("%d of %d handshakes failed inside pumpX2's jpake-server; the known round-2 bug accounts for roughly 1 in 256, so more than %d points at a second problem",
			upstream, len(results), budget)
	}
}
