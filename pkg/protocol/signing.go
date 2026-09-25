package protocol

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 is what the Tandem wire protocol specifies
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// MessageBody is a framed pump message taken apart again: the three-byte
// header, the payload it declares, and the trailing CRC16.
type MessageBody struct {
	Opcode uint8
	TxID   uint8
	// Payload is the declared payload, which for a signed message includes
	// the 24-byte trailer at its end.
	Payload []byte
	// CRC is the two CRC16 bytes as they appeared on the wire.
	CRC []byte
}

// Cargo returns the message-specific body: the payload with the signed trailer
// removed when signed is set.
func (b MessageBody) Cargo(signed bool) ([]byte, error) {
	if !signed {
		return b.Payload, nil
	}
	if len(b.Payload) < SignedTrailerLength {
		return nil, fmt.Errorf("payload is %d bytes, too short to hold the %d-byte signed trailer",
			len(b.Payload), SignedTrailerLength)
	}
	return b.Payload[:len(b.Payload)-SignedTrailerLength], nil
}

// MessageBodyFromFragmentsHex undoes the per-fragment [remaining][txId] framing
// of a hex-encoded BLE notification sequence and returns the message bytes.
//
// It is the inverse of FragmentMessage, and exists so a message that came back
// from the pumpX2 cliparser already framed can be taken apart and rebuilt --
// which is what re-signing a cliparser-encoded response requires.
func MessageBodyFromFragmentsHex(packetsHex []string) ([]byte, error) {
	if len(packetsHex) == 0 {
		return nil, fmt.Errorf("no fragments to reassemble")
	}

	var body []byte
	for i, packetHex := range packetsHex {
		fragment, err := hex.DecodeString(packetHex)
		if err != nil {
			return nil, fmt.Errorf("fragment %d is not hex: %w", i, err)
		}
		if len(fragment) < 2 {
			return nil, fmt.Errorf("fragment %d is %d bytes, too short to carry its framing", i, len(fragment))
		}
		body = append(body, fragment[2:]...)
	}
	return body, nil
}

// ParseMessageBody splits a framed message into its header, payload and CRC,
// rejecting anything whose declared payload length does not account for every
// byte present. That strictness is deliberate: this is used to re-derive the
// cargo of a message somebody else encoded, and a silent mis-slice there would
// put a plausible-looking but wrong message on the wire.
func ParseMessageBody(body []byte) (MessageBody, error) {
	if len(body) < 5 {
		return MessageBody{}, fmt.Errorf("message is %d bytes, too short to be framed", len(body))
	}

	payloadLength := int(body[2])
	if want := 3 + payloadLength + 2; len(body) != want {
		return MessageBody{}, fmt.Errorf("message declares a %d-byte payload, which needs %d bytes in all, but it is %d",
			payloadLength, want, len(body))
	}

	return MessageBody{
		Opcode:  body[0],
		TxID:    body[1],
		Payload: body[3 : 3+payloadLength],
		CRC:     body[3+payloadLength:],
	}, nil
}

// ResignFragmentsHex rebuilds the signed trailer of an already-framed message
// with the given key and timeSinceReset, returning the re-fragmented result.
//
// The cargo is taken verbatim from the input, so whatever encoded the message
// in the first place stays responsible for it; only the last 24 payload bytes
// and the CRC are recomputed. The fragment size is unchanged from
// FragmentMessage's default, which is what cliparser and the real pump both
// use, so a message whose trailer happens to be identical comes back
// byte-identical.
func ResignFragmentsHex(packetsHex []string, authKey []byte, timeSinceReset uint32) ([]string, error) {
	body, err := MessageBodyFromFragmentsHex(packetsHex)
	if err != nil {
		return nil, err
	}
	parsed, err := ParseMessageBody(body)
	if err != nil {
		return nil, err
	}
	cargo, err := parsed.Cargo(true)
	if err != nil {
		return nil, err
	}

	fragments, err := EncodeMessage(MessageSpec{
		Opcode:         parsed.Opcode,
		TxID:           parsed.TxID,
		Cargo:          cargo,
		Signed:         true,
		AuthKey:        authKey,
		TimeSinceReset: timeSinceReset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(fragments))
	for _, fragment := range fragments {
		out = append(out, hex.EncodeToString(fragment))
	}
	return out, nil
}

// SignedTrailer is the 24 bytes a signed message carries at the end of its
// payload.
type SignedTrailer struct {
	// TimeSinceReset is the pump's seconds-since-reset counter, as written
	// into the four bytes preceding the digest.
	TimeSinceReset uint32
	// HMAC is the 20-byte HMAC-SHA1 digest as it appeared on the wire.
	HMAC []byte
}

// VerifySignedMessage recomputes the HMAC-SHA1 over a framed message body and
// reports whether the trailer it carries is the one authKey produces.
//
// This is the checking half of the procedure BuildMessageBody performs, written
// out independently rather than by calling the builder, so that a mistake in
// the builder cannot verify itself: the digest is taken over every byte of the
// message up to (but not including) the 20-byte digest slot, with the
// little-endian timeSinceReset sitting in the four bytes immediately before it.
// That is exactly what TandemKit's Packetize does.
func VerifySignedMessage(body []byte, authKey []byte) (SignedTrailer, bool, error) {
	parsed, err := ParseMessageBody(body)
	if err != nil {
		return SignedTrailer{}, false, err
	}
	if len(parsed.Payload) < SignedTrailerLength {
		return SignedTrailer{}, false, fmt.Errorf("payload is %d bytes, too short to hold the %d-byte signed trailer",
			len(parsed.Payload), SignedTrailerLength)
	}
	if len(authKey) == 0 {
		return SignedTrailer{}, false, fmt.Errorf("no authentication key to verify against")
	}

	// The signed region is the header plus the payload up to the digest slot;
	// the CRC is computed afterwards and is not covered.
	signedEnd := 3 + len(parsed.Payload) - 20
	trailer := SignedTrailer{
		TimeSinceReset: binary.LittleEndian.Uint32(body[signedEnd-4 : signedEnd]),
		HMAC:           body[signedEnd : signedEnd+20],
	}

	mac := hmac.New(sha1.New, authKey)
	mac.Write(body[:signedEnd])
	expected := mac.Sum(nil)

	return trailer, hmac.Equal(expected, trailer.HMAC), nil
}

// VerifySignedFragmentsHex is VerifySignedMessage over a hex fragment
// sequence, and additionally checks the CRC, so a test can assert on exactly
// the bytes a central received.
func VerifySignedFragmentsHex(packetsHex []string, authKey []byte) (SignedTrailer, bool, error) {
	body, err := MessageBodyFromFragmentsHex(packetsHex)
	if err != nil {
		return SignedTrailer{}, false, err
	}
	parsed, err := ParseMessageBody(body)
	if err != nil {
		return SignedTrailer{}, false, err
	}
	if want := CalculateCRC16(body[:3+len(parsed.Payload)]); !bytes.Equal(want, parsed.CRC) {
		return SignedTrailer{}, false, fmt.Errorf("CRC is %x, want %x", parsed.CRC, want)
	}
	return VerifySignedMessage(body, authKey)
}
