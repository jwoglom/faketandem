package handler

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/state"
)

// TestBuildHistoryLogStream_FramingRoundTrip runs the stream messages the
// history handler emits back through the emulator's own packet reassembler --
// the same code path an inbound message takes -- and checks that what comes out
// the other end is a well-formed HistoryLogStreamResponse: right opcode, right
// txId, a payload length that matches, a valid CRC, and the exact record bytes
// that went in.
//
// A single 26-byte record makes a 33-byte message, which is two fragments at
// the protocol's 18-byte chunk size, so this also exercises the multi-packet
// path rather than just a single notification.
func TestBuildHistoryLogStream_FramingRoundTrip(t *testing.T) {
	entries := []state.HistoryLogEntry{
		{
			Sequence: 41, TypeID: state.HistoryPumpingSuspended, Type: "PumpingSuspended",
			PumpTime: 446_000_000,
			Data: map[string]interface{}{
				"preSuspendState": 106, "insulinAmount": 150, "reasonId": 0, "rpaTimeout": 15,
			},
		},
		{
			Sequence: 42, TypeID: state.HistoryPumpingResumed, Type: "PumpingResumed",
			PumpTime: 446_000_600,
			Data:     map[string]interface{}{"preResumeState": 100, "insulinAmount": 149},
		},
	}

	messages, err := buildHistoryLogStream(6, 3, entries)
	if err != nil {
		t.Fatalf("buildHistoryLogStream: %v", err)
	}
	if len(messages) != len(entries) {
		t.Fatalf("expected one message per entry, got %d for %d entries", len(messages), len(entries))
	}

	// The pump streams backward from the requested sequence, so the newest
	// entry goes out first.
	if got := messages[0].Fragments[0]; got[1] != 6 {
		t.Errorf("txId = %d, want 6", got[1])
	}

	reassembler := protocol.NewReassembler(5 * time.Second)
	defer reassembler.Stop()

	for i, msg := range messages {
		// Newest first: message 0 carries the last entry.
		assertStreamMessage(t, reassembler, i, msg, entries[len(entries)-1-i])
	}
}

// assertStreamMessage reassembles one stream message and checks every framing
// field a driver reads, plus the record it carries.
func assertStreamMessage(
	t *testing.T,
	reassembler *protocol.Reassembler,
	index int,
	msg *protocol.NativeMessage,
	wantEntry state.HistoryLogEntry,
) {
	t.Helper()

	if msg.Characteristic != bluetooth.CharHistoryLog {
		t.Errorf("message %d is on %s, want HistoryLog", index, msg.Characteristic)
	}
	if len(msg.Fragments) != 2 {
		t.Errorf("message %d has %d fragments, want 2", index, len(msg.Fragments))
	}

	body := reassembleFragments(t, reassembler, msg.Fragments)

	if body[0] != protocol.OpcodeHistoryLogStreamResponse {
		t.Errorf("message %d opcode = %#x, want %#x", index, body[0], protocol.OpcodeHistoryLogStreamResponse)
	}
	if body[1] != 6 {
		t.Errorf("message %d txId = %d, want 6", index, body[1])
	}

	payloadLen := int(body[2])
	if payloadLen != 2+protocol.HistoryLogRecordLength {
		t.Fatalf("message %d payload length = %d, want %d", index, payloadLen, 2+protocol.HistoryLogRecordLength)
	}
	if len(body) != 3+payloadLen+2 {
		t.Fatalf("message %d is %d bytes, want %d", index, len(body), 3+payloadLen+2)
	}

	crc := protocol.CalculateCRC16(body[:3+payloadLen])
	if hex.EncodeToString(crc) != hex.EncodeToString(body[3+payloadLen:]) {
		t.Errorf("message %d CRC = %x, want %x", index, body[3+payloadLen:], crc)
	}

	cargo := body[3 : 3+payloadLen]
	if cargo[0] != 1 {
		t.Errorf("message %d numberOfHistoryLogs = %d, want 1", index, cargo[0])
	}
	if cargo[1] != 3 {
		t.Errorf("message %d streamId = %d, want 3", index, cargo[1])
	}
	if got, want := hex.EncodeToString(cargo[2:]), hex.EncodeToString(wantEntry.EncodeRecord()); got != want {
		t.Errorf("message %d record =\n  %s\nwant\n  %s", index, got, want)
	}
}

// reassembleFragments feeds fragments through the emulator's own reassembler
// and returns the completed message body.
func reassembleFragments(t *testing.T, reassembler *protocol.Reassembler, fragments [][]byte) []byte {
	t.Helper()

	var body []byte
	for j, fragment := range fragments {
		assembled, _, complete, err := reassembler.AddPacket(bluetooth.CharHistoryLog, fragment)
		if err != nil {
			t.Fatalf("fragment %d: %v", j, err)
		}
		if complete {
			body = assembled
		}
	}
	if body == nil {
		t.Fatal("message never reassembled")
	}
	return body
}

// TestBuildHistoryLogStream_Empty checks that a request whose range matches
// nothing produces no stream messages rather than an error or an empty one --
// an empty HistoryLogStreamResponse is not a thing a pump sends.
func TestBuildHistoryLogStream_Empty(t *testing.T) {
	messages, err := buildHistoryLogStream(1, 1, nil)
	if err != nil {
		t.Fatalf("buildHistoryLogStream: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected no messages, got %d", len(messages))
	}
}
