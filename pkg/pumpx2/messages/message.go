// Package messages is a Go implementation of the pumpX2 message library: a Go
// type for every pumpX2 message class, in a package per characteristic
// (authentication, control, controlstream, currentstatus, historylog), and the
// framing, signing, parsing, printing and building they share, which match the
// pumpX2 cliparser jar's byte for byte.
//
// The message packages register their types here when they are initialized.
// Import pkg/pumpx2/messages/all to link every one of them.
package messages

// Message is one pumpX2 message.
type Message interface {
	// Cargo is the payload: what follows the opcode, transaction ID and
	// length on the wire, without a signed message's trailer.
	Cargo() []byte
	// SetCargo sets the payload, without decoding it.
	SetCargo(cargo []byte)
	// Parse decodes a payload into the message, failing where pumpX2's
	// parse(byte[]) throws. A signed message's payload may still carry its
	// 24-byte trailer, which Parse drops (RemoveSignedRequestHmacBytes).
	Parse(raw []byte) error
}

// Base is what every message holds. Message types embed it.
type Base struct {
	cargo []byte
}

// Cargo implements Message. It is nil until a constructor or Parse sets it.
func (b *Base) Cargo() []byte { return b.cargo }

// SetCargo implements Message.
func (b *Base) SetCargo(cargo []byte) { b.cargo = cargo }

// MessageType is pumpX2's MessageType: which way a message travels.
type MessageType string

// The two message types.
const (
	Request  MessageType = "REQUEST"
	Response MessageType = "RESPONSE"
)

// Characteristic is pumpX2's Characteristic: the GATT characteristic a
// message is written or notified on.
type Characteristic string

// pumpX2's characteristics, in declaration order.
const (
	CurrentStatus Characteristic = "CURRENT_STATUS"
	HistoryLog    Characteristic = "HISTORY_LOG"
	Authorization Characteristic = "AUTHORIZATION"
	Control       Characteristic = "CONTROL"
	ControlStream Characteristic = "CONTROL_STREAM"
)

// Characteristics are pumpX2's characteristics, in declaration order.
var Characteristics = []Characteristic{CurrentStatus, HistoryLog, Authorization, Control, ControlStream}

var characteristicUUIDs = map[Characteristic]string{
	CurrentStatus: "7b83fff6-9f77-4e5c-8064-aae2c24838b9",
	HistoryLog:    "7b83fff8-9f77-4e5c-8064-aae2c24838b9",
	Authorization: "7b83fff9-9f77-4e5c-8064-aae2c24838b9",
	Control:       "7b83fffc-9f77-4e5c-8064-aae2c24838b9",
	ControlStream: "7b83fffd-9f77-4e5c-8064-aae2c24838b9",
}

// UUID is the characteristic's GATT UUID, lower case.
func (c Characteristic) UUID() string { return characteristicUUIDs[c] }

// CharacteristicUUID returns the UUID of the characteristic pumpX2 names name.
func CharacteristicUUID(name string) (string, bool) {
	uuid, ok := characteristicUUIDs[Characteristic(name)]
	return uuid, ok
}

// Props is pumpX2's @MessageProps annotation. A field left zero takes the
// annotation's default: CURRENT_STATUS, API_V2_1 and ALL.
type Props struct {
	OpCode                  int
	Size                    int
	VariableSize            bool
	Stream                  bool
	Signed                  bool
	Type                    MessageType
	Characteristic          Characteristic
	ModifiesInsulinDelivery bool
	MinAPI                  string
	SupportedDevices        string
	// Response names a request's response class, and Request a response's
	// request class, by pumpX2 class name ("ApiVersionResponse").
	Response string
	Request  string
}

func (p Props) withDefaults() Props {
	if p.Characteristic == "" {
		p.Characteristic = CurrentStatus
	}
	if p.MinAPI == "" {
		p.MinAPI = "API_V2_1"
	}
	if p.SupportedDevices == "" {
		p.SupportedDevices = "ALL"
	}
	return p
}

// RemoveSignedRequestHmacBytes drops a signed message's 24-byte trailer
// (time since reset and HMAC) from raw, when raw is exactly that much longer
// than the message.
func RemoveSignedRequestHmacBytes(props Props, raw []byte) []byte {
	if props.Signed && len(raw) == props.Size+24 {
		return append([]byte{}, raw[:len(raw)-24]...)
	}
	return raw
}
