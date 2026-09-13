package protocol

import (
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// TestBuildAlarmStatusResponse reproduces a packet captured from a real pump
// (TandemKit's AlarmStatusResponseTests fixture) from nothing but a txId and a
// bitmask.
func TestBuildAlarmStatusResponse(t *testing.T) {
	cases := []struct {
		name    string
		txID    uint8
		bitmask uint64
		want    string
	}{
		{"empty", 3, 0, "000347030800000000000000005721"},
		// PUMP_RESET_ALARM (bit 3) and RESUME_PUMP_ALARM2 (bit 23).
		{"pumpResetAndResume", 12, (1 << 3) | (1 << 23), "000c470c0808008000000000001cbd"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg, err := BuildAlarmStatusResponse(c.txID, c.bitmask)
			if err != nil {
				t.Fatalf("BuildAlarmStatusResponse: %v", err)
			}
			if msg.Characteristic != bluetooth.CharCurrentStatus {
				t.Errorf("characteristic = %s, want CurrentStatus", msg.Characteristic)
			}
			if len(msg.Fragments) != 1 {
				t.Fatalf("expected a single fragment, got %d", len(msg.Fragments))
			}
			if got := hex.EncodeToString(msg.Fragments[0]); got != c.want {
				t.Errorf("fragment = %s, want %s", got, c.want)
			}
		})
	}
}

// TestBuildErrorResponse pins the two-byte cargo and the single-fragment shape
// the driver's ErrorResponse path depends on -- it parses the first fragment
// directly rather than reassembling.
func TestBuildErrorResponse(t *testing.T) {
	msg, err := BuildErrorResponse(9, 60, 3, ErrorResponseOptions{})
	if err != nil {
		t.Fatalf("BuildErrorResponse: %v", err)
	}
	if len(msg.Fragments) != 1 {
		t.Fatalf("expected a single fragment, got %d", len(msg.Fragments))
	}

	fragment := msg.Fragments[0]
	if fragment[2] != OpcodeErrorResponse {
		t.Errorf("opcode = %d, want %d", fragment[2], OpcodeErrorResponse)
	}
	if fragment[4] != 2 {
		t.Errorf("payload length = %d, want 2", fragment[4])
	}
	if fragment[5] != 60 || fragment[6] != 3 {
		t.Errorf("cargo = [%d %d], want [60 3]", fragment[5], fragment[6])
	}
}

// TestBuildErrorResponse_Signed checks the signed variant carries the 24-byte
// trailer pumpX2 declares for this message.
func TestBuildErrorResponse_Signed(t *testing.T) {
	msg, err := BuildErrorResponse(9, 60, 3, ErrorResponseOptions{
		Signed:         true,
		AuthKey:        []byte("0123456789abcdef"),
		TimeSinceReset: 42,
	})
	if err != nil {
		t.Fatalf("BuildErrorResponse: %v", err)
	}
	if got := msg.Fragments[0][4]; got != 2+SignedTrailerLength {
		t.Errorf("payload length = %d, want %d", got, 2+SignedTrailerLength)
	}
}

// TestBuildHistoryLogStreamResponse checks the [count][streamId][record...]
// cargo layout and that malformed records are rejected rather than truncated
// onto the wire.
func TestBuildHistoryLogStreamResponse(t *testing.T) {
	record, err := hex.DecodeString("370071ef951adfc902000d04010000000000cdcc8c3f00000000")
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	msg, err := BuildHistoryLogStreamResponse(5, 1, [][]byte{record})
	if err != nil {
		t.Fatalf("BuildHistoryLogStreamResponse: %v", err)
	}
	if msg.Characteristic != bluetooth.CharHistoryLog {
		t.Errorf("characteristic = %s, want HistoryLog", msg.Characteristic)
	}

	var body []byte
	for _, fragment := range msg.Fragments {
		body = append(body, fragment[2:]...)
	}
	if body[0] != OpcodeHistoryLogStreamResponse {
		t.Errorf("opcode = %#x, want %#x", body[0], OpcodeHistoryLogStreamResponse)
	}
	if int(body[2]) != 2+HistoryLogRecordLength {
		t.Errorf("payload length = %d, want %d", body[2], 2+HistoryLogRecordLength)
	}
	if body[3] != 1 || body[4] != 1 {
		t.Errorf("[numberOfHistoryLogs streamId] = [%d %d], want [1 1]", body[3], body[4])
	}
	if got := hex.EncodeToString(body[5 : 5+HistoryLogRecordLength]); got != hex.EncodeToString(record) {
		t.Errorf("record = %s, want %s", got, hex.EncodeToString(record))
	}
}

func TestBuildHistoryLogStreamResponse_RejectsBadInput(t *testing.T) {
	if _, err := BuildHistoryLogStreamResponse(1, 1, nil); err == nil {
		t.Error("expected an error for a stream response with no records")
	}
	if _, err := BuildHistoryLogStreamResponse(1, 1, [][]byte{make([]byte, 10)}); err == nil {
		t.Error("expected an error for a record that is not 26 bytes")
	}
}
