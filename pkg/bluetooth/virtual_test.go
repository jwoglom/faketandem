package bluetooth

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

const testReadTimeout = 3 * time.Second

// testClient is a minimal virtual-GATT central used to drive the transport
// over loopback.
type testClient struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
	next int
}

func dialTransport(t *testing.T, v *VirtualTransport) *testClient {
	t.Helper()
	conn, err := net.Dial("tcp", v.Addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &testClient{t: t, conn: conn, r: bufio.NewReader(conn)}
}

func (c *testClient) send(msg virtualMessage) int {
	c.t.Helper()
	c.next++
	id := c.next
	msg.ID = &id
	line, err := json.Marshal(msg)
	if err != nil {
		c.t.Fatalf("marshal: %v", err)
	}
	if _, err := c.conn.Write(append(line, '\n')); err != nil {
		c.t.Fatalf("write: %v", err)
	}
	return id
}

// recv reads the next message, failing the test on timeout.
func (c *testClient) recv() virtualMessage {
	c.t.Helper()
	msg, err := c.tryRecv()
	if err != nil {
		c.t.Fatalf("recv: %v", err)
	}
	return msg
}

func (c *testClient) tryRecv() (virtualMessage, error) {
	c.t.Helper()
	if err := c.conn.SetReadDeadline(time.Now().Add(testReadTimeout)); err != nil {
		return virtualMessage{}, err
	}
	line, err := c.r.ReadString('\n')
	if err != nil {
		return virtualMessage{}, err
	}
	var msg virtualMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &msg); err != nil {
		return virtualMessage{}, err
	}
	return msg, nil
}

// expectNothing asserts that no message arrives within a short window.
func (c *testClient) expectNothing(d time.Duration) {
	c.t.Helper()
	if err := c.conn.SetReadDeadline(time.Now().Add(d)); err != nil {
		c.t.Fatalf("set deadline: %v", err)
	}
	line, err := c.r.ReadString('\n')
	if err == nil {
		c.t.Fatalf("expected no message, got %s", strings.TrimSpace(line))
	}
	var netErr net.Error
	if !isTimeout(err, &netErr) {
		c.t.Fatalf("expected timeout, got %v", err)
	}
}

func isTimeout(err error, target *net.Error) bool {
	if ne, ok := err.(net.Error); ok {
		*target = ne
		return ne.Timeout()
	}
	return false
}

func (c *testClient) hello() virtualMessage {
	c.t.Helper()
	msg := c.recv()
	if msg.Type != "hello" {
		c.t.Fatalf("expected hello, got %q", msg.Type)
	}
	return msg
}

func (c *testClient) attach() {
	c.t.Helper()
	id := c.send(virtualMessage{Type: "attach"})
	ack := c.recv()
	if ack.Type != "attach_ack" || ack.ID == nil || *ack.ID != id {
		c.t.Fatalf("expected attach_ack id=%d, got %+v", id, ack)
	}
}

func (c *testClient) subscribe(uuid string, enabled bool) {
	c.t.Helper()
	id := c.send(virtualMessage{Type: "subscribe", Characteristic: uuid, Enabled: &enabled})
	ack := c.recv()
	if ack.Type != "subscribe_ack" || ack.ID == nil || *ack.ID != id {
		c.t.Fatalf("expected subscribe_ack id=%d, got %+v", id, ack)
	}
}

