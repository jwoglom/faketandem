package handler

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	expect "github.com/google/goexpect"
	log "github.com/sirupsen/logrus"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// PumpX2JPAKEAuthenticator uses pumpX2's actual server-side JPAKE implementation
type PumpX2JPAKEAuthenticator struct {
	pairingCode string
	bridge      *pumpx2.Bridge
	pumpX2Path    string
	pumpX2Mode    string
	gradleCmd     string
	javaCmd       string
	pumpX2JarPath string

	// JPAKE state
	round        int
	gexp         *expect.GExpect
	sharedSecret []byte
	serverNonce  []byte

	// Response cache for each round
	round1aResponse map[string]interface{}
	round1bResponse map[string]interface{}
	round2Response  map[string]interface{}
	round3Response  map[string]interface{}
	round4Response  map[string]interface{}

	mutex sync.Mutex
}

// NewPumpX2JPAKEAuthenticator creates a new pumpX2-based JPAKE authenticator
func NewPumpX2JPAKEAuthenticator(pairingCode string, bridge *pumpx2.Bridge, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd, pumpX2JarPath string) *PumpX2JPAKEAuthenticator {
	return &PumpX2JPAKEAuthenticator{
		pairingCode:   pairingCode,
		bridge:        bridge,
		pumpX2Path:    pumpX2Path,
		pumpX2Mode:    pumpX2Mode,
		gradleCmd:     gradleCmd,
		javaCmd:       javaCmd,
		pumpX2JarPath: pumpX2JarPath,
		round:         0,
	}
}

// responseParamNames maps each real JPAKE response message name to its
// constructor's parameter names, in order. jpake-server's own JSON output
// (e.g. the line after "JPAKE_1A: ") already ran these through cliparser's
// "encode" command for its own bookkeeping, so its "messageParams" array is
// positional in this same constructor-argument order (see cliparser's
// Main.encode/Main.encode(byte,Message,Object[])).
var responseParamNames = map[string][]string{
	"Jpake1aResponse":               {"appInstanceId", "centralChallengeHash"},
	"Jpake1bResponse":               {"appInstanceId", "centralChallengeHash"},
	"Jpake2Response":                {"appInstanceId", "centralChallengeHash"},
	"Jpake3SessionKeyResponse":      {"appInstanceId", "nonce", "reserved"},
	"Jpake4KeyConfirmationResponse": {"appInstanceId", "nonce", "reserved", "hashDigest"},
}

// convertServerResponseToParams converts one of jpake-server's own response
// envelopes (the full JSON dumped after "JPAKE_1A:"/"JPAKE_1B:"/etc, shaped for
// its own display/debugging with keys like messageName/txId/messageParams/
// packets/characteristic) into the flat {fieldName: value} shape that
// bridge.EncodeMessage needs to build the response message fresh with the
// real request's txID. Without this, the 6-key envelope was passed through
// wholesale and cliparser's "encode" (which picks a constructor purely by
// matching parameter count) failed to find a match.
func convertServerResponseToParams(envelope map[string]interface{}) (map[string]interface{}, error) {
	messageName, _ := envelope["messageName"].(string)
	fieldNames, ok := responseParamNames[messageName]
	if !ok {
		return nil, fmt.Errorf("unknown response message name for param extraction: %s", messageName)
	}

	rawParams, ok := envelope["messageParams"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("response envelope for %s missing messageParams array", messageName)
	}
	if len(rawParams) != len(fieldNames) {
		return nil, fmt.Errorf("response envelope for %s has %d messageParams, expected %d", messageName, len(rawParams), len(fieldNames))
	}

	params := make(map[string]interface{}, len(fieldNames))
	for i, name := range fieldNames {
		params[name] = convertServerResponseValue(rawParams[i])
	}
	return params, nil
}

// convertServerResponseValue converts a single messageParams entry from
// jpake-server's JSON output into the shape cliparser's "encode" command
// expects for a byte[] constructor parameter: a hex string. org.json
// serializes a Java byte[] as a JSON array of signed byte values (e.g.
// [65,4,-116,...]); non-array values (e.g. the int appInstanceId) pass
// through unchanged.
func convertServerResponseValue(value interface{}) interface{} {
	arr, ok := value.([]interface{})
	if !ok {
		return value
	}
	b := make([]byte, len(arr))
	for i, v := range arr {
		if f, ok := v.(float64); ok {
			b[i] = byte(int64(f))
		}
	}
	return hex.EncodeToString(b)
}

