package pumpx2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// parseEnv returns the environment for a cliparser "parse" subprocess. When
// btChar is non-empty it sets PUMPX2_CHARACTERISTIC, which cliparser's
// CharacteristicGuesser reads to disambiguate an opcode that maps to more
// than one characteristic (Characteristic.valueOf(...), so btChar must be one
// of pumpX2's Characteristic enum constant names, e.g. "CURRENT_STATUS") --
// see CharacteristicType.ToBtChar. Without this, cliparser falls back to a
// fixed CONTROL > AUTHORIZATION > CURRENT_STATUS precedence that can resolve
// to the wrong message class (and wrong expected cargo size) for opcodes
// shared across characteristics.
func parseEnv(btChar string) []string {
	if btChar == "" {
		return nil
	}
	return append(os.Environ(), "PUMPX2_CHARACTERISTIC="+btChar)
}

// Runner is an interface for executing cliparser commands
type Runner interface {
	// Parse decodes a message from its raw BLE fragments. rawPacketsHex must be the
	// original, unstripped fragment bytes (including framing) in receive order --
	// see PacketBuffer.RawPacketsHex for why the stripped/concatenated payload
	// alone isn't enough.
	Parse(btChar string, rawPacketsHex []string) (string, error)
	Encode(txID int, messageName string, params map[string]interface{}) (string, error)
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

// Parse parses a message using gradle cliparser. btChar identifies the
// characteristic the raw fragments were received on -- see parseEnv.
func (r *GradleRunner) Parse(btChar string, rawPacketsHex []string) (string, error) {
	// The cliparser "parse" command expects each raw BLE fragment (including its
	// framing bytes) as its own whitespace-delimited token; see
	// Main.splitRawHexPackets in pumpX2's cliparser module.
	hexValue := strings.Join(rawPacketsHex, " ")

	// Execute: ./gradlew cliparser -q --console=plain --args="parse <fragments>"
	gradlePath := filepath.Join(r.pumpX2Path, r.gradleCmd)
	cmd := exec.Command(gradlePath, "cliparser", "-q", "--console=plain", "--args=parse "+hexValue)
	cmd.Dir = r.pumpX2Path
	cmd.Env = parseEnv(btChar)

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

// Parse parses a message using JAR cliparser. btChar identifies the
// characteristic the raw fragments were received on -- see parseEnv.
func (r *JarRunner) Parse(btChar string, rawPacketsHex []string) (string, error) {
	// The cliparser "parse" command expects each raw BLE fragment (including its
	// framing bytes) as its own whitespace-delimited token; see
	// Main.splitRawHexPackets in pumpX2's cliparser module.
	hexValue := strings.Join(rawPacketsHex, " ")
	args := []string{"-jar", r.jarPath, "parse", hexValue}

	cmd := exec.Command(r.javaCmd, args...)
	cmd.Env = parseEnv(btChar)

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
