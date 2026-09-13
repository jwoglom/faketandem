package bluetooth

import (
	"bufio"
	"crypto/sha1" //nolint:gosec // not security relevant: only used to derive a stable UUIDv5
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// VirtualProtocolVersion is the version of the virtual GATT link protocol
// implemented here. See docs/virtual-gatt-protocol.md.
const VirtualProtocolVersion = 1

// DefaultVirtualAddr is the address the virtual transport listens on when no
// address is configured.
const DefaultVirtualAddr = "127.0.0.1:8081"

// maxVirtualLine bounds a single newline-delimited JSON message. A GATT write
// is at most a few hundred bytes even with generous MTUs; 1 MiB is far beyond
// anything legitimate and keeps a malformed peer from exhausting memory.
const maxVirtualLine = 1 << 20

// GATT property names used in the virtual link's service table.
const (
	propRead                 = "read"
	propWrite                = "write"
	propWriteWithoutResponse = "write_without_response"
	propNotify               = "notify"
	propIndicate             = "indicate"
)

// virtualChar describes one characteristic in the advertised GATT table.
type virtualChar struct {
	UUID       string   `json:"uuid"`
	Properties []string `json:"properties"`
}

// virtualService describes one service in the advertised GATT table.
type virtualService struct {
	UUID            string        `json:"uuid"`
	Characteristics []virtualChar `json:"characteristics"`
}

// virtualMessage is the wire format of every message on the virtual link. All
// fields are optional; "type" selects which are meaningful.
type virtualMessage struct {
	Type string `json:"type"`
	ID   *int   `json:"id,omitempty"`

	Characteristic string `json:"characteristic,omitempty"`
	Value          string `json:"value,omitempty"`
	WithResponse   *bool  `json:"with_response,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`

	Error  string `json:"error,omitempty"`
	Reason string `json:"reason,omitempty"`

	// hello fields
	ProtocolVersion  int              `json:"protocol_version,omitempty"`
	PeripheralID     string           `json:"peripheral_id,omitempty"`
	Name             string           `json:"name,omitempty"`
	ManufacturerData string           `json:"manufacturer_data,omitempty"`
	PairingState     PairingState     `json:"pairing_state,omitempty"`
	Services         []virtualService `json:"services,omitempty"`
}

// VirtualOptions configures a VirtualTransport.
type VirtualOptions struct {
	// Addr is the TCP address to listen on. Defaults to DefaultVirtualAddr.
	Addr string
	// PeripheralID is the stable UUID reported in the hello message. When
	// empty it is derived deterministically from SerialNumber.
	PeripheralID string
	// SerialNumber seeds the default PeripheralID. When empty, PumpName is
	// used instead.
	SerialNumber string
	// AckAfterHandling runs the write handler synchronously and acknowledges
	// the write afterwards, reproducing the Linux/gatt ordering. The default
	// (false) acknowledges immediately and dispatches asynchronously, which is
	// how a real pump behaves.
	AckAfterHandling bool
}

// VirtualTransport is a radio-free Transport: it speaks the virtual GATT link
// protocol over TCP, so a central with a real BLE stack can be exercised
// against the emulator one layer below CoreBluetooth. It is safe for
// concurrent use.
type VirtualTransport struct {
	addr             string
	peripheralID     string
	ackAfterHandling bool

	listener net.Listener

	mu       sync.Mutex
	attached *virtualConn

	// Data storage for each pump characteristic (for reads)
	charData    map[CharacteristicType][]byte
	charDataMtx sync.RWMutex

	// Data storage for the non-pump (GAP/DIS) characteristics, keyed by
	// normalized UUID.
	extraCharData    map[string][]byte
	extraCharDataMtx sync.RWMutex

	handlerMtx        sync.RWMutex
	writeHandler      WriteHandler
	readHandler       ReadHandler
	connectionHandler ConnectionHandler

	pairingState    PairingState
	pairingStateMtx sync.RWMutex

	// radioOff models a pump whose BLE radio has gone away: the listener
	// keeps running (so a harness can turn it back on) but every connection
	// is refused and any current one is dropped.
	radioOff bool
	// notifyFilter, when set, is consulted for every notification fragment
	// before it leaves the link. Returning false drops that one fragment,
	// which is the fragment-level seam faults are injected at -- the router
	// works in whole messages and cannot express "this fragment and no more".
	notifyFilter func(charType CharacteristicType, data []byte) bool
	radioMtx     sync.RWMutex

	services []virtualService
	// uuidToChar maps a normalized pump characteristic UUID to its type.
	uuidToChar map[string]CharacteristicType
}

// NewVirtual creates a VirtualTransport and starts listening. The returned
// transport is advertising immediately, but (like the Linux transport) it
// starts in PairingStateNotDiscoverable and rejects connections until the
// pairing state is changed.
func NewVirtual(opts VirtualOptions) (*VirtualTransport, error) {
	addr := opts.Addr
	if addr == "" {
		addr = DefaultVirtualAddr
	}

	peripheralID := opts.PeripheralID
	if peripheralID == "" {
		seed := opts.SerialNumber
		if seed == "" {
			seed = PumpName
		}
		peripheralID = derivePeripheralID(seed)
	}

	v := &VirtualTransport{
		addr:             addr,
		peripheralID:     peripheralID,
		ackAfterHandling: opts.AckAfterHandling,
		charData:         make(map[CharacteristicType][]byte),
		extraCharData:    make(map[string][]byte),
		pairingState:     PairingStateNotDiscoverable,
		uuidToChar:       make(map[string]CharacteristicType),
	}
	v.buildServiceTable()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("virtual transport: failed to listen on %s: %w", addr, err)
	}
	v.listener = ln

	log.Infof("pkg bluetooth; virtual GATT transport listening on %s (peripheral_id=%s)", ln.Addr(), peripheralID)
	go v.acceptLoop(ln)

	return v, nil
}

