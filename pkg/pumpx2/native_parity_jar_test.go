package pumpx2

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/protocol"
)

// These tests prove the native Go encoder in pkg/protocol produces exactly the
// bytes the real pumpX2 cliparser produces. They are the safety net that lets
// handlers bypass cliparser for messages it cannot encode: if the framing, CRC,
// signing or chunking ever drifted, a message built natively would be silently
// rejected by a real driver, and nothing else in the tree would notice.
//
// Gated on FAKETANDEM_TEST_CLIPARSER_JAR, like the other jar tests.
func requireJar(t *testing.T) string {
	t.Helper()
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping cliparser parity test")
	}
	return jarPath
}

func javaCmd() string {
	if cmd := os.Getenv("FAKETANDEM_TEST_JAVA"); cmd != "" {
		return cmd
	}
	return "java"
}

// splitFramedMessage undoes the per-fragment [remaining][txId] framing and
// returns the message body: opcode, txId, payload length, payload, CRC.
func splitFramedMessage(t *testing.T, packetsHex []string) []byte {
	t.Helper()
	var body []byte
	for _, packetHex := range packetsHex {
		packet, err := hex.DecodeString(packetHex)
		if err != nil {
			t.Fatalf("cliparser emitted un-decodable packet %q: %v", packetHex, err)
		}
		if len(packet) < 2 {
			t.Fatalf("cliparser emitted a runt packet %q", packetHex)
		}
		body = append(body, packet[2:]...)
	}
	return body
}

// TestNativeEncoder_MatchesCliparser re-encodes messages cliparser produced,
// from cliparser's own cargo, and asserts the resulting fragments are
// byte-identical -- header, CRC and 18-byte chunking included.
func TestNativeEncoder_MatchesCliparser(t *testing.T) {
	jarPath := requireJar(t)
	bridge := &Bridge{runner: NewJarRunner(jarPath, javaCmd())}

	cases := []struct {
		name    string
		txID    int
		message string
		params  map[string]interface{}
	}{
		{
			name: "ApiVersionResponse", txID: 3, message: "ApiVersionResponse",
			params: map[string]interface{}{"majorVersion": 3, "minorVersion": 5},
		},
		{
			name: "CurrentBasalStatusResponse", txID: 11, message: "CurrentBasalStatusResponse",
			params: map[string]interface{}{
				"profileBasalRate": 1000, "currentBasalRate": 0, "basalModifiedBitmask": 1,
			},
		},
		{
			// Three fragments, so this also pins the descending
			// remaining-packet counter and the 18-byte chunk boundary.
			name: "PumpVersionResponse", txID: 7, message: "PumpVersionResponse",
			params: map[string]interface{}{
				"armSwVer": int64(3628697757), "mspSwVer": 0, "configABits": 0, "configBBits": 0,
				"serialNum": 1226976, "partNum": 1013045, "pumpRev": "0",
				"pcbaSN": 232700077, "pcbaRev": "0", "modelNum": 1004000,
			},
		},
		{
			name: "HistoryLogStatusResponse", txID: 2, message: "HistoryLogStatusResponse",
			params: map[string]interface{}{
				"numEntries": 12, "firstSequenceNum": 1, "lastSequenceNum": 12,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded, err := bridge.EncodeMessage(c.txID, c.message, c.params)
			if err != nil {
				t.Fatalf("cliparser encode failed: %v", err)
			}

			body := splitFramedMessage(t, encoded.Packets)
			if len(body) < 5 {
				t.Fatalf("cliparser body too short: %x", body)
			}
			payloadLen := int(body[2])
			cargo := body[3 : 3+payloadLen]

			native, err := protocol.EncodeMessage(protocol.MessageSpec{
				Opcode: body[0],
				TxID:   body[1],
				Cargo:  cargo,
			})
			if err != nil {
				t.Fatalf("native encode failed: %v", err)
			}

			got := make([]string, 0, len(native))
			for _, fragment := range native {
				got = append(got, hex.EncodeToString(fragment))
			}
			if strings.Join(got, " ") != strings.Join(encoded.Packets, " ") {
				t.Errorf("native encoding differs from cliparser:\n native:    %v\n cliparser: %v",
					got, encoded.Packets)
			}
		})
	}
}

// TestNativeEncoder_MatchesCliparserSigned covers the 24-byte signed trailer:
// the little-endian timeSinceReset and the HMAC-SHA1 over everything before it.
//
// The jar is invoked directly rather than through JarRunner because the signing
// inputs only reach cliparser through the environment. Note cliparser uses the
// raw bytes of the PUMP_AUTHENTICATION_KEY string as the HMAC key rather than
// decoding it as hex, so the native side is given the same ASCII bytes.
func TestNativeEncoder_MatchesCliparserSigned(t *testing.T) {
	jarPath := requireJar(t)

	const authKey = "00112233445566778899aabbccddeeff"
	const timeSinceReset = 1000

	cmd := exec.Command(javaCmd(), "-jar", jarPath, "encode", "5", "SuspendPumpingResponse", `{"status":0}`)
	cmd.Env = append(os.Environ(),
		"PUMP_AUTHENTICATION_KEY="+authKey,
		"PUMP_TIME_SINCE_RESET=1000",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("cliparser encode failed: %v\n%s", err, stderr.String())
	}

	packets := packetsFromEncodeOutput(t, stdout.String())
	body := splitFramedMessage(t, packets)
	if len(body) < 3 {
		t.Fatalf("cliparser body too short: %x", body)
	}
	// The payload length includes the 24-byte trailer, so the cargo is what is
	// left after taking it off.
	payloadLen := int(body[2])
	if payloadLen != 1+protocol.SignedTrailerLength {
		t.Fatalf("expected a signed payload of %d bytes, got %d", 1+protocol.SignedTrailerLength, payloadLen)
	}
	cargo := body[3 : 3+payloadLen-protocol.SignedTrailerLength]

	native, err := protocol.EncodeMessage(protocol.MessageSpec{
		Opcode:         body[0],
		TxID:           body[1],
		Cargo:          cargo,
		Signed:         true,
		AuthKey:        []byte(authKey),
		TimeSinceReset: timeSinceReset,
	})
	if err != nil {
		t.Fatalf("native encode failed: %v", err)
	}

	got := make([]string, 0, len(native))
	for _, fragment := range native {
		got = append(got, hex.EncodeToString(fragment))
	}
	if strings.Join(got, " ") != strings.Join(packets, " ") {
		t.Errorf("native signed encoding differs from cliparser:\n native:    %v\n cliparser: %v",
			got, packets)
	}
}

