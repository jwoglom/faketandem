//nolint:revive // api is a standard package name for API servers
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// fakeTransport is the minimum of bluetooth.Transport the websocket path
// touches: it only ever asks whether a central is connected.
type fakeTransport struct{ connected bool }

func (f *fakeTransport) SetWriteHandler(bluetooth.WriteHandler)                     {}
func (f *fakeTransport) SetReadHandler(bluetooth.ReadHandler)                       {}
func (f *fakeTransport) SetConnectionHandler(bluetooth.ConnectionHandler)           {}
func (f *fakeTransport) SetCharacteristicData(bluetooth.CharacteristicType, []byte) {}
func (f *fakeTransport) Notify(bluetooth.CharacteristicType, []byte) error          { return nil }
func (f *fakeTransport) IsConnected() bool                                          { return f.connected }
func (f *fakeTransport) ShutdownConnection()                                        {}
func (f *fakeTransport) SetPairingState(bluetooth.PairingState) error               { return nil }
func (f *fakeTransport) GetPairingState() bluetooth.PairingState {
	return bluetooth.PairingStateNotDiscoverable
}

// wsMux serves only the websocket endpoint, which is all these tests drive.
func wsMux(s *Server) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/ws", s)
	return mux
}

// dialWS opens a websocket client against srv and drains the initial state
// frame the server sends on connect.
func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial %s: %v", url, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("did not receive the initial state frame: %v", err)
	}
	return conn
}

// readEvent reads one frame and decodes it as a BleEvent.
func readEvent(t *testing.T, conn *websocket.Conn) BleEvent {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}
	var event BleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to decode event %s: %v", data, err)
	}
	return event
}

// TestSurvivesAnotherClientDisconnecting is the regression this multi-client
// rework exists for: the reader's defer used to clear the single shared
// connection whenever ANY socket closed, so closing a second, short-lived
// client silently killed the first client's event stream. Nothing reported an
// error -- events simply stopped arriving.
func TestSurvivesAnotherClientDisconnecting(t *testing.T) {
	server := New(&fakeTransport{})
	srv := httptest.NewServer(wsMux(server))
	defer srv.Close()

	first := dialWS(t, srv)

	second := dialWS(t, srv)
	if err := second.Close(); err != nil {
		t.Fatalf("failed to close the second client: %v", err)
	}
	// Let the server's reader goroutine observe the close before we test that
	// it removed only that client.
	waitForClients(t, server, 1)

	server.SendEvent(BleEvent{Type: "notify", Message: "after the other client left"})

	event := readEvent(t, first)
	if event.Message != "after the other client left" {
		t.Errorf("first client received %+v, want the event sent after the second closed", event)
	}
}

// TestBroadcastsToEveryClient checks the other half: an event reaches all
// connected clients, not just the most recent one.
func TestBroadcastsToEveryClient(t *testing.T) {
	server := New(&fakeTransport{})
	srv := httptest.NewServer(wsMux(server))
	defer srv.Close()

	first := dialWS(t, srv)
	second := dialWS(t, srv)

	server.SendEvent(BleEvent{Type: "connected", Message: "hello both"})

	for i, conn := range []*websocket.Conn{first, second} {
		if event := readEvent(t, conn); event.Message != "hello both" {
			t.Errorf("client %d received %+v, want the broadcast event", i, event)
		}
	}
}

// TestGetStateRepliesToTheRequester checks that a command's reply goes back to
// the socket that sent it.
func TestGetStateRepliesToTheRequester(t *testing.T) {
	server := New(&fakeTransport{connected: true})
	srv := httptest.NewServer(wsMux(server))
	defer srv.Close()

	first := dialWS(t, srv)
	second := dialWS(t, srv)

	if err := second.WriteMessage(websocket.TextMessage, []byte(`{"command":"getState"}`)); err != nil {
		t.Fatalf("failed to send getState: %v", err)
	}

	if err := second.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	_, data, err := second.ReadMessage()
	if err != nil {
		t.Fatalf("requester did not get a reply: %v", err)
	}
	var state PumpState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("failed to decode state %s: %v", data, err)
	}
	if !state.Connected {
		t.Errorf("state reply = %+v, want connected", state)
	}

	// The other client must not have been sent the reply. A broadcast would
	// land here; with nothing pending, the read times out instead.
	if err := first.SetReadDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	if _, unexpected, err := first.ReadMessage(); err == nil {
		t.Errorf("the other client received the command reply %s; it should go only to the requester", unexpected)
	}
}

// waitForClients blocks until the server holds exactly want clients.
func waitForClients(t *testing.T, s *Server, want int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		s.mtx.Lock()
		got := len(s.clients)
		s.mtx.Unlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server has %d websocket clients, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