// ProcessRound processes a JPAKE round using pumpX2's server-side implementation
func (j *PumpX2JPAKEAuthenticator) ProcessRound(round int, requestData map[string]interface{}) (map[string]interface{}, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	log.Infof("Processing JPAKE round %d using pumpX2 server mode", round)

	// Start the jpake-server command if not started
	if j.gexp == nil {
		if err := j.startJPAKEServerProcess(); err != nil {
			return nil, fmt.Errorf("failed to start JPAKE server process: %w", err)
		}

		// Read only the initial JPAKE_1A response
		// JPAKE_1B is read after we send the client's 1a request
		if err := j.readServerRound1aResponse(); err != nil {
			return nil, fmt.Errorf("failed to read server round 1a response: %w", err)
		}
	}

	switch round {
	case 1:
		return j.processRound1(requestData)
	case 2:
		return j.processRound2(requestData)
	case 3:
		return j.processRound3(requestData)
	case 4:
		return j.processRound4(requestData)
	default:
		return nil, fmt.Errorf("invalid JPAKE round: %d", round)
	}
}

// getRepoRoot returns the absolute path to the git repository root
// Falls back to current working directory if not in a git repo
func getRepoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		// Not in a git repo, use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			log.Warnf("Failed to get current working directory: %v", err)
			return "."
		}
		return cwd
	}
	return strings.TrimSpace(string(output))
}

// startJPAKEServerProcess starts the pumpX2 jpake-server command
func (j *PumpX2JPAKEAuthenticator) startJPAKEServerProcess() error {
	var scriptPath string
	var args []string

	if j.pumpX2Mode == "gradle" {
		// Use wrapper script that handles directory change and gradle invocation
		// Get absolute path to the script in the repo
		repoRoot := getRepoRoot()
		scriptPath = filepath.Join(repoRoot, "scripts", "cliparser-gradle.sh")
		args = []string{
			j.pumpX2Path,
			"jpake-server",
			j.pairingCode,
		}
	} else {
		// JAR mode - run java directly
		jarPath := j.pumpX2JarPath
		if jarPath == "" {
			jarPath = filepath.Join(j.pumpX2Path, "cliparser/build/libs/pumpx2-cliparser-all.jar")
		}
		scriptPath = j.javaCmd
		args = []string{
			"-jar",
			jarPath,
			"jpake-server",
			j.pairingCode,
		}
	}

	log.Infof("Starting pumpX2 JPAKE server process: %s %v", scriptPath, args)

	// Test if we can run the command
	cmd := exec.Command(scriptPath, args...)
	if err := cmd.Start(); err != nil {
		log.Errorf("Failed to start command %s %v: %v", scriptPath, args, err)
		return fmt.Errorf("failed to start JPAKE server process: %w", err)
	}

	// Wait a bit to see if the process immediately exits
	time.Sleep(100 * time.Millisecond)
	
	// Check if process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		log.Errorf("Process exited immediately with status: %s", cmd.ProcessState.String())
		return fmt.Errorf("JPAKE server process exited immediately")
	}

	// Kill the test process - we'll spawn it properly with expect
	if cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			log.Warnf("Failed to kill test process: %v", err)
		}
	}

	// Now spawn with expect using the script
	fullCmd := append([]string{scriptPath}, args...)
	log.Debugf("Spawning with expect: %v", fullCmd)

	var err error
	j.gexp, _, err = expect.SpawnWithArgs(
		fullCmd,
		-1,
		expect.CheckDuration(100*time.Millisecond),
		expect.PartialMatch(true),
		expect.Verbose(true),
	)

	if err != nil {
		log.Errorf("Failed to spawn with expect: %v", err)
		return fmt.Errorf("failed to spawn JPAKE server process with expect: %w", err)
	}

	log.Debug("pumpX2 JPAKE server process started successfully with expect")

	// Give the process a moment to start and output initial messages
	time.Sleep(200 * time.Millisecond)

	return nil
}

