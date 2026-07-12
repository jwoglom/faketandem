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

// NewBridge creates a new pumpX2 cliparser bridge. If jarPath is non-empty, it is
// used directly as the cliparser JAR, skipping gradle entirely regardless of mode.
func NewBridge(pumpX2Path, mode, gradleCmd, javaCmd, jarPath string) (*Bridge, error) {
	var runner Runner

	if mode == "gradle" {
		log.Info("Using gradle mode for cliparser")
		runner = NewGradleRunner(pumpX2Path, gradleCmd)
	} else if jarPath != "" {
		log.Infof("Using prebuilt cliparser JAR: %s", jarPath)
		runner = NewJarRunner(jarPath, javaCmd)
	} else {
		log.Info("Using JAR mode for cliparser")
		// Build/find the cliparser JAR
		builtJarPath, err := BuildCliParserJAR(pumpX2Path, gradleCmd)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cliparser JAR: %w", err)
		}
		log.Infof("Using cliparser JAR: %s", builtJarPath)
		runner = NewJarRunner(builtJarPath, javaCmd)
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

// ParseMessage parses a message from its raw BLE fragments into a structured
// format. rawPacketsHex must be the original, unstripped fragment bytes
// (including framing) in receive order -- see PacketBuffer.RawPacketsHex.
func (b *Bridge) ParseMessage(charType bluetooth.CharacteristicType, rawPacketsHex []string) (*ParsedMessage, error) {
	btChar := charType.ToBtChar()
	output, err := b.runner.Parse(btChar, rawPacketsHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	opcode, txID, ok := opcodeAndTxIDFromFirstFragment(rawPacketsHex)
	if !ok {
		return nil, fmt.Errorf("failed to extract opcode/txId from raw fragments")
	}
	messageName, cargo := parseCliparserOutput(output)

	msg := &ParsedMessage{
		Opcode:        opcode,
		MessageType:   messageName,
		TxID:          txID,
		Cargo:         cargo,
		Raw:           strings.Join(rawPacketsHex, ""),
		IsValid:       messageName != "",
		RawPacketsHex: rawPacketsHex,
	}

	return msg, nil
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