// Addr returns the address the virtual transport is listening on. Useful when
// the configured address used port 0.
func (v *VirtualTransport) Addr() string {
	if v.listener == nil {
		return v.addr
	}
	return v.listener.Addr().String()
}

// PeripheralID returns the stable peripheral identifier reported in hello.
func (v *VirtualTransport) PeripheralID() string {
	return v.peripheralID
}

// Close stops the listener and drops any connected central.
func (v *VirtualTransport) Close() error {
	v.ShutdownConnection()
	if v.listener == nil {
		return nil
	}
	return v.listener.Close()
}

// derivePeripheralID produces a stable RFC 4122 v5-shaped UUID from seed, so
// the same emulator identity persists across runs without any state on disk.
func derivePeripheralID(seed string) string {
	sum := sha1.Sum([]byte("faketandem:virtual-peripheral:" + seed)) //nolint:gosec // see import comment
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x50 // version 5
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}

// buildServiceTable mirrors the services, characteristics and properties the
// Linux GATT server registers, in the same registration order.
func (v *VirtualTransport) buildServiceTable() {
	writeNotify := []string{propWrite, propWriteWithoutResponse, propNotify, propIndicate}
	notifyOnly := []string{propNotify, propIndicate}
	readOnly := []string{propRead}
	readWrite := []string{propRead, propWrite, propWriteWithoutResponse}
	writeOnly := []string{propWrite, propWriteWithoutResponse}

	gap := virtualService{
		UUID: NormalizeUUID(GenericAccessServiceUUID),
		Characteristics: []virtualChar{
			{UUID: NormalizeUUID(DeviceNameCharUUID), Properties: readWrite},
			{UUID: NormalizeUUID(AppearanceCharUUID), Properties: readOnly},
			{UUID: NormalizeUUID(PeripheralPreferredConnectionParametersCharUUID), Properties: readOnly},
			{UUID: NormalizeUUID(CentralAddressResolutionCharUUID), Properties: readOnly},
		},
	}
	v.setExtraCharacteristicData(DeviceNameCharUUID, []byte(PumpName))
	v.setExtraCharacteristicData(AppearanceCharUUID, gapAppearance)
	v.setExtraCharacteristicData(PeripheralPreferredConnectionParametersCharUUID, gapPeripheralPreferredConnectionParameters)
	v.setExtraCharacteristicData(CentralAddressResolutionCharUUID, gapCentralAddressResolution)

	gatt := virtualService{
		UUID: NormalizeUUID(GenericAttributeServiceUUID),
		Characteristics: []virtualChar{
			{UUID: NormalizeUUID(ServiceChangedCharUUID), Properties: notifyOnly},
		},
	}

	dis := virtualService{
		UUID: NormalizeUUID(DeviceInformationServiceUUID),
		Characteristics: []virtualChar{
			{UUID: NormalizeUUID(ManufacturerNameStringCharUUID), Properties: readOnly},
			{UUID: NormalizeUUID(ModelNumberStringCharUUID), Properties: readOnly},
			{UUID: NormalizeUUID(SerialNumberStringCharUUID), Properties: readOnly},
			{UUID: NormalizeUUID(SoftwareRevisionStringCharUUID), Properties: readOnly},
		},
	}
	v.setExtraCharacteristicData(ManufacturerNameStringCharUUID, []byte(DISManufacturerName))
	v.setExtraCharacteristicData(ModelNumberStringCharUUID, []byte(DISModelNumber))
	v.setExtraCharacteristicData(SerialNumberStringCharUUID, []byte(DISSerialNumber()))
	v.setExtraCharacteristicData(SoftwareRevisionStringCharUUID, []byte(DISSoftwareRevision))

	pumpChars := []struct {
		charType CharacteristicType
		props    []string
	}{
		{CharCurrentStatus, writeNotify},
		{CharQualifyingEvents, writeNotify},
		{CharHistoryLog, notifyOnly},
		{CharAuthorization, writeNotify},
		{CharControl, writeNotify},
		{CharControlStream, writeNotify},
	}
	pump := virtualService{UUID: NormalizeUUID(PumpServiceUUID)}
	for _, c := range pumpChars {
		uuid := NormalizeUUID(PumpCharacteristicUUID(c.charType))
		pump.Characteristics = append(pump.Characteristics, virtualChar{UUID: uuid, Properties: c.props})
		v.uuidToChar[uuid] = c.charType
	}

	unknown := virtualService{
		UUID: NormalizeUUID("FDFA"),
		Characteristics: []virtualChar{
			{UUID: NormalizeUUID(UnknownCharFFE8UUID), Properties: writeNotify},
			{UUID: NormalizeUUID(UnknownCharFFE7UUID), Properties: writeOnly},
		},
	}

	// Same registration order as bluetooth_linux.go's setupService.
	v.services = []virtualService{gap, gatt, dis, pump, unknown}
}

