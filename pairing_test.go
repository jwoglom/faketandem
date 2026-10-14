package main

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jwoglom/faketandem/pkg/api"
	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/state"
)

type pairingTransport struct {
	connectionHandler bluetooth.ConnectionHandler
	disconnects       int
}

func (*pairingTransport) SetWriteHandler(bluetooth.WriteHandler) {}
func (*pairingTransport) SetReadHandler(bluetooth.ReadHandler)   {}
func (p *pairingTransport) SetConnectionHandler(h bluetooth.ConnectionHandler) {
	p.connectionHandler = h
}
func (*pairingTransport) SetCharacteristicData(bluetooth.CharacteristicType, []byte) {}
func (*pairingTransport) Notify(bluetooth.CharacteristicType, []byte) error          { return nil }
func (*pairingTransport) IsConnected() bool                                         { return false }
func (p *pairingTransport) ShutdownConnection()                                    { p.disconnects++ }
func (*pairingTransport) SetPairingState(bluetooth.PairingState) error               { return nil }
func (*pairingTransport) GetPairingState() bluetooth.PairingState {
	return bluetooth.PairingStatePairStep2
}

func pairingWebsocket(t *testing.T, server *api.Server) *websocket.Conn {
	t.Helper()
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	return conn
}

func readPairingEvent(t *testing.T, conn *websocket.Conn) api.BleEvent {
	t.Helper()
	var event api.BleEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	return event
}

func TestSavingSamePINPreservesAuthenticationAndKey(t *testing.T) {
	pumpState := state.NewPumpState()
	pumpState.SetAuthenticated([]byte("session-key"))
	pumpState.SetLongTermKey([]byte("long-term-key"))
	transport := &pairingTransport{}
	server := api.New(transport)
	configureWebsocketCommands(server, transport, pumpState, nil)
	conn := pairingWebsocket(t, server)
	if err := conn.WriteJSON(map[string]string{"command": "setPairingCode", "pairingCode": "123456"}); err != nil {
		t.Fatal(err)
	}
	if event := readPairingEvent(t, conn); event.Type != "pairing_progress" || !strings.Contains(event.Message, "saved") {
		t.Fatalf("no useful save confirmation: %+v", event)
	}
	if event := readPairingEvent(t, conn); event.Authenticated == nil || !*event.Authenticated || event.LongTermKey == "" {
		t.Fatalf("saving unchanged PIN erased the pairing: %+v", event)
	}
	if !pumpState.Authenticated() || !bytes.Equal(pumpState.GetLongTermKey(), []byte("long-term-key")) || transport.disconnects != 0 {
		t.Fatal("saving unchanged PIN interrupted the connection")
	}
}

func TestInvalidPINReportsErrorWithoutChangingState(t *testing.T) {
	pumpState := state.NewPumpState()
	transport := &pairingTransport{}
	server := api.New(transport)
	configureWebsocketCommands(server, transport, pumpState, nil)
	conn := pairingWebsocket(t, server)
	for _, code := range []string{"", "12345", "abcdef", "１２３４５６"} {
		if err := conn.WriteJSON(map[string]string{"command": "setPairingCode", "pairingCode": code}); err != nil {
			t.Fatal(err)
		}
		if event := readPairingEvent(t, conn); event.Type != "pairing_error" {
			t.Fatalf("invalid PIN %q did not report error: %+v", code, event)
		}
		if pumpState.GetPairingCode() != "123456" || transport.disconnects != 0 {
			t.Fatal("invalid PIN changed pairing state")
		}
	}
}

func TestDisconnectBroadcastsAuthenticationReset(t *testing.T) {
	pumpState := state.NewPumpState()
	pumpState.SetAuthenticated([]byte("session-key"))
	pumpState.SetLongTermKey([]byte("long-term-key"))
	transport := &pairingTransport{}
	server := api.New(transport)
	router := handler.NewRouter(nil, pumpState, transport, nil, "go", "", "", "", "", "")
	reassembler := protocol.NewReassembler(time.Second)
	defer reassembler.Stop()
	configureConnectionHandlers(transport, server, router, pumpState, nil, reassembler)
	conn := pairingWebsocket(t, server)
	transport.connectionHandler(false)
	_ = readPairingEvent(t, conn) // connection state frame
	if event := readPairingEvent(t, conn); event.Authenticated == nil || *event.Authenticated || event.LongTermKey == "" {
		t.Fatalf("disconnect did not publish reset with cached key preserved: %+v", event)
	}
	if event := readPairingEvent(t, conn); !strings.Contains(event.Message, "disconnected") {
		t.Fatalf("disconnect missing explanation: %+v", event)
	}
}