func newTestTransport(t *testing.T, opts VirtualOptions) *VirtualTransport {
	t.Helper()
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:0"
	}
	v, err := NewVirtual(opts)
	if err != nil {
		t.Fatalf("NewVirtual: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })
	return v
}

// waitFor polls cond until it holds or the timeout elapses.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(testReadTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestVirtualTransportHelloContents(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{SerialNumber: "11223344"})
	if err := v.SetPairingState(PairingStatePairStep1); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	c := dialTransport(t, v)
	hello := c.hello()

	if hello.ProtocolVersion != VirtualProtocolVersion {
		t.Errorf("protocol_version = %d, want %d", hello.ProtocolVersion, VirtualProtocolVersion)
	}
	if hello.PeripheralID != derivePeripheralID("11223344") {
		t.Errorf("peripheral_id = %q, not derived from serial number", hello.PeripheralID)
	}
	if hello.PeripheralID != v.PeripheralID() {
		t.Errorf("peripheral_id %q != PeripheralID() %q", hello.PeripheralID, v.PeripheralID())
	}
	if hello.Name != PumpName {
		t.Errorf("name = %q, want %q", hello.Name, PumpName)
	}
	if want := hex.EncodeToString(ManufacturerData(PairingStatePairStep1)); hello.ManufacturerData != want {
		t.Errorf("manufacturer_data = %q, want %q", hello.ManufacturerData, want)
	}
	if !strings.HasSuffix(hello.ManufacturerData, "11") {
		t.Errorf("manufacturer_data %q should end in the PairStep1 byte 0x11", hello.ManufacturerData)
	}
	if hello.PairingState != PairingStatePairStep1 {
		t.Errorf("pairing_state = %q", hello.PairingState)
	}
}

func TestVirtualTransportServiceTable(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}
	c := dialTransport(t, v)
	hello := c.hello()

	// Same five services in the same order as the Linux GATT server.
	wantServices := []string{
		NormalizeUUID(GenericAccessServiceUUID),
		NormalizeUUID(GenericAttributeServiceUUID),
		NormalizeUUID(DeviceInformationServiceUUID),
		NormalizeUUID(PumpServiceUUID),
		NormalizeUUID("FDFA"),
	}
	if len(hello.Services) != len(wantServices) {
		t.Fatalf("got %d services, want %d", len(hello.Services), len(wantServices))
	}
	for i, want := range wantServices {
		if hello.Services[i].UUID != want {
			t.Errorf("service[%d] = %q, want %q", i, hello.Services[i].UUID, want)
		}
	}

	pump := hello.Services[3]
	if len(pump.Characteristics) != 6 {
		t.Fatalf("pump service has %d characteristics, want 6", len(pump.Characteristics))
	}
	if pump.Characteristics[0].UUID != NormalizeUUID(CurrentStatusCharUUID) {
		t.Errorf("first pump characteristic = %q", pump.Characteristics[0].UUID)
	}
	if !hasProp(pump.Characteristics[0].Properties, propWrite) || !hasProp(pump.Characteristics[0].Properties, propNotify) {
		t.Errorf("CurrentStatus properties = %v, want write+notify", pump.Characteristics[0].Properties)
	}

	// HistoryLog is notify-only on the Linux server.
	hist := pump.Characteristics[2]
	if hist.UUID != NormalizeUUID(HistoryLogCharUUID) {
		t.Fatalf("expected HistoryLog at index 2, got %q", hist.UUID)
	}
	if hasProp(hist.Properties, propWrite) {
		t.Errorf("HistoryLog should not be writable: %v", hist.Properties)
	}

	dis := hello.Services[2]
	if len(dis.Characteristics) != 4 {
		t.Errorf("DIS has %d characteristics, want 4", len(dis.Characteristics))
	}
}

func TestVirtualTransportAttachLifecycle(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	connected := make(chan bool, 4)
	v.SetConnectionHandler(func(c bool) { connected <- c })

	c := dialTransport(t, v)
	c.hello()

	// ConnectionHandler must not fire on hello alone: a central may connect
	// purely to read hello as a stand-in for a scan.
	select {
	case got := <-connected:
		t.Fatalf("connection handler fired (%v) before attach", got)
	case <-time.After(100 * time.Millisecond):
	}
	if v.IsConnected() {
		t.Error("IsConnected() true before attach")
	}

	c.attach()

	select {
	case got := <-connected:
		if !got {
			t.Error("connection handler fired with false on attach")
		}
	case <-time.After(testReadTimeout):
		t.Fatal("connection handler did not fire on attach")
	}
	if !v.IsConnected() {
		t.Error("IsConnected() false after attach")
	}

	// EOF == disconnect.
	_ = c.conn.Close()
	select {
	case got := <-connected:
		if got {
			t.Error("expected disconnect(false) on EOF")
		}
	case <-time.After(testReadTimeout):
		t.Fatal("connection handler did not fire on EOF")
	}
	waitFor(t, "IsConnected() false after EOF", func() bool { return !v.IsConnected() })
}