// readServerRound1aResponse reads only the server's initial JPAKE_1A response
// JPAKE_1B is read after sending client's round 1a request (pumpX2 waits for input)
func (j *PumpX2JPAKEAuthenticator) readServerRound1aResponse() error {
	// Read JPAKE_1A response
	// The regex needs to handle potential stderr output before the JPAKE_1A line
	// Match JPAKE_1A: followed by JSON (JSONObject.toString() outputs single-line JSON)
	round1aRegex := regexp.MustCompile(`(?s).*?JPAKE_1A:\s*(\{.*?\})`)
	output, _, err := j.gexp.Expect(round1aRegex, 30*time.Second)
	if err != nil {
		// Try to get any remaining output for debugging
		log.Errorf("Failed to read JPAKE_1A. Last output captured: %s", output)
		return fmt.Errorf("failed to read JPAKE_1A from pumpX2: %w", err)
	}

	matches := round1aRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		log.Errorf("Failed to parse JPAKE_1A. Full output: %s", output)
		return fmt.Errorf("failed to parse JPAKE_1A output: %s", output)
	}

	// Parse the JSON response
	if err := json.Unmarshal([]byte(matches[1]), &j.round1aResponse); err != nil {
		log.Errorf("Failed to unmarshal JPAKE_1A JSON: %s. Error: %v", matches[1], err)
		return fmt.Errorf("failed to unmarshal JPAKE_1A response: %w", err)
	}

	log.Debugf("Got server Round1a response: %+v", j.round1aResponse)

	return nil
}

// readServerRound1bResponse reads the server's JPAKE_1B response after sending client's 1a
func (j *PumpX2JPAKEAuthenticator) readServerRound1bResponse() error {
	// Read JPAKE_1B response
	// Match JPAKE_1B: followed by JSON (JSONObject.toString() outputs single-line JSON)
	round1bRegex := regexp.MustCompile(`(?s).*?JPAKE_1B:\s*(\{.*?\})`)
	output, _, err := j.gexp.Expect(round1bRegex, 30*time.Second)
	if err != nil {
		log.Errorf("Failed to read JPAKE_1B. Last output captured: %s", output)
		return fmt.Errorf("failed to read JPAKE_1B from pumpX2: %w", err)
	}

	matches := round1bRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		log.Errorf("Failed to parse JPAKE_1B. Full output: %s", output)
		return fmt.Errorf("failed to parse JPAKE_1B output: %s", output)
	}

	if err := json.Unmarshal([]byte(matches[1]), &j.round1bResponse); err != nil {
		log.Errorf("Failed to unmarshal JPAKE_1B JSON: %s. Error: %v", matches[1], err)
		return fmt.Errorf("failed to unmarshal JPAKE_1B response: %w", err)
	}

	log.Debugf("Got server Round1b response: %+v", j.round1bResponse)

	return nil
}