// ---------------------------------------------------------------------------
// Connection handling
// ---------------------------------------------------------------------------

func (v *VirtualTransport) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				log.Debug("pkg bluetooth; virtual transport listener closed")
				return
			}
			log.Warnf("pkg bluetooth; virtual transport accept error: %v", err)
			return
		}
		go v.handleConn(conn)
	}
}

// virtualConn is one TCP connection, i.e. at most one BLE connection.
type virtualConn struct {
	v    *VirtualTransport
	conn net.Conn

	writeMtx sync.Mutex

	subsMtx sync.RWMutex
	subs    map[string]bool

	attached bool

	closeOnce sync.Once
	work      chan func()
}

func (v *VirtualTransport) handleConn(conn net.Conn) {
	c := &virtualConn{
		v:    v,
		conn: conn,
		subs: make(map[string]bool),
		work: make(chan func(), 64),
	}
	go c.workLoop()

	state := v.GetPairingState()
	if err := c.send(virtualMessage{
		Type:             "hello",
		ProtocolVersion:  VirtualProtocolVersion,
		PeripheralID:     v.peripheralID,
		Name:             PumpName,
		ManufacturerData: hex.EncodeToString(ManufacturerData(state)),
		PairingState:     state,
		Services:         v.services,
	}); err != nil {
		log.Warnf("pkg bluetooth; virtual transport failed to send hello: %v", err)
		c.close()
		return
	}

	if !v.RadioEnabled() {
		log.Warnf("pkg bluetooth; virtual transport rejecting connection from %s - radio is off", conn.RemoteAddr())
		c.sendDisconnect("radio_off")
		c.close()
		return
	}

	if state == PairingStateNotDiscoverable {
		log.Warnf("pkg bluetooth; virtual transport rejecting connection from %s - not discoverable", conn.RemoteAddr())
		c.sendDisconnect("not_discoverable")
		c.close()
		return
	}

	c.readLoop()
}

// workLoop runs dispatched write handlers serially, in arrival order, off the
// socket read goroutine.
func (c *virtualConn) workLoop() {
	for fn := range c.work {
		fn()
	}
}

func (c *virtualConn) readLoop() {
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, 64*1024), maxVirtualLine)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg virtualMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Warnf("pkg bluetooth; virtual transport received malformed JSON: %v", err)
			_ = c.send(virtualMessage{Type: "error", Error: "malformed json"})
			continue
		}
		if !c.dispatch(msg) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Debugf("pkg bluetooth; virtual transport read error: %v", err)
	}
	// EOF or error == BLE disconnect.
	c.close()
}