func hasProp(props []string, want string) bool {
	for _, p := range props {
		if p == want {
			return true
		}
	}
	return false
}

func TestVirtualTransportRejectsWhenNotDiscoverable(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	fired := make(chan bool, 1)
	v.SetConnectionHandler(func(c bool) { fired <- c })

	c := dialTransport(t, v)
	hello := c.hello()
	if hello.PairingState != PairingStateNotDiscoverable {
		t.Errorf("pairing_state = %q, want NotDiscoverable", hello.PairingState)
	}

	msg := c.recv()
	if msg.Type != "disconnect" || msg.Reason != "not_discoverable" {
		t.Fatalf("expected disconnect/not_discoverable, got %+v", msg)
	}
	if _, err := c.tryRecv(); err == nil {
		t.Fatal("expected socket to be closed after rejection")
	}

	select {
	case got := <-fired:
		t.Fatalf("connection handler fired (%v) for a rejected connection", got)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestVirtualTransportSecondCentralIsBusy(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	first := dialTransport(t, v)
	first.hello()
	first.attach()

	second := dialTransport(t, v)
	second.hello() // a scan-style peek always gets hello
	second.send(virtualMessage{Type: "attach"})
	msg := second.recv()
	if msg.Type != "disconnect" || msg.Reason != "busy" {
		t.Fatalf("expected disconnect/busy for second central, got %+v", msg)
	}
	if _, err := second.tryRecv(); err == nil {
		t.Fatal("expected second connection to be closed")
	}

	// The first central must still be attached.
	if !v.IsConnected() {
		t.Error("first central was dropped when a second connected")
	}
}

func TestVirtualTransportWriteAckOrdering(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	handled := make(chan []byte, 4)
	release := make(chan struct{})
	v.SetWriteHandler(func(charType CharacteristicType, data []byte) {
		if charType != CharControl {
			t.Errorf("write handler got %s, want Control", charType)
		}
		<-release
		// Respond while the handler runs, as the router does.
		if err := v.Notify(CharControl, []byte{0xaa, 0xbb}); err != nil {
			t.Errorf("Notify from handler: %v", err)
		}
		handled <- data
	})

	c := dialTransport(t, v)
	c.hello()
	c.attach()

	c.subscribe(ControlCharUUID, true)

	payload := []byte{0x01, 0x02, 0x03}
	wID := c.send(virtualMessage{Type: "write", Characteristic: ControlCharUUID, Value: hex.EncodeToString(payload), WithResponse: boolPtr(true)})

	// Default (real-pump) ordering: the ack arrives before the handler runs.
	got := c.recv()
	if got.Type != "write_ack" || got.ID == nil || *got.ID != wID {
		t.Fatalf("expected immediate write_ack, got %+v", got)
	}
	if got.Error != "" {
		t.Fatalf("write_ack carried error %q", got.Error)
	}

	close(release)
	select {
	case data := <-handled:
		if hex.EncodeToString(data) != hex.EncodeToString(payload) {
			t.Errorf("handler got %x, want %x", data, payload)
		}
	case <-time.After(testReadTimeout):
		t.Fatal("write handler never ran")
	}

	n := c.recv()
	if n.Type != "notify" || n.Value != "aabb" || n.Characteristic != NormalizeUUID(ControlCharUUID) {
		t.Fatalf("expected notify aabb on Control, got %+v", n)
	}
}

func TestVirtualTransportAckAfterHandling(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{AckAfterHandling: true})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	v.SetWriteHandler(func(charType CharacteristicType, data []byte) {
		if err := v.Notify(CharControl, []byte{0xde, 0xad}); err != nil {
			t.Errorf("Notify from handler: %v", err)
		}
	})

	c := dialTransport(t, v)
	c.hello()
	c.attach()
	c.subscribe(ControlCharUUID, true)

	c.send(virtualMessage{Type: "write", Characteristic: ControlCharUUID, Value: "0102"})

	// Linux/gatt ordering: the response notification precedes the write ack.
	first := c.recv()
	if first.Type != "notify" || first.Value != "dead" {
		t.Fatalf("expected notify first in ack-after-handling mode, got %+v", first)
	}
	second := c.recv()
	if second.Type != "write_ack" {
		t.Fatalf("expected write_ack second, got %+v", second)
	}
}

func TestVirtualTransportNotifyRequiresSubscription(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	// No central at all.
	if err := v.Notify(CharCurrentStatus, []byte{0x01}); err == nil {
		t.Error("Notify with no central should fail")
	}

	c := dialTransport(t, v)
	c.hello()
	c.attach()

	// Attached but not subscribed: dropped.
	if err := v.Notify(CharCurrentStatus, []byte{0x01}); err == nil {
		t.Error("Notify without subscription should fail")
	}
	c.expectNothing(150 * time.Millisecond)

	c.subscribe(CurrentStatusCharUUID, true)

	if err := v.Notify(CharCurrentStatus, []byte{0x01, 0x02}); err != nil {
		t.Fatalf("Notify while subscribed: %v", err)
	}
	n := c.recv()
	if n.Type != "notify" || n.Value != "0102" {
		t.Fatalf("expected notify 0102, got %+v", n)
	}

	// Unsubscribe: notifications are dropped again.
	c.subscribe(CurrentStatusCharUUID, false)
	if err := v.Notify(CharCurrentStatus, []byte{0x03}); err == nil {
		t.Error("Notify after unsubscribe should fail")
	}
	c.expectNothing(150 * time.Millisecond)
}

func TestVirtualTransportReads(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	c := dialTransport(t, v)
	c.hello()
	c.attach()

	cases := []struct {
		uuid string
		want string
	}{
		{ManufacturerNameStringCharUUID, DISManufacturerName},
		{ModelNumberStringCharUUID, DISModelNumber},
		{SerialNumberStringCharUUID, DISSerialNumber()},
		{SoftwareRevisionStringCharUUID, DISSoftwareRevision},
		{DeviceNameCharUUID, PumpName},
	}
	for _, tc := range cases {
		id := c.send(virtualMessage{Type: "read", Characteristic: tc.uuid})
		res := c.recv()
		if res.Type != "read_result" || res.ID == nil || *res.ID != id {
			t.Fatalf("read %s: got %+v", tc.uuid, res)
		}
		if res.Error != "" {
			t.Fatalf("read %s: error %q", tc.uuid, res.Error)
		}
		raw, err := hex.DecodeString(res.Value)
		if err != nil {
			t.Fatalf("read %s: bad hex %q", tc.uuid, res.Value)
		}
		if string(raw) != tc.want {
			t.Errorf("read %s = %q, want %q", tc.uuid, raw, tc.want)
		}
	}

	// Stored characteristic data is served when no read handler is set.
	v.SetCharacteristicData(CharCurrentStatus, []byte{0xca, 0xfe})
	c.send(virtualMessage{Type: "read", Characteristic: CurrentStatusCharUUID})
	if res := c.recv(); res.Value != "cafe" {
		t.Errorf("stored characteristic read = %+v", res)
	}

	// A read handler takes precedence.
	v.SetReadHandler(func(charType CharacteristicType) []byte {
		if charType == CharCurrentStatus {
			return []byte{0xbe, 0xef}
		}
		return nil
	})
	c.send(virtualMessage{Type: "read", Characteristic: CurrentStatusCharUUID})
	if res := c.recv(); res.Value != "beef" {
		t.Errorf("read handler result = %+v", res)
	}

	// Unknown characteristic.
	c.send(virtualMessage{Type: "read", Characteristic: "0000abcd-0000-1000-8000-00805f9b34fb"})
	if res := c.recv(); res.Error == "" {
		t.Errorf("expected error for unknown characteristic, got %+v", res)
	}
}

func TestVirtualTransportShutdownConnection(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	fired := make(chan bool, 4)
	v.SetConnectionHandler(func(c bool) { fired <- c })

	c := dialTransport(t, v)
	c.hello()
	c.attach()
	if got := <-fired; !got {
		t.Fatal("expected connect(true)")
	}

	v.ShutdownConnection()

	msg := c.recv()
	if msg.Type != "disconnect" {
		t.Fatalf("expected disconnect, got %+v", msg)
	}
	if _, err := c.tryRecv(); err == nil {
		t.Fatal("expected socket to be closed by ShutdownConnection")
	}

	select {
	case got := <-fired:
		if got {
			t.Error("expected connect(false) after ShutdownConnection")
		}
	case <-time.After(testReadTimeout):
		t.Fatal("connection handler did not fire on ShutdownConnection")
	}
	if v.IsConnected() {
		t.Error("IsConnected() true after ShutdownConnection")
	}

	// Idempotent.
	v.ShutdownConnection()
}

func TestVirtualTransportCentralDisconnect(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}
	fired := make(chan bool, 4)
	v.SetConnectionHandler(func(c bool) { fired <- c })

	c := dialTransport(t, v)
	c.hello()
	c.attach()
	<-fired

	c.send(virtualMessage{Type: "disconnect"})
	select {
	case got := <-fired:
		if got {
			t.Error("expected connect(false) after central disconnect")
		}
	case <-time.After(testReadTimeout):
		t.Fatal("connection handler did not fire on central disconnect")
	}

	// A fresh central can attach afterwards.
	c2 := dialTransport(t, v)
	c2.hello()
	c2.attach()
	if !v.IsConnected() {
		t.Error("second central failed to attach after the first disconnected")
	}
}

