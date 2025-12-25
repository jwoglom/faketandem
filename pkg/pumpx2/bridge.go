package pumpx2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwoglom/faketandem/pkg/bluetooth"

	log "github.com/sirupsen/logrus"
)

// Bridge provides an interface to the pumpX2 cliparser
type Bridge struct {
	runner         Runner
	mode           string
	authKey        string
	pairingCode    string
	timeSinceReset uint32
}

// NewBridge creates a new pumpX2 cliparser bridge
func NewBridge(pumpX2Path, mode, gradleCmd, javaCmd string) (*Bridge, error) {
	var runner Runner

	if mode == "gradle" {
		log.Info("Using gradle mode for cliparser")
		runner = NewGradleRunner(pumpX2Path, gradleCmd)
	} else {
		log.Info("Using JAR mode for cliparser")
		// Build/find the cliparser JAR
		jarPath, err := BuildCliParserJAR(pumpX2Path, gradleCmd)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cliparser JAR: %w", err)
		}
		log.Infof("Using cliparser JAR: %s", jarPath)
		runner = NewJarRunner(jarPath, javaCmd)
	}

	return &Bridge{
		runner:         runner,
		mode:           mode,
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

// ParseMessage parses a hex message into a structured format
func (b *Bridge) ParseMessage(charType bluetooth.CharacteristicType, hexData string) (*ParsedMessage, error) {
	btChar := charType.ToBtChar()
	output, err := b.runner.Parse(btChar, hexData)
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
	output, err := b.runner.Encode(txID, messageName, params)
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

// ExecuteJPAKE runs the JPAKE authentication flow with interactive response handling
// Note: This is for when acting as a CLIENT connecting to a pump.
// For pump simulator operation, use individual encode/decode for each JPAKE round instead.
func (b *Bridge) ExecuteJPAKE(pairingCode string, responseProvider JPAKEResponseProvider) (string, error) {
	output, err := b.runner.ExecuteJPAKE(pairingCode, responseProvider)
	if err != nil {
		return "", fmt.Errorf("JPAKE execution failed: %w", err)
	}

	return output, nil
}

// ListAllCommands returns all available request opcodes
func (b *Bridge) ListAllCommands() ([]string, error) {
	output, err := b.runner.ListAllCommands()
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
