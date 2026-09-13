package handler

import (
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
)

// recordingTransport captures every notification the router emits, so tests can
// assert on the exact bytes that would reach a driver.
type recordingTransport struct {
	notifications []recordedNotification
}

type recordedNotification struct {
	charType bluetooth.CharacteristicType
	data     []byte
}

func (r *recordingTransport) SetWriteHandler(bluetooth.WriteHandler)           {}
func (r *recordingTransport) SetReadHandler(bluetooth.ReadHandler)             {}
func (r *recordingTransport) SetConnectionHandler(bluetooth.ConnectionHandler) {}
func (r *recordingTransport) SetCharacteristicData(bluetooth.CharacteristicType, []byte) {
}
func (r *recordingTransport) IsConnected() bool   { return true }
func (r *recordingTransport) ShutdownConnection() {}
func (r *recordingTransport) GetPairingState() bluetooth.PairingState {
	return bluetooth.PairingStateNotDiscoverable
}
func (r *recordingTransport) SetPairingState(bluetooth.PairingState) error { return nil }

func (r *recordingTransport) Notify(charType bluetooth.CharacteristicType, data []byte) error {
	stored := make([]byte, len(data))
	copy(stored, data)
	r.notifications = append(r.notifications, recordedNotification{charType: charType, data: stored})
	return nil
}

// TestSendErrorResponse checks the fault-injection entry point: a single
// well-formed ErrorResponse fragment carrying the rejected request's opcode and
// the error code, on the characteristic asked for.
func TestSendErrorResponse(t *testing.T) {
	transport := &recordingTransport{}
	router := &Router{ble: transport}

	// 60 is HistoryLogRequest's opcode; 3 is pumpX2's TRANSACTION_ID_MISMATCH.
	if err := router.SendErrorResponse(bluetooth.CharCurrentStatus, 9, 60, 3); err != nil {
		t.Fatalf("SendErrorResponse: %v", err)
	}

	if len(transport.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(transport.notifications))
	}
	notification := transport.notifications[0]
	if notification.charType != bluetooth.CharCurrentStatus {
		t.Errorf("notified on %s, want CurrentStatus", notification.charType)
	}

	// [remaining][txId][opcode][txId][len][requestCodeId][errorCodeId][crc][crc]
	if got := hex.EncodeToString(notification.data); got != "00094d09023c0392cc" {
		t.Errorf("ErrorResponse fragment = %s, want 00094d09023c0392cc", got)
	}
}

// TestSendNative_SendsEveryFragmentInOrder guards the multi-fragment path: the
// router must notify each fragment exactly once, in order, on the message's own
// characteristic.
func TestSendNative_SendsEveryFragmentInOrder(t *testing.T) {
	transport := &recordingTransport{}
	router := &Router{ble: transport}

	record := make([]byte, protocol.HistoryLogRecordLength)
	msg, err := protocol.BuildHistoryLogStreamResponse(4, 2, [][]byte{record})
	if err != nil {
		t.Fatalf("BuildHistoryLogStreamResponse: %v", err)
	}
	if len(msg.Fragments) < 2 {
		t.Fatalf("expected a multi-fragment message, got %d fragment(s)", len(msg.Fragments))
	}

	if err := router.SendNative(msg); err != nil {
		t.Fatalf("SendNative: %v", err)
	}

	if len(transport.notifications) != len(msg.Fragments) {
		t.Fatalf("sent %d notifications for %d fragments", len(transport.notifications), len(msg.Fragments))
	}
	for i, notification := range transport.notifications {
		if notification.charType != bluetooth.CharHistoryLog {
			t.Errorf("fragment %d notified on %s, want HistoryLog", i, notification.charType)
		}
		if hex.EncodeToString(notification.data) != hex.EncodeToString(msg.Fragments[i]) {
			t.Errorf("fragment %d = %x, want %x", i, notification.data, msg.Fragments[i])
		}
	}
}