func TestVirtualTransportSetPairingStateNotDiscoverableDrops(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStatePairStep2); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}
	c := dialTransport(t, v)
	hello := c.hello()
	if !strings.HasSuffix(hello.ManufacturerData, "12") {
		t.Errorf("manufacturer_data %q should end in the PairStep2 byte 0x12", hello.ManufacturerData)
	}
	c.attach()

	if err := v.SetPairingState(PairingStateNotDiscoverable); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}
	waitFor(t, "central to be dropped", func() bool { return !v.IsConnected() })
}

func TestVirtualTransportUnknownMessageType(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{})
	if err := v.SetPairingState(PairingStateDiscoverableOnly); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}
	c := dialTransport(t, v)
	c.hello()
	c.attach()

	id := c.send(virtualMessage{Type: "frobnicate"})
	msg := c.recv()
	if msg.Type != "error" || msg.Error != "unknown type" || msg.ID == nil || *msg.ID != id {
		t.Fatalf("expected error/unknown type, got %+v", msg)
	}

	// The connection stays up.
	c.send(virtualMessage{Type: "read", Characteristic: ModelNumberStringCharUUID})
	if res := c.recv(); res.Type != "read_result" {
		t.Fatalf("connection unusable after unknown type: %+v", res)
	}
}

func TestDerivePeripheralIDIsStableAndWellFormed(t *testing.T) {
	a := derivePeripheralID("11223344")
	b := derivePeripheralID("11223344")
	if a != b {
		t.Errorf("derivePeripheralID is not deterministic: %q vs %q", a, b)
	}
	if c := derivePeripheralID("99887766"); c == a {
		t.Error("different serials produced the same peripheral id")
	}
	if len(a) != 36 {
		t.Errorf("peripheral id %q is not a 36-character UUID", a)
	}
	if a[14] != '5' {
		t.Errorf("peripheral id %q is not UUID version 5", a)
	}
	if v := a[19]; v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Errorf("peripheral id %q has the wrong RFC 4122 variant nibble", a)
	}
}