// processRound1 handles round 1 (combines 1a and 1b)
func (j *PumpX2JPAKEAuthenticator) processRound1(requestData map[string]interface{}) (map[string]interface{}, error) {
	// The round 1a involves two sub-rounds
	// First call (round==0): Send client's 1a, read JPAKE_1B, return round1aResponse
	// Second call (round==1): Send client's 1b, read JPAKE_2, return round1bResponse

	if j.round == 0 {
		// First call - send client's Jpake1aRequest
		requestHex := j.encodeClientRequest(requestData)

		log.Debugf("Sending client Jpake1aRequest to pumpX2: %s", requestHex)
		if err := j.gexp.Send(requestHex + "\n"); err != nil {
			log.Warnf("Failed to send to pumpX2: %v", err)
		}

		// Now read JPAKE_1B (pumpX2 outputs it after receiving client's 1a)
		if err := j.readServerRound1bResponse(); err != nil {
			return nil, fmt.Errorf("failed to read JPAKE_1B after sending client 1a: %w", err)
		}

		j.round = 1
		return convertServerResponseToParams(j.round1aResponse)
	}

	// Second call - send client's Jpake1bRequest
	requestHex := j.encodeClientRequest(requestData)

	log.Debugf("Sending client Jpake1bRequest to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Warnf("Failed to send to pumpX2: %v", err)
	}

	// Read server's round 2 response (pumpX2 sends it after receiving round 1b)
	round2Regex := regexp.MustCompile(`JPAKE_2:\s*({.+})`)
	output, _, err := j.gexp.Expect(round2Regex, 30*time.Second)
	if err != nil {
		if j.gexp != nil {
			log.Errorf("Failed to read JPAKE_2. Last output captured: %s", output)
		}
		return nil, fmt.Errorf("failed to read JPAKE_2 from pumpX2: %w", err)
	}

	matches := round2Regex.FindStringSubmatch(output)
	if len(matches) < 2 {
		log.Errorf("Failed to parse JPAKE_2. Full output: %s", output)
		return nil, fmt.Errorf("failed to parse JPAKE_2 output: %s", output)
	}

	if err := json.Unmarshal([]byte(matches[1]), &j.round2Response); err != nil {
		log.Errorf("Failed to unmarshal JPAKE_2 JSON: %s. Error: %v", matches[1], err)
		return nil, fmt.Errorf("failed to unmarshal JPAKE_2 response: %w", err)
	}

	log.Debugf("Got server Round2 response: %+v", j.round2Response)

	return convertServerResponseToParams(j.round1bResponse)
}

// processRound2 handles round 2
//
// jpake-server's Main.jpakeAuthServer does NOT print anything in response to
// the client's round 2 request (Jpake2Request) -- its own round 2 public value
// was already sent proactively as "JPAKE_2:" right after round 1b (see
// processRound1), before it ever reads the client's round 2 message. It reads
// Jpake2Request from stdin and then, without printing, immediately blocks
// trying to read the client's round 3 request (Jpake3SessionKeyRequest) next;
// only once *that* arrives does it print "JPAKE_3:". So this function must
// only forward the client's request and return the already-cached
// Jpake2Response -- trying to read "JPAKE_3:" here (as an earlier version of
// this code did) deadlocks, since jpake-server won't produce it until the
// round 3 request (sent by processRound3, on a later call) has also arrived.
func (j *PumpX2JPAKEAuthenticator) processRound2(requestData map[string]interface{}) (map[string]interface{}, error) {
	requestHex := j.encodeClientRequest(requestData)

	log.Debugf("Sending client Jpake2Request to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Warnf("Failed to send Jpake2Request to pumpX2: %v", err)
	}

	j.round = 2

	return convertServerResponseToParams(j.round2Response)
}

// processRound3 handles round 3
func (j *PumpX2JPAKEAuthenticator) processRound3(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Send client's Jpake3SessionKeyRequest. jpake-server has been blocked
	// waiting for exactly this since it finished reading round 2 (see
	// processRound2) -- only once it arrives does jpake-server print "JPAKE_3:".
	requestHex := j.encodeClientRequest(requestData)

	log.Debugf("Sending client Jpake3SessionKeyRequest to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Warnf("Failed to send to pumpX2: %v", err)
	}

	round3Regex := regexp.MustCompile(`JPAKE_3:\s*({.+})`)
	output, _, err := j.gexp.Expect(round3Regex, 30*time.Second)
	if err != nil {
		if j.gexp != nil {
			log.Errorf("Failed to read JPAKE_3. Last output captured: %s", output)
		}
		return nil, fmt.Errorf("failed to read JPAKE_3 from pumpX2: %w", err)
	}

	matches := round3Regex.FindStringSubmatch(output)
	if len(matches) < 2 {
		log.Errorf("Failed to parse JPAKE_3. Full output: %s", output)
		return nil, fmt.Errorf("failed to parse JPAKE_3 output: %s", output)
	}

	if err := json.Unmarshal([]byte(matches[1]), &j.round3Response); err != nil {
		log.Errorf("Failed to unmarshal JPAKE_3 JSON: %s. Error: %v", matches[1], err)
		return nil, fmt.Errorf("failed to unmarshal JPAKE_3 response: %w", err)
	}

	log.Debugf("Got server Round3 response: %+v", j.round3Response)

	j.round = 3

	// Extract server nonce from round3 response
	// pumpX2 v1.8.0+ renamed "deviceKeyNonce" to "nonce"
	if serverNonceHex, ok := j.round3Response["nonce"].(string); ok {
		j.serverNonce = []byte(serverNonceHex)
	} else if serverNonceHex, ok := j.round3Response["deviceKeyNonce"].(string); ok {
		// Fallback for older pumpX2 versions
		j.serverNonce = []byte(serverNonceHex)
	}

	return convertServerResponseToParams(j.round3Response)
}

