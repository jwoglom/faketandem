package protocol

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 is what the Tandem wire protocol specifies
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// Fragment payload sizes, matching pumpX2's Packetize and TandemKit's
// Sources/TandemCore/Messages/Packetizer.swift. These count the message bytes
// carried inside one BLE notification, NOT the notification itself: each
// fragment is [remainingPackets][txId] followed by up to this many bytes, so an
// 18-byte payload chunk goes out as a 20-byte notification (the most a 23-byte
// ATT MTU allows).
//
// Note this is a different convention from GetChunkSize/AssemblePackets in
// packet.go, which treat their chunk size as the whole fragment. The values
// here were verified byte-for-byte against the pumpX2 cliparser jar 1.9.1,
// which emits 18-byte payload chunks for every response it encodes -- on
// CURRENT_STATUS, AUTHORIZATION, HISTORY_LOG and CONTROL alike, signed or not.
const (
	// DefaultMaxChunkPayload is the payload size used for every pump response.
	DefaultMaxChunkPayload = 18
	// ControlMaxChunkPayload is the larger payload size the phone uses when it
	// writes a request to the CONTROL characteristic. The emulator never sends
	// requests, so it exists for completeness and for tests that reproduce a
	// captured client-side packetization.
	ControlMaxChunkPayload = 40
)

// ChunkPayloadForMTU returns the per-fragment payload size an ATT MTU allows:
// the notification carries MTU-3 bytes, of which the first two are this
// protocol's own [remainingPackets][txId] framing.
//
// At the 23-byte spec minimum this is exactly DefaultMaxChunkPayload (18),
// which is what pumpX2's Packetize and the cliparser jar produce. A real Mobi
// paired with the official iOS app negotiates a much larger MTU, so most
// responses arrive in a single notification with remaining=0 -- which both
// receivers handle without any special case, since TandemKit's BTResponseParser
// and pumpX2's PacketArrayList key off that remaining counter and never off a
// fragment's length.
func ChunkPayloadForMTU(mtu int) int {
	payload := bluetooth.MaxNotificationBytes(mtu) - 2
	if payload < DefaultMaxChunkPayload {
		return DefaultMaxChunkPayload
	}
	return payload
}

// RefragmentHex takes a hex-encoded fragment sequence apart and rebuilds it at
// a different per-fragment payload size, preserving the message bytes exactly.
//
// It is how a response the pumpX2 cliparser encoded (always at 18-byte chunks)
// is re-framed for a link that negotiated a larger MTU. A chunkPayload that
// already matches what the message uses returns it untouched, byte for byte.
func RefragmentHex(packetsHex []string, txID uint8, chunkPayload int) ([]string, error) {
	body, err := MessageBodyFromFragmentsHex(packetsHex)
	if err != nil {
		return nil, err
	}

	fragments, err := FragmentMessage(body, txID, chunkPayload)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(fragments))
	for _, f := range fragments {
		out = append(out, hex.EncodeToString(f))
	}
	return out, nil
}

// SignedTrailerLength is the size of the trailer appended to signed messages:
// a 4-byte little-endian timeSinceReset followed by a 20-byte HMAC-SHA1.
const SignedTrailerLength = 24

// maxFragments is the largest number of fragments one message may be split
// into. The receiving side (TandemKit's BTResponseParser and PacketArrayList,
// pumpX2's PacketArrayList) reads the remaining-packet counter as
// data[0] & 0x0F, so a 17th fragment would alias onto "no more packets" and
// silently truncate the message. At 18-byte payload chunks a maximum-size
// message (3-byte header + 255-byte payload + 2-byte CRC) needs 15 fragments,
// so this ceiling is never reached in practice.
const maxFragments = 16

// MessageSpec describes one complete pump message to build.
//
// Cargo is the message body only: the opcode/txId/length header and the CRC are
// added by the encoder, as is the 24-byte trailer when Signed is set.
type MessageSpec struct {
	// Opcode is the message opcode. Response opcodes above 127 (such as
	// HistoryLogStreamResponse's 0x81, which pumpX2 spells as the signed byte
	// -127) are written as-is.
	Opcode uint8
	// TxID echoes the transaction id of the request being answered.
	TxID uint8
	// Cargo is the message-specific payload.
	Cargo []byte
	// Signed requests the 24-byte timestamp+HMAC trailer that the pump appends
	// to messages pumpX2 marks signed=true (most CONTROL responses).
	Signed bool
	// AuthKey is the HMAC key; required when Signed is set.
	//
	// Note that the pumpX2 cliparser uses the raw bytes of the
	// PUMP_AUTHENTICATION_KEY environment string as the key, so reproducing a
	// cliparser-signed message means passing the ASCII bytes of the hex string
	// rather than the decoded key.
	AuthKey []byte
	// TimeSinceReset is the pump's seconds-since-reset counter, written into
	// the first 4 bytes of the signed trailer.
	TimeSinceReset uint32
	// MaxChunkPayload overrides the per-fragment payload size; 0 means
	// DefaultMaxChunkPayload.
	MaxChunkPayload int
}

