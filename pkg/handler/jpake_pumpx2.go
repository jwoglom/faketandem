package handler

import (
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
	pumpX2Path  string
	pumpX2Mode  string
	gradleCmd   string
	javaCmd     string

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
func NewPumpX2JPAKEAuthenticator(pairingCode string, bridge *pumpx2.Bridge, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd string) *PumpX2JPAKEAuthenticator {
	return &PumpX2JPAKEAuthenticator{
		pairingCode: pairingCode,
		bridge:      bridge,
		pumpX2Path:  pumpX2Path,
		pumpX2Mode:  pumpX2Mode,
		gradleCmd:   gradleCmd,
		javaCmd:     javaCmd,
		round:       0,
	}
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
		jarPath := filepath.Join(j.pumpX2Path, "cliparser/build/libs/cliparser.jar")
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
		requestHex, err := j.encodeClientRequest(requestData)
		if err != nil {
			log.Warnf("Failed to encode client Jpake1aRequest: %v", err)
		}

		log.Debugf("Sending client Jpake1aRequest to pumpX2: %s", requestHex)
		if err := j.gexp.Send(requestHex + "\n"); err != nil {
			log.Warnf("Failed to send to pumpX2: %v", err)
		}

		// Now read JPAKE_1B (pumpX2 outputs it after receiving client's 1a)
		if err := j.readServerRound1bResponse(); err != nil {
			return nil, fmt.Errorf("failed to read JPAKE_1B after sending client 1a: %w", err)
		}

		j.round = 1
		return j.round1aResponse, nil
	}

	// Second call - send client's Jpake1bRequest
	requestHex, err := j.encodeClientRequest(requestData)
	if err != nil {
		log.Warnf("Failed to encode client Jpake1bRequest: %v", err)
	}

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

	return j.round1bResponse, nil
}

// processRound2 handles round 2
func (j *PumpX2JPAKEAuthenticator) processRound2(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Send client's Jpake2Request
	requestHex, err := j.encodeClientRequest(requestData)
	if err != nil {
		log.Warnf("Failed to encode client Jpake2Request: %v", err)
	}

	log.Debugf("Sending client Jpake2Request to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Errorf("Failed to send Jpake2Request to pumpX2: %v", err)
		log.Warnf("Failed to send to pumpX2: %v", err)
	}

	// Read server's round 3 response
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

	j.round = 2

	return j.round2Response, nil
}

// processRound3 handles round 3
//nolint:unparam // error return required by interface, may be used in future
func (j *PumpX2JPAKEAuthenticator) processRound3(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Send client's Jpake3SessionKeyRequest
	requestHex, err := j.encodeClientRequest(requestData)
	if err != nil {
		log.Warnf("Failed to encode client Jpake3SessionKeyRequest: %v", err)
	}

	log.Debugf("Sending client Jpake3SessionKeyRequest to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Warnf("Failed to send to pumpX2: %v", err)
	}

	j.round = 3

	// Extract server nonce from round3 response
	if serverNonceHex, ok := j.round3Response["deviceKeyNonce"].(string); ok {
		j.serverNonce = []byte(serverNonceHex)
	}

	return j.round3Response, nil
}

// processRound4 handles round 4
func (j *PumpX2JPAKEAuthenticator) processRound4(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Send client's Jpake4KeyConfirmationRequest
	requestHex, err := j.encodeClientRequest(requestData)
	if err != nil {
		log.Warnf("Failed to encode client Jpake4KeyConfirmationRequest: %v", err)
	}

	log.Debugf("Sending client Jpake4KeyConfirmationRequest to pumpX2: %s", requestHex)
	if err := j.gexp.Send(requestHex + "\n"); err != nil {
		log.Errorf("Failed to send Jpake4KeyConfirmationRequest to pumpX2: %v", err)
		log.Warnf("Failed to send to pumpX2: %v", err)
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

	return j.round4Response, nil
}

// encodeClientRequest encodes a client request using pumpX2's format
func (j *PumpX2JPAKEAuthenticator) encodeClientRequest(requestData map[string]interface{}) (string, error) {
	// If bridge is not available, return empty string
	if j.bridge == nil {
		log.Warnf("No bridge available for encoding client request")
		return "", nil
	}

	// Extract message name from request data
	messageName, ok := requestData["messageName"].(string)
	if !ok {
		log.Warnf("Request data missing messageName")
		return "", nil
	}

	// Build params map excluding messageName
	params := make(map[string]interface{})
	for key, value := range requestData {
		if key != "messageName" {
			params[key] = value
		}
	}

	// Use bridge to encode the message
	// Use txID 0 for simplicity - pumpX2 jpake-server doesn't validate txID
	encoded, err := j.bridge.EncodeMessage(0, messageName, params)
	if err != nil {
		log.Warnf("Failed to encode client request: %v", err)
		return "", err
	}

	// Join all packets into a single hex string for stdin input
	// pumpX2's jpake-server expects a single line of hex data
	if len(encoded.Packets) == 0 {
		log.Warnf("Encoded message has no packets")
		return "", nil
	}

	// For pump messages, concatenate all packets (stripping any packet headers)
	// The first bytes of each packet are usually sequence/length info for BLE
	// For stdin input to jpake-server, we might need the raw message
	// For now, try joining all packets and let pumpX2 parse it
	result := strings.Join(encoded.Packets, "")
	log.Debugf("Encoded client request: %s -> %s", messageName, result)

	return result, nil
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