func TestNormalizeUUID(t *testing.T) {
	cases := map[string]string{
		"180A":                                 "0000180a-0000-1000-8000-00805f9b34fb",
		"FDFB":                                 "0000fdfb-0000-1000-8000-00805f9b34fb",
		"7B83FFF6-9F77-4E5C-8064-AAE2C24838B9": "7b83fff6-9f77-4e5c-8064-aae2c24838b9",
	}
	for in, want := range cases {
		if got := NormalizeUUID(in); got != want {
			t.Errorf("NormalizeUUID(%q) = %q, want %q", in, got, want)
		}
	}
}

func boolPtr(b bool) *bool { return &b }

// TestVirtualATTMTUDefaultsToTheSpecFloor guards the default: an emulator
// nobody configured must keep the 20-byte notifications it has always sent.
func TestVirtualATTMTUDefaultsToTheSpecFloor(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{SerialNumber: "11223344"})
	if err := v.SetPairingState(PairingStatePairStep1); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	if got := v.ATTMTU(); got != DefaultATTMTU {
		t.Errorf("ATTMTU() = %d, want %d", got, DefaultATTMTU)
	}
	if got := MaxNotificationBytes(v.ATTMTU()); got != 20 {
		t.Errorf("a %d-byte MTU allows %d-byte notifications, want 20", v.ATTMTU(), got)
	}

	// The hello message must tell a central what the link can carry.
	if hello := dialTransport(t, v).hello(); hello.ATTMTU != DefaultATTMTU {
		t.Errorf("hello att_mtu = %d, want %d", hello.ATTMTU, DefaultATTMTU)
	}
}