// dispatch handles one inbound message. It returns false when the connection
// should be torn down.
func (c *virtualConn) dispatch(msg virtualMessage) bool {
	switch msg.Type {
	case "attach":
		return c.handleAttach(msg)
	case "write":
		c.handleWrite(msg)
	case "read":
		c.handleRead(msg)
	case "subscribe":
		c.handleSubscribe(msg)
	case "disconnect":
		log.Infof("pkg bluetooth; virtual transport central requested disconnect")
		return false
	default:
		log.Warnf("pkg bluetooth; virtual transport received unknown message type %q", msg.Type)
		_ = c.send(virtualMessage{Type: "error", ID: msg.ID, Error: "unknown type"})
	}
	return true
}

func (c *virtualConn) handleAttach(msg virtualMessage) bool {
	v := c.v

	v.mu.Lock()
	if v.attached != nil && v.attached != c {
		v.mu.Unlock()
		log.Warnf("pkg bluetooth; virtual transport rejecting second central from %s - busy", c.conn.RemoteAddr())
		c.sendDisconnect("busy")
		return false
	}
	if c.attached {
		v.mu.Unlock()
		_ = c.send(virtualMessage{Type: "attach_ack", ID: msg.ID})
		return true
	}
	v.attached = c
	c.attached = true
	v.mu.Unlock()

	log.Infof("pkg bluetooth; ** New virtual connection from: %s", c.conn.RemoteAddr())
	if err := c.send(virtualMessage{Type: "attach_ack", ID: msg.ID}); err != nil {
		log.Warnf("pkg bluetooth; virtual transport failed to ack attach: %v", err)
		return false
	}

	if h := v.getConnectionHandler(); h != nil {
		h(true)
	}
	return true
}

func (c *virtualConn) handleWrite(msg virtualMessage) {
	data, err := hex.DecodeString(msg.Value)
	if err != nil {
		_ = c.send(virtualMessage{Type: "write_ack", ID: msg.ID, Error: "invalid hex value"})
		return
	}

	uuid := NormalizeUUID(msg.Characteristic)
	charType, isPumpChar := c.v.uuidToChar[uuid]

	log.Debugf("pkg bluetooth; virtual transport received write on %s: %s", uuid, hex.EncodeToString(data))

	handle := func() {
		if !isPumpChar {
			// GAP device name and the unknown FDFA characteristics are
			// writable but never routed; store the value like the Linux
			// transport does and drop it otherwise.
			if uuid == NormalizeUUID(DeviceNameCharUUID) {
				c.v.setExtraCharacteristicData(uuid, data)
			}
			return
		}
		if h := c.v.getWriteHandler(); h != nil {
			h(charType, data)
		}
	}

	if c.v.ackAfterHandling {
		// Linux/gatt ordering: the handler runs to completion (emitting any
		// response notifications) before the write is acknowledged.
		handle()
		_ = c.send(virtualMessage{Type: "write_ack", ID: msg.ID})
		return
	}

	// Real-pump ordering: acknowledge immediately, handle asynchronously on a
	// per-connection serial worker so write order is preserved.
	_ = c.send(virtualMessage{Type: "write_ack", ID: msg.ID})
	c.enqueue(handle)
}

func (c *virtualConn) enqueue(fn func()) {
	defer func() {
		// The work channel is closed on teardown; a write racing with close
		// would panic. Dropping the work is the correct behavior there, since
		// the connection is gone.
		if r := recover(); r != nil {
			log.Debugf("pkg bluetooth; virtual transport dropped work after close: %v", r)
		}
	}()
	c.work <- fn
}

func (c *virtualConn) handleRead(msg virtualMessage) {
	uuid := NormalizeUUID(msg.Characteristic)

	if charType, ok := c.v.uuidToChar[uuid]; ok {
		var data []byte
		if h := c.v.getReadHandler(); h != nil {
			data = h(charType)
		}
		if data == nil {
			data = c.v.getCharacteristicData(charType)
		}
		if data == nil {
			data = []byte{}
		}
		_ = c.send(virtualMessage{Type: "read_result", ID: msg.ID, Characteristic: uuid, Value: hex.EncodeToString(data)})
		return
	}

	data := c.v.getExtraCharacteristicData(uuid)
	if data == nil {
		_ = c.send(virtualMessage{Type: "read_result", ID: msg.ID, Characteristic: uuid, Error: "unknown characteristic"})
		return
	}
	log.Debugf("pkg bluetooth; virtual transport read request on %s, responding with: %s", uuid, hex.EncodeToString(data))
	_ = c.send(virtualMessage{Type: "read_result", ID: msg.ID, Characteristic: uuid, Value: hex.EncodeToString(data)})
}

