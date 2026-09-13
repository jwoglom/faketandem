package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// TestHistoryLogRequest_AcksAndStreams drives the full HistoryLogRequest
// exchange as a driver would: a request carrying pumpX2's real field names
// (startLog / numberOfLogs), an ack that must come back on CURRENT_STATUS
// rather than HISTORY_LOG, and one HistoryLogStreamResponse per record on
// HISTORY_LOG. The ack is parsed back through the real cliparser so the
// streamId a driver would read is checked, not just the bytes we meant to send.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.
func TestHistoryLogRequest_AcksAndStreams(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	for i := 0; i < 5; i++ {
		pumpState.AppendHistory(state.HistoryEvent{
			TypeID:   state.HistoryPumpingSuspended,
			Name:     "PumpingSuspended",
			PumpTime: uint32(446_000_000 + i),
			Fields:   map[string]interface{}{"insulinAmount": 150 - i},
		})
	}

	handler := NewHistoryLogHandler(bridge)

	// Ask for the newest three: the pump streams backward from startLog, so
	// this is sequences 3, 4 and 5.
	req := &pumpx2.ParsedMessage{
		TxID:        7,
		MessageType: "HistoryLogRequest",
		Cargo:       map[string]interface{}{"startLog": 5, "numberOfLogs": 3},
	}

	resp, err := handler.HandleMessage(req, pumpState)
	if err != nil {
		t.Fatalf("HistoryLogRequest handler failed: %v", err)
	}
	if resp.Characteristic != bluetooth.CharCurrentStatus {
		t.Errorf("ack characteristic = %s, want CurrentStatus (the driver funnels everything on "+
			"HistoryLog into its stream accumulator, so an ack sent there is never delivered as one)",
			resp.Characteristic)
	}

	parsed, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("failed to parse the HistoryLogResponse back: %v", err)
	}
	if parsed.MessageType != "HistoryLogResponse" {
		t.Fatalf("ack parsed back as %q", parsed.MessageType)
	}
	streamID, ok := cargoInt(parsed, "streamId")
	if !ok || streamID == 0 {
		t.Errorf("HistoryLogResponse.streamId = %v (present: %v), want a non-zero id", streamID, ok)
	}

	if len(resp.NativeNotifications) != 3 {
		t.Fatalf("expected 3 stream messages, got %d", len(resp.NativeNotifications))
	}

	// Newest first, and every stream message must be on HISTORY_LOG carrying
	// the same stream id as the ack.
	wantSequences := []uint32{5, 4, 3}
	for i, msg := range resp.NativeNotifications {
		if msg.Characteristic != bluetooth.CharHistoryLog {
			t.Errorf("stream message %d on %s, want HistoryLog", i, msg.Characteristic)
		}

		record := streamRecord(t, msg.Fragments)
		if got := uint32(record[6]) | uint32(record[7])<<8 | uint32(record[8])<<16 | uint32(record[9])<<24; got != wantSequences[i] {
			t.Errorf("stream message %d sequence = %d, want %d", i, got, wantSequences[i])
		}
		if got := uint16(record[0]) | uint16(record[1])<<8; got != state.HistoryPumpingSuspended {
			t.Errorf("stream message %d typeId = %d, want %d", i, got, state.HistoryPumpingSuspended)
		}
	}
}

// TestHistoryLogRequest_ClampsToStoredRange checks that a request reaching below
// sequence 1 (the first sequence the log ever allocates) returns only what
// exists instead of inventing records.
func TestHistoryLogRequest_ClampsToStoredRange(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	pumpState.AppendHistory(state.HistoryEvent{TypeID: state.HistoryPumpingResumed, Name: "PumpingResumed"})

	handler := NewHistoryLogHandler(bridge)
	resp, err := handler.HandleMessage(&pumpx2.ParsedMessage{
		TxID:        1,
		MessageType: "HistoryLogRequest",
		Cargo:       map[string]interface{}{"startLog": 1, "numberOfLogs": 255},
	}, pumpState)
	if err != nil {
		t.Fatalf("HistoryLogRequest handler failed: %v", err)
	}
	if len(resp.NativeNotifications) != 1 {
		t.Errorf("expected 1 stream message for a 1-entry log, got %d", len(resp.NativeNotifications))
	}
}

// streamRecord reassembles a stream message's fragments and returns the single
// 26-byte history record it carries.
func streamRecord(t *testing.T, fragments [][]byte) []byte {
	t.Helper()

	var body []byte
	for _, fragment := range fragments {
		body = append(body, fragment[2:]...)
	}
	// [opcode][txId][len][numberOfLogs][streamId][record...][crc][crc]
	if len(body) < 5+26 {
		t.Fatalf("stream message too short: %x", body)
	}
	return body[5 : 5+26]
}