// TestVirtualRejectsAnMTUOutsideTheSpecRange covers both the constructor and
// the setter, so a bad value can never reach the wire.
func TestVirtualRejectsAnMTUOutsideTheSpecRange(t *testing.T) {
	for _, mtu := range []int{22, 1, -5, MaxATTMTU + 1} {
		if _, err := NewVirtual(VirtualOptions{Addr: "127.0.0.1:0", ATTMTU: mtu}); err == nil {
			t.Errorf("NewVirtual with ATT MTU %d succeeded, want an error", mtu)
		}
	}

	v := newTestTransport(t, VirtualOptions{})

	for _, mtu := range []int{22, 0, MaxATTMTU + 1} {
		if err := v.SetATTMTU(mtu); err == nil {
			t.Errorf("SetATTMTU(%d) succeeded, want an error", mtu)
		}
	}
	if got := v.ATTMTU(); got != DefaultATTMTU {
		t.Errorf("ATTMTU() = %d after rejected updates, want it unchanged at %d", got, DefaultATTMTU)
	}

	if err := v.SetATTMTU(247); err != nil {
		t.Fatalf("SetATTMTU(247): %v", err)
	}
	if got := v.ATTMTU(); got != 247 {
		t.Errorf("ATTMTU() = %d, want 247", got)
	}
}

// TestVirtualRejectsAnOversizedNotification is the safety net under the
// fragmentation path: a notification a real link could never carry must fail
// loudly here rather than pass on the virtual link and then break against
// hardware.
func TestVirtualRejectsAnOversizedNotification(t *testing.T) {
	v := newTestTransport(t, VirtualOptions{SerialNumber: "11223344"})
	if err := v.SetPairingState(PairingStatePairStep1); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	c := dialTransport(t, v)
	c.hello()
	c.attach()
	c.subscribe(CurrentStatusCharUUID, true)

	if err := v.Notify(CharCurrentStatus, make([]byte, 20)); err != nil {
		t.Fatalf("a 20-byte notification must be accepted at the default MTU: %v", err)
	}
	if err := v.Notify(CharCurrentStatus, make([]byte, 21)); err == nil {
		t.Error("a 21-byte notification was accepted at a 23-byte ATT MTU, want an error")
	}

	if err := v.SetATTMTU(247); err != nil {
		t.Fatalf("SetATTMTU: %v", err)
	}
	if err := v.Notify(CharCurrentStatus, make([]byte, 244)); err != nil {
		t.Fatalf("a 244-byte notification must be accepted at a 247-byte MTU: %v", err)
	}
	if err := v.Notify(CharCurrentStatus, make([]byte, 245)); err == nil {
		t.Error("a 245-byte notification was accepted at a 247-byte ATT MTU, want an error")
	}
}