func (c *virtualConn) handleSubscribe(msg virtualMessage) {
	uuid := NormalizeUUID(msg.Characteristic)
	enabled := true
	if msg.Enabled != nil {
		enabled = *msg.Enabled
	}

	c.subsMtx.Lock()
	if enabled {
		c.subs[uuid] = true
	} else {
		delete(c.subs, uuid)
	}
	c.subsMtx.Unlock()

	log.Infof("pkg bluetooth; virtual transport notifications %s for %s from %s",
		map[bool]string{true: "enabled", false: "disabled"}[enabled], uuid, c.conn.RemoteAddr())
	_ = c.send(virtualMessage{Type: "subscribe_ack", ID: msg.ID, Characteristic: uuid, Enabled: &enabled})
}

func (c *virtualConn) isSubscribed(uuid string) bool {
	c.subsMtx.RLock()
	defer c.subsMtx.RUnlock()
	return c.subs[uuid]
}

func (c *virtualConn) send(msg virtualMessage) error {
	line, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	c.writeMtx.Lock()
	defer c.writeMtx.Unlock()
	_, err = c.conn.Write(line)
	return err
}

func (c *virtualConn) sendDisconnect(reason string) {
	if err := c.send(virtualMessage{Type: "disconnect", Reason: reason}); err != nil {
		log.Debugf("pkg bluetooth; virtual transport failed to send disconnect: %v", err)
	}
}

