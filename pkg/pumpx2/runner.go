package pumpx2

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// JPAKEResponseProvider is a function that provides responses for each JPAKE round
// It receives the step name and the hex-encoded request, and returns the hex-encoded response
type JPAKEResponseProvider func(step string, requestHex string) (responseHex string, err error)

// Runner is an interface for executing cliparser commands
type Runner interface {
	// Parse decodes a message from its raw BLE fragments. rawPacketsHex must be the
	// original, unstripped fragment bytes (including framing) in receive order --
	// see PacketBuffer.RawPacketsHex for why the stripped/concatenated payload
	// alone isn't enough.
	Parse(btChar string, rawPacketsHex []string) (string, error)
	Encode(txID int, messageName string, params map[string]interface{}) (string, error)
	ExecuteJPAKE(pairingCode string, responseProvider JPAKEResponseProvider) (string, error)
	ListAllCommands() (string, error)
}

// GradleRunner executes cliparser via gradle
type GradleRunner struct {
	pumpX2Path string
	gradleCmd  string
}

// NewGradleRunner creates a new gradle runner
func NewGradleRunner(pumpX2Path, gradleCmd string) *GradleRunner {
	return &GradleRunner{
		pumpX2Path: pumpX2Path,
		gradleCmd:  gradleCmd,
	}
}

// Parse parses a message using gradle cliparser. btChar is currently unused --
// the cliparser "parse" command determines the characteristic from the opcode
// itself (see CharacteristicGuesser in pumpX2) -- but is kept for interface
// symmetry with other Runner implementations that may need it.
func (r *GradleRunner) Parse(btChar string, rawPacketsHex []string) (string, error) {
	// The cliparser "parse" command expects each raw BLE fragment (including its
	// framing bytes) as its own whitespace-delimited token; see
	// Main.splitRawHexPackets in pumpX2's cliparser module.
	hexValue := strings.Join(rawPacketsHex, " ")

	// Execute: ./gradlew cliparser -q --console=plain --args="parse <fragments>"
	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args=parse "+hexValue)
	cmd.Dir = r.pumpX2Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing gradle parse: btChar=%s, fragments=%s", btChar, hexValue)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gradle parse failed: %w\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	log.Tracef("Gradle parse output: %s", output)

	return output, nil
}

// Encode builds a message using gradle cliparser
func (r *GradleRunner) Encode(txID int, messageName string, params map[string]interface{}) (string, error) {
	// Build args: encode <txID> <messageName> <params>
	var paramsJSON string
	if len(params) == 0 {
		paramsJSON = "{}"
	} else {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return "", fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = string(paramsBytes)
	}

	args := fmt.Sprintf("encode %d %s %s", txID, messageName, paramsJSON)

	// Execute: ./gradlew cliparser -q --args="..."
	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args="+args)
	cmd.Dir = r.pumpX2Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing gradle encode: %s", args)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gradle encode failed: %w\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	log.Tracef("Gradle encode output: %s", output)

	return output, nil
}

// ExecuteJPAKE runs JPAKE authentication with interactive stdin/stdout
func (r *GradleRunner) ExecuteJPAKE(pairingCode string, responseProvider JPAKEResponseProvider) (string, error) {
	args := fmt.Sprintf("jpake %s", pairingCode)

	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args="+args)
	cmd.Dir = r.pumpX2Path

	// Set up pipes for interactive communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	defer func() {
		if err := stdin.Close(); err != nil {
			log.Debugf("Error closing stdin: %v", err)
		}
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Infof("Starting interactive JPAKE authentication with pairing code: %s", pairingCode)

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start JPAKE command: %w", err)
	}

	// Read from stdout line by line and interact
	scanner := bufio.NewScanner(stdout)
	var finalResult strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		log.Tracef("JPAKE output: %s", line)

		// Check if this is a step output (format: "STEP_NAME: encoded_hex")
		if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				step := strings.TrimSpace(parts[0])
				requestHex := strings.TrimSpace(parts[1])

				log.Infof("JPAKE step: %s, request: %s", step, requestHex)

				// Call the response provider to get the pump's response
				responseHex, err := responseProvider(step, requestHex)
				if err != nil {
					log.Errorf("Response provider failed for step %s: %v", step, err)
					if err := stdin.Close(); err != nil {
					log.Debugf("Error closing stdin: %v", err)
				}
					_ = cmd.Wait() // Ignore error in cleanup path
					return "", fmt.Errorf("response provider error at step %s: %w", step, err)
				}

				log.Infof("JPAKE step %s: sending response: %s", step, responseHex)

				// Write the response to stdin
				if _, err := fmt.Fprintln(stdin, responseHex); err != nil {
					return "", fmt.Errorf("failed to write response for step %s: %w", step, err)
				}
			}
		} else {
			// This might be the final result (JSON output)
			finalResult.WriteString(line)
			finalResult.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading JPAKE output: %w", err)
	}

	// Close stdin to signal we're done
	if err := stdin.Close(); err != nil {
		log.Debugf("Error closing stdin: %v", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("JPAKE command failed: %w\nStderr: %s", err, stderrStr)
		}
		return "", fmt.Errorf("JPAKE command failed: %w", err)
	}

	result := strings.TrimSpace(finalResult.String())
	log.Infof("JPAKE authentication completed: %s", result)

	return result, nil
}

