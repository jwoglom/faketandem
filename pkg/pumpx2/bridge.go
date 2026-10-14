package pumpx2

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"

	log "github.com/sirupsen/logrus"
)

// Codec names: which implementation of the pumpX2 message library parses
// requests and encodes responses.
const (
	// CodecNative is the Go port in pkg/pumpx2/messages. It needs no JVM.
	CodecNative = "native"
	// CodecPumpX2 is pumpX2's own cliparser, run as a jar or through gradle.
	CodecPumpX2 = "pumpx2"
	// CodecCompare answers with the Go port but also runs every message
	// through cliparser and logs any difference, for checking the port against
	// live traffic.
	CodecCompare = "compare"
)

// Bridge parses and encodes pump messages. The zero value uses the Go port;
// NewBridge attaches pumpX2's cliparser.
type Bridge struct {
	runner         Runner
	codec          string
	mode           string
	authKey        string
	pairingCode    string
	timeSinceReset uint32

	mismatchMu sync.Mutex
	mismatches int
}

// NewNativeBridge creates a bridge backed by the Go port alone.
func NewNativeBridge() *Bridge {
	return &Bridge{codec: CodecNative}
}

// NewBridge creates a bridge backed by pumpX2's cliparser (CodecPumpX2), or by
// the Go port checked against it (CodecCompare). If jarPath is non-empty it
// is used directly as the cliparser JAR, skipping gradle entirely regardless
// of mode.
func NewBridge(pumpX2Path, mode, gradleCmd, javaCmd, jarPath string) (*Bridge, error) {
	return NewBridgeWithCodec(CodecPumpX2, pumpX2Path, mode, gradleCmd, javaCmd, jarPath)
}

// NewBridgeWithCodec is NewBridge for an explicit codec. CodecNative ignores
// every other argument.
func NewBridgeWithCodec(codec, pumpX2Path, mode, gradleCmd, javaCmd, jarPath string) (*Bridge, error) {
	if codec == CodecNative {
		return NewNativeBridge(), nil
	}
	if codec != CodecPumpX2 && codec != CodecCompare {
		return nil, fmt.Errorf("unknown message codec %q", codec)
	}

	var runner Runner
	switch {
	case mode == "gradle":
		log.Info("Using gradle mode for cliparser")
		runner = NewGradleRunner(pumpX2Path, gradleCmd)
	case jarPath != "":
		log.Infof("Using prebuilt cliparser JAR: %s", jarPath)
		runner = NewServeRunner(jarPath, javaCmd)
	default:
		log.Info("Using JAR mode for cliparser")
		builtJarPath, err := BuildCliParserJAR(pumpX2Path, gradleCmd)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cliparser JAR: %w", err)
		}
		log.Infof("Using cliparser JAR: %s", builtJarPath)
		runner = NewServeRunner(builtJarPath, javaCmd)
	}

	return &Bridge{runner: runner, codec: codec, mode: mode}, nil
}

// Codec reports which implementation the bridge answers with.
func (b *Bridge) Codec() string {
	if b.runner == nil {
		return CodecNative
	}
	return b.codec
}

