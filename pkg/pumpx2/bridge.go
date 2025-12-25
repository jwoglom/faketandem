package pumpx2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Bridge provides an interface to the pumpX2 cliparser
type Bridge struct {
	jarPath      string
	javaCmd      string
	authKey      string
	pairingCode  string
	timeSinceReset uint32
}

// NewBridge creates a new pumpX2 cliparser bridge
func NewBridge(pumpX2Path, gradleCmd, javaCmd string) (*Bridge, error) {
	// Build/find the cliparser JAR
	jarPath, err := BuildCliParserJAR(pumpX2Path, gradleCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cliparser: %w", err)
	}

	log.Infof("Initialized pumpX2 bridge with JAR: %s", jarPath)

	return &Bridge{
		jarPath:        jarPath,
		javaCmd:        javaCmd,
		timeSinceReset: 0, // Will be updated as needed
	}, nil
}

// SetAuthenticationKey sets the authentication key for signing messages
func (b *Bridge) SetAuthenticationKey(key string) {
	b.authKey = key
	log.Debug("Updated authentication key")
}

// SetPairingCode sets the pairing code for authentication
func (b *Bridge) SetPairingCode(code string) {
	b.pairingCode = code
	log.Debugf("Updated pairing code: %s", code)
}

// SetTimeSinceReset sets the pump time since reset
func (b *Bridge) SetTimeSinceReset(seconds uint32) {
	b.timeSinceReset = seconds
}

// execCliParser executes a cliparser command and returns the output
func (b *Bridge) execCliParser(args ...string) (string, error) {
	cmdArgs := []string{"-jar", b.jarPath}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(b.javaCmd, cmdArgs...)

	// Set environment variables for authentication
	env := []string{}
	if b.authKey != "" {
		env = append(env, fmt.Sprintf("PUMP_AUTHENTICATION_KEY=%s", b.authKey))
	}
	if b.pairingCode != "" {
		env = append(env, fmt.Sprintf("PUMP_PAIRING_CODE=%s", b.pairingCode))
	}
	if b.timeSinceReset > 0 {
		env = append(env, fmt.Sprintf("PUMP_TIME_SINCE_RESET=%d", b.timeSinceReset))
	}
	cmd.Env = append(cmd.Env, env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Tracef("Executing cliparser: %s %s", b.javaCmd, strings.Join(cmdArgs, " "))

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cliparser execution failed: %w\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	log.Tracef("cliparser output: %s", output)

	return output, nil
}

// ParseOpcode identifies the opcode and message type from hex data
func (b *Bridge) ParseOpcode(hexData string) (*OpcodeInfo, error) {
	output, err := b.execCliParser("opcode", hexData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse opcode: %w", err)
	}

	// The opcode command outputs format like: "Opcode: 65 (0x41) - ApiVersionRequest"
	// We need to parse this
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no output from opcode command")
	}

	// For now, return a simple struct - we can enhance parsing later
	info := &OpcodeInfo{}

	// Try to extract opcode and message type from the output
	// This is a simple parser - enhance as needed
	for _, line := range lines {
		if strings.Contains(line, "Opcode:") {
			// Parse opcode value
			var opcode int
			var msgType string
			_, err := fmt.Sscanf(line, "Opcode: %d", &opcode)
			if err == nil {
				info.Opcode = opcode
			}

			// Extract message type (after the dash)
			if idx := strings.Index(line, "-"); idx >= 0 {
				msgType = strings.TrimSpace(line[idx+1:])
				info.MessageType = msgType

				// Determine direction
				if strings.HasSuffix(msgType, "Request") {
					info.Direction = "request"
				} else if strings.HasSuffix(msgType, "Response") {
					info.Direction = "response"
				}
			}
		}
	}

	if info.MessageType == "" {
		return nil, fmt.Errorf("could not parse message type from output: %s", output)
	}

	return info, nil
}