// close tears the connection down exactly once, firing the connection handler
// if this connection was the attached central.
func (c *virtualConn) close() {
	c.closeOnce.Do(func() {
		if err := c.conn.Close(); err != nil {
			log.Debugf("pkg bluetooth; error closing virtual connection: %v", err)
		}
		close(c.work)

		v := c.v
		v.mu.Lock()
		wasAttached := v.attached == c
		if wasAttached {
			v.attached = nil
		}
		v.mu.Unlock()

		if wasAttached {
			log.Debugf("pkg bluetooth; ** virtual disconnect: %s", c.conn.RemoteAddr())
			if h := v.getConnectionHandler(); h != nil {
				h(false)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Transport implementation
// ---------------------------------------------------------------------------

// SetWriteHandler sets the callback for when data is written to any characteristic
func (v *VirtualTransport) SetWriteHandler(handler WriteHandler) {
	v.handlerMtx.Lock()
	defer v.handlerMtx.Unlock()
	v.writeHandler = handler
}

// SetReadHandler sets the callback for when data is read from any characteristic
func (v *VirtualTransport) SetReadHandler(handler ReadHandler) {
	v.handlerMtx.Lock()
	defer v.handlerMtx.Unlock()
	v.readHandler = handler
}

// SetConnectionHandler sets the callback for when a central connects or disconnects
func (v *VirtualTransport) SetConnectionHandler(handler ConnectionHandler) {
	v.handlerMtx.Lock()
	defer v.handlerMtx.Unlock()
	v.connectionHandler = handler
}

func (v *VirtualTransport) getWriteHandler() WriteHandler {
	v.handlerMtx.RLock()
	defer v.handlerMtx.RUnlock()
	return v.writeHandler
}

func (v *VirtualTransport) getReadHandler() ReadHandler {
	v.handlerMtx.RLock()
	defer v.handlerMtx.RUnlock()
	return v.readHandler
}

func (v *VirtualTransport) getConnectionHandler() ConnectionHandler {
	v.handlerMtx.RLock()
	defer v.handlerMtx.RUnlock()
	return v.connectionHandler
}

// SetCharacteristicData sets the data that will be returned when a characteristic is read
func (v *VirtualTransport) SetCharacteristicData(charType CharacteristicType, data []byte) {
	dataCopy := append([]byte(nil), data...)
	v.charDataMtx.Lock()
	defer v.charDataMtx.Unlock()
	v.charData[charType] = dataCopy
}

func (v *VirtualTransport) getCharacteristicData(charType CharacteristicType) []byte {
	v.charDataMtx.RLock()
	defer v.charDataMtx.RUnlock()
	data, ok := v.charData[charType]
	if !ok {
		return nil
	}
	return append([]byte(nil), data...)
}

func (v *VirtualTransport) setExtraCharacteristicData(uuidStr string, data []byte) {
	if data == nil {
		return
	}
	key := NormalizeUUID(uuidStr)
	dataCopy := append([]byte(nil), data...)
	v.extraCharDataMtx.Lock()
	v.extraCharData[key] = dataCopy
	v.extraCharDataMtx.Unlock()
}

func (v *VirtualTransport) getExtraCharacteristicData(uuidStr string) []byte {
	key := NormalizeUUID(uuidStr)
	v.extraCharDataMtx.RLock()
	data := v.extraCharData[key]
	v.extraCharDataMtx.RUnlock()
	if data == nil {
		return nil
	}
	return append([]byte(nil), data...)
}

// Notify sends a notification on the specified characteristic. Like real BLE
// (and CoreBluetooth), notifications are delivered only while the central is
// subscribed; otherwise they are dropped and an error is returned.
func (v *VirtualTransport) Notify(charType CharacteristicType, data []byte) error {
	uuid := NormalizeUUID(PumpCharacteristicUUID(charType))
	if uuid == "" {
		return fmt.Errorf("unknown characteristic %s", charType)
	}

	v.mu.Lock()
	c := v.attached
	v.mu.Unlock()

	if c == nil {
		return fmt.Errorf("no notifier registered for %s", charType)
	}
	if !c.isSubscribed(uuid) {
		log.Debugf("pkg bluetooth; virtual transport dropping notification on %s: central is not subscribed", charType)
		return fmt.Errorf("no notifier registered for %s", charType)
	}

	if filter := v.getNotifyFilter(); filter != nil && !filter(charType, data) {
		log.Warnf("pkg bluetooth; virtual transport dropping notification fragment on %s per notify filter", charType)
		return nil
	}

	log.Debugf("pkg bluetooth; virtual transport sending notification on %s: %s", charType, hex.EncodeToString(data))
	return c.send(virtualMessage{Type: "notify", Characteristic: uuid, Value: hex.EncodeToString(data)})
}

// IsConnected returns true if a central device is attached
func (v *VirtualTransport) IsConnected() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.attached != nil
}

// ShutdownConnection closes the connection with the attached central
func (v *VirtualTransport) ShutdownConnection() {
	v.mu.Lock()
	c := v.attached
	v.mu.Unlock()

	if c == nil {
		return
	}
	c.sendDisconnect("peripheral_shutdown")
	c.close()
}

// SetPairingState sets the pairing/discoverable state
func (v *VirtualTransport) SetPairingState(state PairingState) error {
	v.pairingStateMtx.Lock()
	v.pairingState = state
	v.pairingStateMtx.Unlock()

	// If setting to not discoverable, disconnect any existing connection.
	if state == PairingStateNotDiscoverable && v.IsConnected() {
		log.Info("pkg bluetooth; disconnecting existing virtual connection due to non-discoverable mode")
		v.ShutdownConnection()
	}

	log.Infof("Pairing state set to: %v", state)
	return nil
}

// GetPairingState returns the current pairing state
func (v *VirtualTransport) GetPairingState() PairingState {
	v.pairingStateMtx.RLock()
	defer v.pairingStateMtx.RUnlock()
	return v.pairingState
}

// SetRadioEnabled turns the virtual radio on or off. Turning it off drops the
// attached central and refuses new connections until it is turned back on,
// which is how a harness reproduces a pump that went out of range or powered
// its radio down without tearing the emulator down.
func (v *VirtualTransport) SetRadioEnabled(enabled bool) {
	v.radioMtx.Lock()
	v.radioOff = !enabled
	v.radioMtx.Unlock()

	if enabled {
		log.Info("pkg bluetooth; virtual transport radio on")
		return
	}

	log.Info("pkg bluetooth; virtual transport radio off; dropping any attached central")
	v.ShutdownConnection()
}

// RadioEnabled reports whether the virtual radio is on.
func (v *VirtualTransport) RadioEnabled() bool {
	v.radioMtx.RLock()
	defer v.radioMtx.RUnlock()
	return !v.radioOff
}

// SetNotifyFilter installs (or, with nil, removes) the per-fragment filter
// consulted before each notification leaves the link. Returning false from it
// drops that fragment silently, exactly as a lost BLE notification would be.
func (v *VirtualTransport) SetNotifyFilter(filter func(charType CharacteristicType, data []byte) bool) {
	v.radioMtx.Lock()
	defer v.radioMtx.Unlock()
	v.notifyFilter = filter
}

func (v *VirtualTransport) getNotifyFilter() func(charType CharacteristicType, data []byte) bool {
	v.radioMtx.RLock()
	defer v.radioMtx.RUnlock()
	return v.notifyFilter
}