// ListAllCommands lists all available commands
func (r *GradleRunner) ListAllCommands() (string, error) {
	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args=listallcommands")
	cmd.Dir = r.pumpX2Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gradle listallcommands failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// JarRunner executes cliparser via JAR file
type JarRunner struct {
	jarPath string
	javaCmd string
}

// NewJarRunner creates a new JAR runner
func NewJarRunner(jarPath, javaCmd string) *JarRunner {
	return &JarRunner{
		jarPath: jarPath,
		javaCmd: javaCmd,
	}
}

// Parse parses a message using JAR cliparser. btChar is currently unused -- see
// GradleRunner.Parse.
func (r *JarRunner) Parse(btChar string, rawPacketsHex []string) (string, error) {
	// The cliparser "parse" command expects each raw BLE fragment (including its
	// framing bytes) as its own whitespace-delimited token; see
	// Main.splitRawHexPackets in pumpX2's cliparser module.
	hexValue := strings.Join(rawPacketsHex, " ")
	args := []string{"-jar", r.jarPath, "parse", hexValue}

	cmd := exec.Command(r.javaCmd, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing JAR parse: %s", hexValue)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("JAR parse failed: %w\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	log.Tracef("JAR parse output: %s", output)

	return output, nil
}

// Encode builds a message using JAR cliparser. The cliparser "encode" command
// expects its params argument as a single JSON object string, not key=value
// pairs -- confirmed empirically against a real cliparser jar (a bare key=value
// arg throws org.json.JSONException: "A JSONObject text must begin with '{'").
func (r *JarRunner) Encode(txID int, messageName string, params map[string]interface{}) (string, error) {
	var paramsJSON string
	if len(params) == 0 {
		paramsJSON = "{}"
	} else {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return "", fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = string(paramsBytes)
	}

	args := []string{"-jar", r.jarPath, "encode", fmt.Sprintf("%d", txID), messageName, paramsJSON}

	cmd := exec.Command(r.javaCmd, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing JAR encode: %s %s", strings.Join(args, " "), "")

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("JAR encode failed: %w\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	log.Tracef("JAR encode output: %s", output)

	return output, nil
}

// ExecuteJPAKE runs JPAKE authentication with interactive stdin/stdout
func (r *JarRunner) ExecuteJPAKE(pairingCode string, responseProvider JPAKEResponseProvider) (string, error) {
	cmd := exec.Command(r.javaCmd, "-jar", r.jarPath, "jpake", pairingCode)

	// Set up pipes for interactive communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	defer func() {
		if err := stdin.Close(); err != nil {
			log.Debugf("Error closing stdin: %v", err)
		}
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Infof("Starting interactive JPAKE authentication (JAR mode) with pairing code: %s", pairingCode)

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start JPAKE command: %w", err)
	}

	// Read from stdout line by line and interact
	scanner := bufio.NewScanner(stdout)
	var finalResult strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		log.Tracef("JPAKE output: %s", line)

		// Check if this is a step output (format: "STEP_NAME: encoded_hex")
		if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				step := strings.TrimSpace(parts[0])
				requestHex := strings.TrimSpace(parts[1])

				log.Infof("JPAKE step: %s, request: %s", step, requestHex)

				// Call the response provider to get the pump's response
				responseHex, err := responseProvider(step, requestHex)
				if err != nil {
					log.Errorf("Response provider failed for step %s: %v", step, err)
					if err := stdin.Close(); err != nil {
					log.Debugf("Error closing stdin: %v", err)
				}
					_ = cmd.Wait() // Ignore error in cleanup path
					return "", fmt.Errorf("response provider error at step %s: %w", step, err)
				}

				log.Infof("JPAKE step %s: sending response: %s", step, responseHex)

				// Write the response to stdin
				if _, err := fmt.Fprintln(stdin, responseHex); err != nil {
					return "", fmt.Errorf("failed to write response for step %s: %w", step, err)
				}
			}
		} else {
			// This might be the final result (JSON output)
			finalResult.WriteString(line)
			finalResult.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading JPAKE output: %w", err)
	}

	// Close stdin to signal we're done
	if err := stdin.Close(); err != nil {
		log.Debugf("Error closing stdin: %v", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("JPAKE command failed: %w\nStderr: %s", err, stderrStr)
		}
		return "", fmt.Errorf("JPAKE command failed: %w", err)
	}

	result := strings.TrimSpace(finalResult.String())
	log.Infof("JPAKE authentication completed: %s", result)

	return result, nil
}

// ListAllCommands lists all available commands
func (r *JarRunner) ListAllCommands() (string, error) {
	cmd := exec.Command(r.javaCmd, "-jar", r.jarPath, "listallcommands")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("JAR listallcommands failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
