// Package virtualtest is a minimal virtual-GATT central for tests.
//
// The transport's own tests drive it through a client private to that package;
// this one is exported so tests elsewhere in the tree -- notably the fault
// injection tests, which need a router, a pump and a link all at once -- can
// drive a real loopback connection rather than asserting against a mock that
// would not prove anything about what actually reaches a central.
//
// It speaks the protocol documented in docs/virtual-gatt-protocol.md.
package virtualtest

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// DefaultTimeout bounds every read the client makes.
const DefaultTimeout = 3 * time.Second

// Message is one newline-delimited JSON message on the virtual link. Only the
// fields a test client needs are modeled.
type Message struct {
	Type string `json:"type"`
	ID   *int   `json:"id,omitempty"`

	Characteristic string `json:"characteristic,omitempty"`
	Value          string `json:"value,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`

	Error  string `json:"error,omitempty"`
	Reason string `json:"reason,omitempty"`

	ProtocolVersion int    `json:"protocol_version,omitempty"`
	PeripheralID    string `json:"peripheral_id,omitempty"`
	Name            string `json:"name,omitempty"`
}

// Client is a virtual-GATT central driven from a test.
type Client struct {
	t       *testing.T
	conn    net.Conn
	r       *bufio.Reader
	next    int
	timeout time.Duration
}

// Dial connects to a virtual transport listening at addr, reads the hello and
// returns a client. The connection is closed when the test finishes.
func Dial(t *testing.T, addr string) *Client {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("virtualtest: dial %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	c := &Client{t: t, conn: conn, r: bufio.NewReader(conn), timeout: DefaultTimeout}
	if hello := c.Recv(); hello.Type != "hello" {
		t.Fatalf("virtualtest: expected hello, got %q (reason %q)", hello.Type, hello.Reason)
	}
	return c
}

// SetTimeout changes how long reads wait.
func (c *Client) SetTimeout(d time.Duration) { c.timeout = d }

// Send writes a message with a fresh id and returns that id.
func (c *Client) Send(msg Message) int {
	c.t.Helper()

	c.next++
	id := c.next
	msg.ID = &id
	line, err := json.Marshal(msg)
	if err != nil {
		c.t.Fatalf("virtualtest: marshal: %v", err)
	}
	if _, err := c.conn.Write(append(line, '\n')); err != nil {
		c.t.Fatalf("virtualtest: write: %v", err)
	}
	return id
}

// Recv reads the next message, failing the test on timeout.
func (c *Client) Recv() Message {
	c.t.Helper()

	msg, err := c.TryRecv()
	if err != nil {
		c.t.Fatalf("virtualtest: recv: %v", err)
	}
	return msg
}

// TryRecv reads the next message, returning an error on timeout or EOF.
func (c *Client) TryRecv() (Message, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return Message{}, err
	}
	line, err := c.r.ReadString('\n')
	if err != nil {
		return Message{}, err
	}
	var msg Message
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

// Attach completes the attach handshake, which is what makes this client the
// connected central.
func (c *Client) Attach() {
	c.t.Helper()

	id := c.Send(Message{Type: "attach"})
	ack := c.Recv()
	if ack.Type != "attach_ack" || ack.ID == nil || *ack.ID != id {
		c.t.Fatalf("virtualtest: expected attach_ack id=%d, got %+v", id, ack)
	}
}

// Subscribe enables or disables notifications for a characteristic UUID.
func (c *Client) Subscribe(uuid string, enabled bool) {
	c.t.Helper()

	id := c.Send(Message{Type: "subscribe", Characteristic: uuid, Enabled: &enabled})
	ack := c.Recv()
	if ack.Type != "subscribe_ack" || ack.ID == nil || *ack.ID != id {
		c.t.Fatalf("virtualtest: expected subscribe_ack id=%d, got %+v", id, ack)
	}
}

// Write writes hex-encoded bytes to a characteristic and waits for the ack.
func (c *Client) Write(uuid, valueHex string) {
	c.t.Helper()

	id := c.Send(Message{Type: "write", Characteristic: uuid, Value: valueHex})
	ack := c.Recv()
	if ack.Type != "write_ack" || ack.ID == nil || *ack.ID != id {
		c.t.Fatalf("virtualtest: expected write_ack id=%d, got %+v", id, ack)
	}
}

// CollectNotifications reads until want notifications have arrived or the
// wait elapses, returning their hex payloads in arrival order. Messages of
// other types are skipped, except a disconnect, which ends collection (a link
// the peripheral cut is a result, not a failure).
func (c *Client) CollectNotifications(want int, wait time.Duration) []string {
	c.t.Helper()

	var got []string
	deadline := time.Now().Add(wait)
	for len(got) < want && time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		old := c.timeout
		c.timeout = remaining
		msg, err := c.TryRecv()
		c.timeout = old
		if err != nil {
			break
		}
		switch msg.Type {
		case "notify":
			got = append(got, msg.Value)
		case "disconnect":
			return got
		}
	}
	return got
}

// ExpectDisconnect waits for the peripheral to drop the link, either by
// sending a disconnect message or by closing the socket.
func (c *Client) ExpectDisconnect(wait time.Duration) {
	c.t.Helper()

	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		old := c.timeout
		c.timeout = remaining
		msg, err := c.TryRecv()
		c.timeout = old
		if err != nil {
			// EOF or timeout; EOF is the disconnect, a timeout is a failure.
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
				continue
			}
			return
		}
		if msg.Type == "disconnect" {
			return
		}
	}
	c.t.Fatal("virtualtest: expected the peripheral to disconnect, but the link stayed up")
}

// HexPacket builds a fragment payload of n bytes filled with a marker value,
// for tests that only care about how many fragments arrive.
func HexPacket(marker byte, n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = marker
	}
	return hex.EncodeToString(buf)
}

// Addr formats a host and port as a dial address.
func Addr(host string, port int) string { return fmt.Sprintf("%s:%d", host, port) }
