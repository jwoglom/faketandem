package controlstream_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// testCapture is pumpX2's MessageTester.test for a one-packet capture of a
// stream message whose opcode CONTROL has too. cliparser's parse, which
// parity.TestMessage runs, takes such an opcode as CONTROL's, so this parses
// the capture's payload, signature and all, as BTResponseParser hands it to
// the message; under IGNORE_INVALID_HMAC pumpX2 does not check the signature
// either.
func testCapture(t *testing.T, rawHex string, txID byte, expected messages.Message) messages.Message {
	t.Helper()
	raw, err := hex.DecodeString(rawHex)
	if err != nil {
		t.Fatal(err)
	}
	class := messages.ClassOf(expected)
	if len(raw) < 7 || raw[0] != 0 || raw[1] != txID || int8(raw[2]) != int8(class.Props.OpCode) || raw[3] != txID || len(raw) != 5+int(raw[4])+2 {
		t.Fatalf("%s is not one packet of a %s in transaction %d", rawHex, class.Name, txID)
	}
	if crc := helpers.CRC16(raw[2 : len(raw)-2]); !bytes.Equal(crc, raw[len(raw)-2:]) {
		t.Fatalf("CRC %x, want %x", raw[len(raw)-2:], crc)
	}
	parsed, err := messages.Parse(class, raw[5:len(raw)-2])
	if err != nil {
		t.Fatalf("%s does not parse: %v", rawHex, err)
	}
	if got, want := messages.ToString(parsed), messages.ToString(expected); got != want {
		t.Errorf("parsed:\n  %s\nwant:\n  %s", got, want)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	return parsed
}

func TestEnterChangeCartridgeModeStateStreamResponse(t *testing.T) {
	expected := controlstream.NewEnterChangeCartridgeModeStateStreamResponse(2)
	parsed := testCapture(t, "0083e18319026dc840203e1a10e4806b4079adf00d1cfe2894949c7d58333915", 0x83,
		expected).(*controlstream.EnterChangeCartridgeModeStateStreamResponse)
	if parsed.State == nil || *parsed.State != controlstream.ChangeCartridgeStateReadyToChange {
		t.Errorf("state %v, want READY_TO_CHANGE", parsed.State)
	}
}

func TestEnterChangeCartridgeModeStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "EnterChangeCartridgeModeStateStreamResponse")
}