// ParseMessage parses a hex message into a structured format
func (b *Bridge) ParseMessage(hexData string) (*ParsedMessage, error) {
	output, err := b.execCliParser("parse", hexData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	// The parse command outputs structured information
	// Try to parse as JSON first, otherwise parse text output
	var msg ParsedMessage

	// For now, we'll need to parse the text output
	// The cliparser outputs in a specific format - we need to extract the data
	// Example output might be:
	// Message: ApiVersionRequest
	// TxID: 1
	// Cargo: {...}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	msg.Raw = hexData
	msg.IsValid = true // Assume valid if parsing succeeded

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Message:") {
			msg.MessageType = strings.TrimSpace(strings.TrimPrefix(line, "Message:"))
		} else if strings.HasPrefix(line, "TxID:") || strings.HasPrefix(line, "Transaction ID:") {
			var txid int
			fmt.Sscanf(line, "TxID: %d", &txid)
			msg.TxID = txid
		} else if strings.HasPrefix(line, "Opcode:") {
			var opcode int
			fmt.Sscanf(line, "Opcode: %d", &opcode)
			msg.Opcode = opcode
		}
	}

	// Initialize cargo as empty map
	if msg.Cargo == nil {
		msg.Cargo = make(map[string]interface{})
	}

	return &msg, nil
}

// EncodeMessage builds a message using the specified parameters
func (b *Bridge) EncodeMessage(txID int, messageName string, params map[string]interface{}) (*EncodedMessage, error) {
	// Build the encode command arguments
	args := []string{"encode", fmt.Sprintf("%d", txID), messageName}

	// Add parameters as key=value pairs
	for key, value := range params {
		args = append(args, fmt.Sprintf("%s=%v", key, value))
	}

	output, err := b.execCliParser(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	// Try to parse output as JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// If not JSON, try to parse text output
		return b.parseEncodeTextOutput(output, txID, messageName)
	}

	// Extract encoded message from JSON
	msg := &EncodedMessage{
		MessageType: messageName,
		TxID:        txID,
	}

	// Extract characteristic
	if char, ok := result["characteristic"].(string); ok {
		msg.Characteristic = char
	}

	// Extract packets
	if packets, ok := result["packets"].([]interface{}); ok {
		for _, p := range packets {
			if pStr, ok := p.(string); ok {
				msg.Packets = append(msg.Packets, pStr)
			}
		}
	}

	// Extract opcode
	if opcode, ok := result["opcode"].(float64); ok {
		msg.Opcode = int(opcode)
	}

	return msg, nil
}

// parseEncodeTextOutput parses text output from encode command
func (b *Bridge) parseEncodeTextOutput(output string, txID int, messageName string) (*EncodedMessage, error) {
	msg := &EncodedMessage{
		MessageType: messageName,
		TxID:        txID,
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Characteristic:") {
			msg.Characteristic = strings.TrimSpace(strings.TrimPrefix(line, "Characteristic:"))
		} else if strings.HasPrefix(line, "Packet:") || strings.HasPrefix(line, "Data:") {
			// Extract hex data
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				hexData := strings.TrimSpace(parts[1])
				msg.Packets = append(msg.Packets, hexData)
			}
		}
	}

	if len(msg.Packets) == 0 {
		return nil, fmt.Errorf("no packets found in encode output: %s", output)
	}

	return msg, nil
}

// ExecuteJPAKE runs the JPAKE authentication flow
func (b *Bridge) ExecuteJPAKE(pairingCode string) (string, error) {
	output, err := b.execCliParser("jpake", pairingCode)
	if err != nil {
		return "", fmt.Errorf("JPAKE execution failed: %w", err)
	}

	return output, nil
}

// ListAllCommands returns all available request opcodes
func (b *Bridge) ListAllCommands() ([]string, error) {
	output, err := b.execCliParser("listallcommands")
	if err != nil {
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}

	// Parse output - each line is a command
	lines := strings.Split(strings.TrimSpace(output), "\n")
	commands := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			commands = append(commands, line)
		}
	}

	return commands, nil
}