// packetsFromEncodeOutput pulls the "packets" array out of the JSON line
// cliparser's encode command prints.
func packetsFromEncodeOutput(t *testing.T, output string) []string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		start := strings.Index(line, `"packets":[`)
		if start < 0 {
			continue
		}
		rest := line[start+len(`"packets":[`):]
		end := strings.Index(rest, "]")
		if end < 0 {
			continue
		}
		var packets []string
		for _, token := range strings.Split(rest[:end], ",") {
			packets = append(packets, strings.Trim(strings.TrimSpace(token), `"`))
		}
		return packets
	}
	t.Fatalf("no packets found in cliparser output: %s", output)
	return nil
}

// The three tests below feed the messages only the native encoder can build
// back into the real pumpX2 parser -- the code a driver runs -- and check it
// decodes them as the intended message.

// TestNativeErrorResponse_ParsesBackThroughCliparser covers ErrorResponse,
// which cliparser cannot encode at all ("Unknown message name").
func TestNativeErrorResponse_ParsesBackThroughCliparser(t *testing.T) {
	jarPath := requireJar(t)
	runner := NewJarRunner(jarPath, javaCmd())

	msg, err := protocol.BuildErrorResponse(9, 60, 3, protocol.ErrorResponseOptions{})
	if err != nil {
		t.Fatalf("BuildErrorResponse: %v", err)
	}
	output, err := runner.Parse("CURRENT_STATUS", msg.PacketsHex())
	if err != nil {
		t.Fatalf("cliparser parse failed: %v", err)
	}
	if !strings.Contains(output, "requestCodeId=60") || !strings.Contains(output, "errorCodeId=3") {
		t.Errorf("cliparser did not decode the ErrorResponse fields: %s", output)
	}
}

// TestNativeAlarmStatusResponse_ParsesBackThroughCliparser covers
// AlarmStatusResponse, whose enum varargs constructor cliparser cannot call.
func TestNativeAlarmStatusResponse_ParsesBackThroughCliparser(t *testing.T) {
	jarPath := requireJar(t)
	runner := NewJarRunner(jarPath, javaCmd())

	msg, err := protocol.BuildAlarmStatusResponse(12, (1<<3)|(1<<23))
	if err != nil {
		t.Fatalf("BuildAlarmStatusResponse: %v", err)
	}
	// This is a packet captured from a real pump, which the encoder reproduces
	// byte-for-byte (see the golden tests in pkg/protocol).
	if got := msg.PacketsHex()[0]; got != "000c470c0808008000000000001cbd" {
		t.Fatalf("AlarmStatusResponse = %s, does not match the captured packet", got)
	}

	output, err := runner.Parse("CURRENT_STATUS", msg.PacketsHex())
	if err != nil {
		t.Fatalf("cliparser parse failed: %v", err)
	}
	if !strings.Contains(output, "PUMP_RESET_ALARM") || !strings.Contains(output, "RESUME_PUMP_ALARM2") {
		t.Errorf("cliparser did not decode the alarm bitmask: %s", output)
	}
}

// TestNativeHistoryLogStreamResponse_ParsesBackThroughCliparser covers
// HistoryLogStreamResponse, whose cliparser encode throws ClassCastException.
func TestNativeHistoryLogStreamResponse_ParsesBackThroughCliparser(t *testing.T) {
	jarPath := requireJar(t)
	runner := NewJarRunner(jarPath, javaCmd())

	record, err := hex.DecodeString("370071ef951adfc902000d04010000000000cdcc8c3f00000000")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	// Built with a 40-byte chunk so the whole message fits one fragment. The
	// cliparser CLI's stream branch starts a fresh PacketArrayList per
	// argument, so it cannot reassemble a multi-fragment stream message -- a
	// limitation of the command-line tool, not of the parser a driver uses
	// (TandemKit's handleHistoryLogStreamPacket accumulates across fragments,
	// which the reassembler test in pkg/handler covers).
	cargo := append([]byte{1, 1}, record...)
	fragments, err := protocol.EncodeMessage(protocol.MessageSpec{
		Opcode:          protocol.OpcodeHistoryLogStreamResponse,
		TxID:            5,
		Cargo:           cargo,
		MaxChunkPayload: protocol.ControlMaxChunkPayload,
	})
	if err != nil {
		t.Fatalf("native encode failed: %v", err)
	}
	packetsHex := make([]string, 0, len(fragments))
	for _, fragment := range fragments {
		packetsHex = append(packetsHex, hex.EncodeToString(fragment))
	}

	output, err := runner.Parse("HISTORY_LOG", packetsHex)
	if err != nil {
		t.Fatalf("cliparser parse failed: %v", err)
	}
	for _, want := range []string{
		"HistoryLogStreamResponse", "BolusActivatedHistoryLog",
		"bolusId=1037", "pumpTimeSec=446033777", "sequenceNum=182751",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("cliparser output missing %q: %s", want, output)
		}
	}
}
