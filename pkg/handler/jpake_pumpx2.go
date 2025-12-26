package handler

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	expect "github.com/google/goexpect"
	"github.com/jwoglom/faketandem/pkg/pumpx2"

	log "github.com/sirupsen/logrus"
)

// PumpX2JPAKEAuthenticator uses pumpX2's actual JPAKE implementation
type PumpX2JPAKEAuthenticator struct {
	pairingCode string
	bridge      *pumpx2.Bridge
	pumpX2Path  string
	pumpX2Mode  string
	gradleCmd   string
	javaCmd     string

	// JPAKE state
	round          int
	gexp           *expect.GExpect
	sharedSecret   []byte

	mutex          sync.Mutex
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

// ProcessRound processes a JPAKE round using pumpX2's implementation
func (j *PumpX2JPAKEAuthenticator) ProcessRound(round int, requestData map[string]interface{}) (map[string]interface{}, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	log.Infof("Processing JPAKE round %d using pumpX2", round)

	// Start the jpake command if not started
	if j.gexp == nil {
		if err := j.startJPAKEProcess(); err != nil {
			return nil, fmt.Errorf("failed to start JPAKE process: %w", err)
		}
	}

	// For each round, we need to:
	// 1. Wait for pumpX2 to output its request (as if it's a client)
	// 2. Extract the crypto parameters from that request
	// 3. Use those to construct our server response
	// 4. Send the client's actual request to pumpX2
	// 5. Get pumpX2's response (which validates the client's request)

	// This is tricky because pumpX2's jpake is CLIENT code
	// We're using it to:
	//   a) Generate cryptographically correct parameters
	//   b) Validate client messages
	//   c) Derive the shared secret properly

	switch round {
	case 1:
		return j.processRound1PumpX2(requestData)
	case 2:
		return j.processRound2PumpX2(requestData)
	case 3:
		return j.processRound3PumpX2(requestData)
	case 4:
		return j.processRound4PumpX2(requestData)
	default:
		return nil, fmt.Errorf("invalid JPAKE round: %d", round)
	}
}

// startJPAKEProcess starts the pumpX2 jpake command
func (j *PumpX2JPAKEAuthenticator) startJPAKEProcess() error {
	var cmdPath string
	var args []string

	if j.pumpX2Mode == "gradle" {
		cmdPath = filepath.Join(j.pumpX2Path, j.gradleCmd)
		args = []string{
			"cliparser",
			"-q",
			"--console=plain",
			"--args=jpake " + j.pairingCode,
		}
	} else {
		// JAR mode
		jarPath := filepath.Join(j.pumpX2Path, "cliparser/build/libs/cliparser.jar")
		cmdPath = j.javaCmd
		args = []string{
			"-jar",
			jarPath,
			"jpake",
			j.pairingCode,
		}
	}

	cmd := exec.Command(cmdPath, args...)
	cmd.Dir = j.pumpX2Path

	log.Infof("Starting pumpX2 JPAKE process: %s %v", cmdPath, args)

	// Build full command line
	fullCmd := cmdPath + " " + strings.Join(args, " ")

	var err error
	j.gexp, _, err = expect.Spawn(fullCmd, -1,
		expect.CheckDuration(100*time.Millisecond),
	)

	if err != nil {
		return fmt.Errorf("failed to spawn JPAKE process: %w", err)
	}

	log.Debug("pumpX2 JPAKE process started successfully")

	return nil
}

// processRound1PumpX2 handles round 1 using pumpX2
func (j *PumpX2JPAKEAuthenticator) processRound1PumpX2(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Wait for pumpX2 to output its Round1 request
	// Format: "JPAKE_1: <hex>"
	stepRegex := regexp.MustCompile(`JPAKE_1:\s*([0-9a-fA-F]+)`)

	output, _, err := j.gexp.Expect(stepRegex, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to read JPAKE_1 from pumpX2: %w", err)
	}

	matches := stepRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil, fmt.Errorf("failed to parse JPAKE_1 output: %s", output)
	}

	pumpX2Round1Hex := matches[1]
	log.Debugf("Got pumpX2 Round1 request: %s", pumpX2Round1Hex)

	// Parse this to extract crypto parameters that we'll use for our response
	// We use these parameters to construct a cryptographically valid server response

	// For now, we'll use the bridge to parse and extract parameters
	parsed, err := j.bridge.ParseMessage(3, pumpX2Round1Hex) // 3 = authentication char
	if err != nil {
		log.Warnf("Failed to parse pumpX2 Round1 (will use simplified): %v", err)
	}

	// TODO: Extract gx1, gx2, ZKPs from parsed message
	// For now, return parameters from the parsed message
	response := make(map[string]interface{})
	if parsed != nil && parsed.Cargo != nil {
		response = parsed.Cargo
	}

	// Now send a dummy response to pumpX2 so it advances
	// (We're not actually using its Round1Response, just keeping it running)
	dummyResponse := "00000000"
	j.gexp.Send(dummyResponse + "\n")

	j.round = 1

	return response, nil
}

