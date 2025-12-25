package pumpx2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Runner is an interface for executing cliparser commands
type Runner interface {
	Parse(btChar, hexValue string) (string, error)
	Encode(txID int, messageName string, params map[string]interface{}) (string, error)
	ExecuteJPAKE(pairingCode string) (string, error)
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

// Parse parses a message using gradle cliparser
func (r *GradleRunner) Parse(btChar, hexValue string) (string, error) {
	// Build the JSON input
	input := map[string]interface{}{
		"type":   "ReadResp",
		"btChar": btChar,
		"value":  hexValue,
		"ts":     "",
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input JSON: %w", err)
	}

	// Execute: ./gradlew cliparser -q --console=plain --args='json'
	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args=json")
	cmd.Dir = r.pumpX2Path

	// Provide JSON as stdin
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing gradle parse: btChar=%s, value=%s", btChar, hexValue)
	log.Tracef("Input JSON: %s", string(inputJSON))

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

// ExecuteJPAKE runs JPAKE authentication
func (r *GradleRunner) ExecuteJPAKE(pairingCode string) (string, error) {
	args := fmt.Sprintf("jpake %s", pairingCode)

	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args="+args)
	cmd.Dir = r.pumpX2Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing gradle JPAKE: %s", args)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gradle JPAKE failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
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

// Parse parses a message using JAR cliparser
func (r *JarRunner) Parse(btChar, hexValue string) (string, error) {
	// For JAR mode, use the old 'parse' command format
	// Convert btChar to the appropriate characteristic name
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

// Encode builds a message using JAR cliparser
func (r *JarRunner) Encode(txID int, messageName string, params map[string]interface{}) (string, error) {
	args := []string{"-jar", r.jarPath, "encode", fmt.Sprintf("%d", txID), messageName}

	// Add parameters as key=value pairs
	for key, value := range params {
		args = append(args, fmt.Sprintf("%s=%v", key, value))
	}

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

// ExecuteJPAKE runs JPAKE authentication
func (r *JarRunner) ExecuteJPAKE(pairingCode string) (string, error) {
	cmd := exec.Command(r.javaCmd, "-jar", r.jarPath, "jpake", pairingCode)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("JAR JPAKE failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
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