// Mismatches counts the messages on which the Go port and cliparser
// disagreed, in CodecCompare.
func (b *Bridge) Mismatches() int {
	b.mismatchMu.Lock()
	defer b.mismatchMu.Unlock()
	return b.mismatches
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

// WarmUp loads cliparser before a central starts its timed authentication
// exchange. Encoding a request has no pump-side effects.
func (b *Bridge) WarmUp() error {
	_, err := b.EncodeMessage(0, "ApiVersionRequest", nil)
	return err
}

// Close releases the persistent cliparser subprocess, when using jar mode.
func (b *Bridge) Close() error {
	if closer, ok := b.runner.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// SetTimeSinceReset sets the pump time since reset
func (b *Bridge) SetTimeSinceReset(seconds uint32) {
	b.timeSinceReset = seconds
}

// ParseMessage parses a message from its raw BLE fragments into a structured
// format. rawPacketsHex must be the original, unstripped fragment bytes
// (including framing) in receive order -- see PacketBuffer.RawPacketsHex.
func (b *Bridge) ParseMessage(charType bluetooth.CharacteristicType, rawPacketsHex []string) (*ParsedMessage, error) {
	switch b.Codec() {
	case CodecNative:
		return parseNative(charType, rawPacketsHex)
	case CodecCompare:
		native, nerr := parseNative(charType, rawPacketsHex)
		jar, jerr := b.parseJar(charType, rawPacketsHex)
		b.compareParsed(rawPacketsHex, native, nerr, jar, jerr)
		return native, nerr
	default:
		return b.parseJar(charType, rawPacketsHex)
	}
}

// parseNative is cliparser's parse done by the Go port. Its result is read
// through the same parseCliparserOutput as the jar's printed output, so a
// handler sees the same fields from either.
//
// Where cliparser prints no message (an exception, a bad CRC, an unknown
// opcode) the jar path reads the exception's text as a message name; the
// port leaves the name empty instead. Either way no handler matches it.
func parseNative(charType bluetooth.CharacteristicType, rawPacketsHex []string) (*ParsedMessage, error) {
	opcode, txID, ok := opcodeAndTxIDFromFirstFragment(rawPacketsHex)
	if !ok {
		return nil, fmt.Errorf("failed to extract opcode/txId from raw fragments")
	}
	msg := &ParsedMessage{
		Opcode:        opcode,
		TxID:          txID,
		Cargo:         map[string]interface{}{},
		Raw:           strings.Join(rawPacketsHex, ""),
		RawPacketsHex: rawPacketsHex,
	}
	m, err := messages.ParseFragments(charType.ToBtChar(), rawPacketsHex, messages.EnvFromOS())
	if err != nil {
		log.Debugf("No message in fragments on %s: %v", charType, err)
		return msg, nil
	}
	msg.MessageType, msg.Cargo = parseCliparserOutput(m.ToString())
	msg.IsValid = msg.MessageType != ""
	return msg, nil
}

func (b *Bridge) parseJar(charType bluetooth.CharacteristicType, rawPacketsHex []string) (*ParsedMessage, error) {
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

	return &ParsedMessage{
		Opcode:        opcode,
		MessageType:   messageName,
		TxID:          txID,
		Cargo:         cargo,
		Raw:           strings.Join(rawPacketsHex, ""),
		IsValid:       messageName != "",
		RawPacketsHex: rawPacketsHex,
	}, nil
}

// EncodeMessage builds a message using the specified parameters
func (b *Bridge) EncodeMessage(txID int, messageName string, params map[string]interface{}) (*EncodedMessage, error) {
	switch b.Codec() {
	case CodecNative:
		return encodeNative(txID, messageName, params)
	case CodecCompare:
		native, nerr := encodeNative(txID, messageName, params)
		jar, jerr := b.encodeJar(txID, messageName, params)
		b.compareEncoded(messageName, params, native, nerr, jar, jerr)
		return native, nerr
	default:
		return b.encodeJar(txID, messageName, params)
	}
}

// encodeNative is cliparser's encode done by the Go port, giving the bridge
// the same EncodedMessage the jar's output does: the characteristic as its
// UUID, and no opcode, which cliparser does not print.
func encodeNative(txID int, messageName string, params map[string]interface{}) (*EncodedMessage, error) {
	encoded, err := messages.Encode(txID, messageName, params, messages.EnvFromOS())
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return &EncodedMessage{
		Characteristic: encoded.CharacteristicUUID,
		Packets:        encoded.Packets,
		MessageType:    messageName,
		TxID:           txID,
	}, nil
}

func (b *Bridge) encodeJar(txID int, messageName string, params map[string]interface{}) (*EncodedMessage, error) {
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

func (b *Bridge) mismatch(format string, args ...interface{}) {
	b.mismatchMu.Lock()
	b.mismatches++
	b.mismatchMu.Unlock()
	log.Errorf("pumpX2 parity mismatch: "+format, args...)
}

// compareParsed holds the port's parse to cliparser's. A message cliparser
// printed no dump for counts as failed on both sides when the port found none
// either.
func (b *Bridge) compareParsed(raw []string, native *ParsedMessage, nerr error, jar *ParsedMessage, jerr error) {
	if nerr != nil || jerr != nil {
		if (nerr == nil) != (jerr == nil) {
			b.mismatch("parse %v: go error %v, cliparser error %v", raw, nerr, jerr)
		}
		return
	}
	// Where cliparser printed an exception the jar path reads its text as the
	// name and the port leaves it empty.
	if !native.IsValid && !messages.IsMessageName(jar.MessageType) {
		return
	}
	if native.MessageType != jar.MessageType || !reflect.DeepEqual(native.Cargo, jar.Cargo) {
		b.mismatch("parse %v:\n  go:        %s %v\n  cliparser: %s %v", raw, native.MessageType, native.Cargo, jar.MessageType, jar.Cargo)
	}
}

// compareEncoded holds the port's encode to cliparser's.
func (b *Bridge) compareEncoded(name string, params map[string]interface{}, native *EncodedMessage, nerr error, jar *EncodedMessage, jerr error) {
	switch {
	case (nerr == nil) != (jerr == nil):
		b.mismatch("encode %s %v: go error %v, cliparser error %v", name, params, nerr, jerr)
	case nerr == nil && (!reflect.DeepEqual(native.Packets, jar.Packets) || native.Characteristic != jar.Characteristic):
		b.mismatch("encode %s %v:\n  go:        %s %v\n  cliparser: %s %v", name, params, native.Characteristic, native.Packets, jar.Characteristic, jar.Packets)
	}
}