// processRound4 handles round 4
func (j *PumpX2JPAKEAuthenticator) processRound4(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Send client's Jpake4KeyConfirmationRequest
	requestHex := j.encodeClientRequest(requestData)

	log.Debugf("Sending client Jpake4KeyConfirmationRequest to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Warnf("Failed to send Jpake4KeyConfirmationRequest to pumpX2: %v", err)
	}

	// Read server's round 4 response
	round4Regex := regexp.MustCompile(`JPAKE_4:\s*({.+})`)
	output, _, err := j.gexp.Expect(round4Regex, 30*time.Second)
	if err != nil {
		if j.gexp != nil {
			log.Errorf("Failed to read JPAKE_4. Last output captured: %s", output)
		}
		return nil, fmt.Errorf("failed to read JPAKE_4 from pumpX2: %w", err)
	}

	matches := round4Regex.FindStringSubmatch(output)
	if len(matches) < 2 {
		log.Errorf("Failed to parse JPAKE_4. Full output: %s", output)
		return nil, fmt.Errorf("failed to parse JPAKE_4 output: %s", output)
	}

	if err := json.Unmarshal([]byte(matches[1]), &j.round4Response); err != nil {
		log.Errorf("Failed to unmarshal JPAKE_4 JSON: %s. Error: %v", matches[1], err)
		return nil, fmt.Errorf("failed to unmarshal JPAKE_4 response: %w", err)
	}

	log.Debugf("Got server Round4 response: %+v", j.round4Response)

	// Read the final result with derived secret
	resultRegex := regexp.MustCompile(`({[^{}]*"derivedSecret"[^{}]*})`)
	resultOutput, _, err := j.gexp.Expect(resultRegex, 30*time.Second)
	if err != nil {
		if j.gexp != nil {
			log.Errorf("Failed to read derivedSecret. Last output captured: %s", resultOutput)
		}
		return nil, fmt.Errorf("failed to read derived secret from pumpX2: %w", err)
	}

	resultMatches := resultRegex.FindStringSubmatch(resultOutput)
	if len(resultMatches) >= 2 {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(resultMatches[1]), &result); err != nil {
			log.Errorf("Failed to unmarshal result JSON: %s. Error: %v", resultMatches[1], err)
			log.Warnf("Failed to unmarshal result JSON: %v", err)
		} else {
			if derivedSecretHex, ok := result["derivedSecret"].(string); ok {
				j.sharedSecret = []byte(derivedSecretHex)
				log.Infof("Extracted shared secret from pumpX2: %s", derivedSecretHex)
			}
		}
	}

	j.round = 4

	return convertServerResponseToParams(j.round4Response)
}

