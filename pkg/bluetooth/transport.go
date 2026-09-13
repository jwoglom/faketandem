package bluetooth

import (
	"fmt"
	"strings"
)

// PumpName is the advertised BLE device name of the emulated pump. It is also
// the source of the Device Information Service serial number string (a real
// Tandem Mobi reports a truncated suffix of its device name there).
const PumpName = "Tandem Mobi 123"

// Device Information Service values reported by the emulated pump. These are
// shared by every transport so the virtual link presents byte-identical values
// to the Linux GATT server.
const (
	DISManufacturerName = "Tandem Diabetes Care"
	DISModelNumber      = "X2" // Always "X2" even for Mobi
	DISSoftwareRevision = "3553172181"
)

// Generic Access Service values reported by the emulated pump.
var (
	gapAppearance                              = []byte{0x00, 0x00}
	gapPeripheralPreferredConnectionParameters = []byte{0x18, 0x00, 0x28, 0x00, 0x00, 0x00, 0xf4, 0x01}
	gapCentralAddressResolution                = []byte{0x01}
)

// DISSerialNumber returns the serial number string served on characteristic
// 2A25. A real Tandem Mobi's Serial Number String characteristic reports a
// truncated suffix of its device name (e.g. name "Tandem Mobi 976" -> serial
// "bi 976"), confirmed via a btsnoop capture of an official pairing.
func DISSerialNumber() string {
	return PumpName[9:]
}

// ManufacturerData returns the BLE advertisement manufacturer-specific data
// payload (Tandem company ID 0x059D) for the given pairing state. The trailing
// byte encodes the pairing step and is the only part that varies.
func ManufacturerData(state PairingState) []byte {
	var lastByte byte
	switch state {
	case PairingStateNotDiscoverable:
		lastByte = 0x10 // Not discoverable
	case PairingStateDiscoverableOnly:
		lastByte = 0x10 // Discoverable but no pairing step
	case PairingStatePairStep1:
		lastByte = 0x11 // Discoverable with PairStep1
	case PairingStatePairStep2:
		lastByte = 0x12 // Discoverable with PairStep2
	default:
		lastByte = 0x10
	}
	return []byte{0x00, 0x01, lastByte}
}

// PumpCharacteristicUUID returns the full 128-bit UUID string for a pump
// characteristic type, or "" if the type is unknown.
func PumpCharacteristicUUID(charType CharacteristicType) string {
	switch charType {
	case CharCurrentStatus:
		return CurrentStatusCharUUID
	case CharQualifyingEvents:
		return QualifyingEventsCharUUID
	case CharHistoryLog:
		return HistoryLogCharUUID
	case CharAuthorization:
		return AuthorizationCharUUID
	case CharControl:
		return ControlCharUUID
	case CharControlStream:
		return ControlStreamCharUUID
	default:
		return ""
	}
}

// NormalizeUUID expands a 16-bit UUID to its full 128-bit form and lowercases
// it, so both sides of a link can compare UUIDs as plain strings.
func NormalizeUUID(uuid string) string {
	u := strings.ToLower(strings.TrimSpace(uuid))
	u = strings.ReplaceAll(u, "0x", "")
	switch len(u) {
	case 4:
		return "0000" + u + "-0000-1000-8000-00805f9b34fb"
	case 8:
		return u + "-0000-1000-8000-00805f9b34fb"
	default:
		return u
	}
}

// Transport is the set of operations the emulator needs from a peripheral-side
// link to a central. The Linux GATT implementation (*Ble) is the production
// transport; VirtualTransport is a radio-free TCP implementation used for
// testing on machines without a BLE adapter.
type Transport interface {
	// SetWriteHandler sets the callback invoked when a central writes to a
	// pump characteristic.
	SetWriteHandler(handler WriteHandler)
	// SetReadHandler sets the callback invoked when a central reads a pump
	// characteristic.
	SetReadHandler(handler ReadHandler)
	// SetConnectionHandler sets the callback invoked when a central attaches
	// (true) or detaches (false).
	SetConnectionHandler(handler ConnectionHandler)
	// SetCharacteristicData stores the value returned for reads of a pump
	// characteristic when no read handler supplies one.
	SetCharacteristicData(charType CharacteristicType, data []byte)
	// Notify delivers one notification payload on a characteristic. It returns
	// an error when no central is subscribed to that characteristic.
	Notify(charType CharacteristicType, data []byte) error
	// IsConnected reports whether a central is currently connected.
	IsConnected() bool
	// ShutdownConnection drops the current central connection, if any.
	ShutdownConnection()
	// SetPairingState changes the advertised pairing/discoverable state.
	SetPairingState(state PairingState) error
	// GetPairingState returns the current pairing/discoverable state.
	GetPairingState() PairingState
}