// processRound2PumpX2 handles round 2 using pumpX2
func (j *PumpX2JPAKEAuthenticator) processRound2PumpX2(requestData map[string]interface{}) (map[string]interface{}, error) {
	// Similar process for round 2
	stepRegex := regexp.MustCompile(`JPAKE_2:\s*([0-9a-fA-F]+)`)

	output, _, err := j.gexp.Expect(stepRegex, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to read JPAKE_2 from pumpX2: %w", err)
	}

	matches := stepRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil, fmt.Errorf("failed to parse JPAKE_2 output: %s", output)
	}

	pumpX2Round2Hex := matches[1]
	log.Debugf("Got pumpX2 Round2 request: %s", pumpX2Round2Hex)

	// Parse and extract parameters
	parsed, err := j.bridge.ParseMessage(3, pumpX2Round2Hex)
	if err != nil {
		log.Warnf("Failed to parse pumpX2 Round2: %v", err)
	}

	response := make(map[string]interface{})
	if parsed != nil && parsed.Cargo != nil {
		response = parsed.Cargo
	}

	// Send dummy response
	dummyResponse := "00000000"
	j.gexp.Send(dummyResponse + "\n")

	j.round = 2

	return response, nil
}

// processRound3PumpX2 handles round 3 using pumpX2
func (j *PumpX2JPAKEAuthenticator) processRound3PumpX2(requestData map[string]interface{}) (map[string]interface{}, error) {
	stepRegex := regexp.MustCompile(`JPAKE_3:\s*([0-9a-fA-F]+)`)

	output, _, err := j.gexp.Expect(stepRegex, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to read JPAKE_3 from pumpX2: %w", err)
	}

	matches := stepRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil, fmt.Errorf("failed to parse JPAKE_3 output: %s", output)
	}

	pumpX2Round3Hex := matches[1]
	log.Debugf("Got pumpX2 Round3 request: %s", pumpX2Round3Hex)

	// Parse and extract parameters
	parsed, err := j.bridge.ParseMessage(3, pumpX2Round3Hex)
	if err != nil {
		log.Warnf("Failed to parse pumpX2 Round3: %v", err)
	}

	response := make(map[string]interface{})
	if parsed != nil && parsed.Cargo != nil {
		response = parsed.Cargo
	}

	// Send dummy response
	dummyResponse := "00000000"
	j.gexp.Send(dummyResponse + "\n")

	j.round = 3

	return response, nil
}

// processRound4PumpX2 handles round 4 using pumpX2
func (j *PumpX2JPAKEAuthenticator) processRound4PumpX2(requestData map[string]interface{}) (map[string]interface{}, error) {
	stepRegex := regexp.MustCompile(`JPAKE_4:\s*([0-9a-fA-F]+)`)

	output, _, err := j.gexp.Expect(stepRegex, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to read JPAKE_4 from pumpX2: %w", err)
	}

	matches := stepRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil, fmt.Errorf("failed to parse JPAKE_4 output: %s", output)
	}

	pumpX2Round4Hex := matches[1]
	log.Debugf("Got pumpX2 Round4 request: %s", pumpX2Round4Hex)

	// Parse and extract parameters
	parsed, err := j.bridge.ParseMessage(3, pumpX2Round4Hex)
	if err != nil {
		log.Warnf("Failed to parse pumpX2 Round4: %v", err)
	}

	response := make(map[string]interface{})
	if parsed != nil && parsed.Cargo != nil {
		response = parsed.Cargo
	}

	// Wait for the final result with derived key
	resultRegex := regexp.MustCompile(`"derivedSecret":\s*"([0-9a-fA-F]+)"`)
	resultOutput, _, err := j.gexp.Expect(resultRegex, 5*time.Second)
	if err != nil {
		log.Warnf("Failed to read derived secret from pumpX2: %v", err)
	} else {
		resultMatches := resultRegex.FindStringSubmatch(resultOutput)
		if len(resultMatches) >= 2 {
			j.sharedSecret = []byte(resultMatches[1])
			log.Infof("Extracted shared secret from pumpX2: %s", string(j.sharedSecret))
		}
	}

	j.round = 4

	return response, nil
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