// encodeClientRequest encodes a client request using pumpX2's format
// Always returns a non-empty string (uses fallback if encoding fails)
func (j *PumpX2JPAKEAuthenticator) encodeClientRequest(requestData map[string]interface{}) string {
	// Extract message name from request data
	messageName, ok := requestData["messageName"].(string)
	if !ok {
		log.Warnf("Request data missing messageName")
		return "00"
	}

	// If the caller gave us the client's original raw BLE fragments, forward
	// them to jpake-server verbatim instead of re-encoding through a second
	// cliparser JVM invocation -- jpake-server doesn't validate the request's
	// txID, so the original bytes work as-is and this avoids ~1s+ of JVM
	// startup latency per JPAKE round (a real device+app capture showed the
	// phone disconnecting shortly after round 3, and cumulative per-round
	// latency from spawning a JVM both to forward the request and to build the
	// response is the leading suspect).
	if rawPacketsHex, ok := requestData["rawPacketsHex"].([]string); ok && len(rawPacketsHex) > 0 {
		result := strings.Join(rawPacketsHex, " ")
		log.Debugf("Forwarding client request verbatim (no re-encode): %s -> %s", messageName, result)
		return result
	}

	// Try to use the bridge to encode the message if available
	if j.bridge != nil {
		// Build params map excluding messageName and cargo. cliparser's "encode"
		// picks a constructor purely by matching parameter *count*, so any extra
		// key makes it fail to find one -- "cargo" is a base Message field our
		// output parser always includes (every message's toString() has it), but
		// it's never an actual constructor parameter (constructors take the
		// specific named fields, e.g. appInstanceId/centralChallenge; "cargo" is
		// set internally from those during parse()).
		params := make(map[string]interface{})
		for key, value := range requestData {
			if key != "messageName" && key != "cargo" {
				params[key] = value
			}
		}

		// Jpake3SessionKeyRequest has only one real field (challengeParam, an
		// int) besides cargo, so excluding "cargo" above leaves exactly one
		// param -- which collides with the class's OTHER one-arg constructor,
		// Jpake3SessionKeyRequest(byte[] rawCargo). cliparser's "encode" picks
		// whichever constructor Class.getConstructors() happens to return first
		// for that parameter count, and empirically that's the byte[] one, which
		// then fails to convert challengeParam's plain int/JSON-number value to
		// byte[] ("Cannot convert java.lang.Integer to byte[]"). Route around
		// the ambiguity by targeting that raw-cargo constructor deliberately: it
		// reconstructs an identical message from the same bytes.
		if messageName == "Jpake3SessionKeyRequest" {
			params = map[string]interface{}{"cargo": requestData["cargo"]}
		}

		// Use bridge to encode the message
		// Use txID 0 for simplicity - pumpX2 jpake-server doesn't validate txID
		encoded, err := j.bridge.EncodeMessage(0, messageName, params)
		if err == nil && len(encoded.Packets) > 0 {
			// jpake-server reads one line from stdin and hands it directly to
			// cliparser's "parse" command, which expects each raw BLE fragment as
			// its own whitespace-delimited token (see Main.splitRawHexPackets) --
			// NOT one concatenated blob.
			result := strings.Join(encoded.Packets, " ")
			log.Debugf("Encoded client request via bridge: %s -> %s", messageName, result)
			return result
		}
		// Bridge encoding failed or returned no packets - fall through to fallback
		if err != nil {
			log.Debugf("Bridge encoding failed (will use fallback): %v", err)
		} else {
			log.Debugf("Bridge encoding returned no packets (will use fallback)")
		}
	}

	// Fallback: extract the hex data directly from request params
	// pumpX2's cliparser may not support encoding JPAKE request messages,
	// so we construct a minimal message from the raw data
	if hexData, ok := requestData["centralChallengeHash"].(string); ok && len(hexData) > 0 {
		previewLen := 40
		if len(hexData) < previewLen {
			previewLen = len(hexData)
		}
		log.Debugf("Using centralChallengeHash as fallback for %s: %s...", messageName, hexData[:previewLen])
		return hexData
	}

	// For messages without centralChallengeHash (like Jpake3SessionKeyRequest),
	// return a placeholder that won't crash pumpX2's parse function
	// pumpX2 will reject it, but at least it won't crash on empty input
	log.Debugf("No fallback data available for %s, using placeholder", messageName)
	return "00"
}

// GetSharedSecret returns the derived shared secret
func (j *PumpX2JPAKEAuthenticator) GetSharedSecret() ([]byte, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.round < 4 {
		return nil, fmt.Errorf("JPAKE not complete, current round: %d", j.round)
	}

	return j.sharedSecret, nil
}

// IsComplete returns true if JPAKE authentication is complete
func (j *PumpX2JPAKEAuthenticator) IsComplete() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	return j.round == 4
}

// Close cleans up the goexpect process
func (j *PumpX2JPAKEAuthenticator) Close() error {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.gexp != nil {
		return j.gexp.Close()
	}

	return nil
}