// Compile-time assertions that both transports satisfy the interface.
var (
	_ Transport = (*Ble)(nil)
	_ Transport = (*VirtualTransport)(nil)
)

// RadioController is implemented by transports whose radio can be switched off
// and back on without tearing the transport down. The virtual transport
// supports it; the Linux GATT transport does not, so a harness must check the
// assertion before offering the control.
type RadioController interface {
	// SetRadioEnabled turns the radio on or off. Turning it off drops any
	// attached central and refuses new connections.
	SetRadioEnabled(enabled bool)
	// RadioEnabled reports whether the radio is on.
	RadioEnabled() bool
}

// Compile-time assertion that the virtual transport can control its radio.
var _ RadioController = (*VirtualTransport)(nil)

// NotifyFilterer is implemented by transports that can be asked to drop
// individual notification fragments on their way out.
//
// It is the seam fragment-level fault injection needs, and it is deliberately
// below the message layer: the filter sees exactly one BLE notification at a
// time, which is the granularity at which a real link loses data. The virtual
// transport supports it; the Linux GATT transport does not, so a harness must
// check the assertion before offering the control.
type NotifyFilterer interface {
	// SetNotifyFilter installs the filter consulted before each notification
	// leaves the link, or removes it when passed nil. Returning false drops
	// that fragment silently, exactly as a lost BLE notification would be.
	SetNotifyFilter(filter func(charType CharacteristicType, data []byte) bool)
}

// Compile-time assertion that the virtual transport can filter notifications.
var _ NotifyFilterer = (*VirtualTransport)(nil)

// ATT MTU bounds.
//
// A BLE link negotiates an ATT MTU at connection time; every notification
// after that carries at most MTU-3 bytes (one byte of ATT opcode plus a
// two-byte attribute handle). The Bluetooth spec's floor is 23, which every
// peripheral must support and which is what an un-negotiated link uses.
const (
	// DefaultATTMTU is the spec minimum, and the emulator's default: a 23-byte
	// MTU means 20-byte notifications, which is exactly how pumpX2's own
	// Packetize fragments and what the cliparser jar emits. Keeping it as the
	// default means the wire bytes do not move unless a test asks them to.
	DefaultATTMTU = 23
	// MaxATTMTU is the largest ATT MTU the spec allows.
	MaxATTMTU = 517
	// ATTNotificationOverhead is the ATT header on every notification: one
	// opcode byte plus a two-byte attribute handle.
	ATTNotificationOverhead = 3
)

// MaxNotificationBytes returns how many bytes one notification can carry at
// the given ATT MTU.
func MaxNotificationBytes(mtu int) int {
	return mtu - ATTNotificationOverhead
}

// ValidateATTMTU rejects an MTU outside what the spec allows.
func ValidateATTMTU(mtu int) error {
	if mtu < DefaultATTMTU || mtu > MaxATTMTU {
		return fmt.Errorf("ATT MTU %d is outside the permitted range %d-%d", mtu, DefaultATTMTU, MaxATTMTU)
	}
	return nil
}

// MTUNegotiator is implemented by transports whose ATT MTU can be inspected
// and changed.
//
// A real Mobi paired with the official iOS app negotiates a large MTU, so most
// responses arrive as a single notification rather than a run of 20-byte
// fragments. The virtual transport can be told to do either, so a harness can
// exercise a driver's reassembler against both. The Linux GATT transport
// cannot: gatt negotiates its own MTU with the connecting central and does not
// expose it, so a caller must check the assertion before offering the control.
type MTUNegotiator interface {
	// ATTMTU returns the ATT MTU in force on this link.
	ATTMTU() int
	// SetATTMTU changes the ATT MTU, rejecting a value outside the spec range.
	SetATTMTU(mtu int) error
}

// Compile-time assertion that the virtual transport can set its ATT MTU.
var _ MTUNegotiator = (*VirtualTransport)(nil)