// NativeMessage is a fully framed pump message built in Go rather than by the
// pumpX2 cliparser, ready to hand to the transport one fragment at a time.
//
// It exists because cliparser cannot encode several messages a real pump sends:
// HistoryLogStreamResponse (its constructor takes a List the JSON bridge cannot
// build), ErrorResponse (no encodable class name) and AlarmStatusResponse (an
// enum varargs constructor), among others.
type NativeMessage struct {
	// MessageType is the pumpX2 message name, for logging only.
	MessageType string
	Opcode      uint8
	TxID        uint8
	// Characteristic is the characteristic the message must be notified on.
	Characteristic bluetooth.CharacteristicType
	// Fragments are the BLE notification payloads, in send order.
	Fragments [][]byte
}

// PacketsHex renders the fragments as hex strings, matching the representation
// pumpx2.EncodedMessage uses.
func (m *NativeMessage) PacketsHex() []string {
	out := make([]string, 0, len(m.Fragments))
	for _, f := range m.Fragments {
		out = append(out, hex.EncodeToString(f))
	}
	return out
}

// BuildMessageBody assembles the framed message bytes for spec: the three-byte
// header (opcode, txId, payload length), the cargo, the optional 24-byte signed
// trailer, and the two CRC bytes. The result is what gets split into fragments;
// it does not include any per-fragment framing.
//
// The signing procedure mirrors Packetize exactly: 24 zero bytes are appended,
// the last 4 bytes before the 20-byte HMAC slot are overwritten with the
// little-endian timeSinceReset, an HMAC-SHA1 is taken over everything up to
// (but not including) that 20-byte slot, and the digest fills it. The CRC is
// then computed over the whole thing, trailer included.
func BuildMessageBody(spec MessageSpec) ([]byte, error) {
	payloadLength := len(spec.Cargo)
	if spec.Signed {
		payloadLength += SignedTrailerLength
	}
	if payloadLength > 255 {
		return nil, fmt.Errorf("payload too large for a single message header: %d bytes", payloadLength)
	}

	msg := make([]byte, 0, 3+payloadLength+2)
	msg = append(msg, spec.Opcode, spec.TxID, byte(payloadLength))
	msg = append(msg, spec.Cargo...)

	if spec.Signed {
		if len(spec.AuthKey) == 0 {
			return nil, fmt.Errorf("signed message requires an authentication key")
		}
		msg = append(msg, make([]byte, SignedTrailerLength)...)

		hmacStart := len(msg) - 20
		binary.LittleEndian.PutUint32(msg[hmacStart-4:hmacStart], spec.TimeSinceReset)

		mac := hmac.New(sha1.New, spec.AuthKey)
		mac.Write(msg[:hmacStart])
		copy(msg[hmacStart:], mac.Sum(nil))
	}

	return append(msg, CalculateCRC16(msg)...), nil
}

// FragmentMessage splits a framed message body into BLE notification payloads.
// Each fragment is [remainingFragments][txId] followed by up to chunkPayload
// message bytes; the remaining counter starts at len(fragments)-1 and counts
// down to 0 on the last fragment.
func FragmentMessage(body []byte, txID uint8, chunkPayload int) ([][]byte, error) {
	if chunkPayload <= 0 {
		chunkPayload = DefaultMaxChunkPayload
	}
	total := (len(body) + chunkPayload - 1) / chunkPayload
	if total == 0 {
		total = 1
	}
	if total > maxFragments {
		return nil, fmt.Errorf("message needs %d fragments, more than the %d the 4-bit remaining counter can express",
			total, maxFragments)
	}

	fragments := make([][]byte, 0, total)
	for i := 0; i < total; i++ {
		start := i * chunkPayload
		end := start + chunkPayload
		if end > len(body) {
			end = len(body)
		}
		fragment := make([]byte, 0, 2+(end-start))
		fragment = append(fragment, byte(total-i-1), txID)
		fragment = append(fragment, body[start:end]...)
		fragments = append(fragments, fragment)
	}
	return fragments, nil
}

// EncodeMessage builds and fragments a message in one step.
func EncodeMessage(spec MessageSpec) ([][]byte, error) {
	body, err := BuildMessageBody(spec)
	if err != nil {
		return nil, err
	}
	return FragmentMessage(body, spec.TxID, spec.MaxChunkPayload)
}

// NewNativeMessage builds a NativeMessage from spec, tagging it with a
// pumpX2 message name and the characteristic it belongs on.
func NewNativeMessage(messageType string, charType bluetooth.CharacteristicType, spec MessageSpec) (*NativeMessage, error) {
	fragments, err := EncodeMessage(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", messageType, err)
	}
	return &NativeMessage{
		MessageType:    messageType,
		Opcode:         spec.Opcode,
		TxID:           spec.TxID,
		Characteristic: charType,
		Fragments:      fragments,
	}, nil
}
