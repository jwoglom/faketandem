package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// ErrorResponseEncoderFunc builds the protocol-level ErrorResponse (opcode 77)
// that the error_response fault sends in place of a handler's own answer.
//
// txID is the transaction the error answers, requestOpcode/requestName
// identify what it answers, and errorCode is the code the fault carries.
type ErrorResponseEncoderFunc func(txID int, errorCode int, requestOpcode int, requestName string) (*pumpx2.EncodedMessage, error)

// ErrorResponseEncoder is the encoder the response path uses for the
// error_response fault. It defaults to NativeErrorResponseEncoder and is a
// variable only so a test can swap it out (or clear it, to exercise the
// degraded path below).
//
// It has to be a hook into the native encoder rather than a cliparser call:
// pumpX2 has no encodable ErrorResponse message class, so the bytes can only
// be framed in Go (header, cargo, CRC16). Setting this to nil makes an armed
// error_response fault behave as a drop and say so in the request log and the
// emulator log, so a harness never silently believes it exercised an error
// path it did not.
var ErrorResponseEncoder ErrorResponseEncoderFunc = NativeErrorResponseEncoder

// NativeErrorResponseEncoder builds the ErrorResponse with the Go encoder in
// pkg/protocol.
//
// The response is left unsigned. pumpX2 and TandemKit both mark ErrorResponse
// signed, but the emulator has no pump-side HMAC key before authentication --
// which is exactly when a rejected request is most likely -- and an unsigned
// ErrorResponse fits in a single fragment, which is what the driver's
// ErrorResponse path actually reads (TandemPeripheralManager parses the first
// fragment of a CURRENT_STATUS opcode-77 notification directly rather than
// reassembling). See protocol.BuildErrorResponse.
//
// requestName is not encoded: the wire message identifies the rejected request
// by opcode only. It is accepted so the signature can carry it for logging.
func NativeErrorResponseEncoder(txID int, errorCode int, requestOpcode int, _ string) (*pumpx2.EncodedMessage, error) {
	if err := checkByteRange("txID", txID); err != nil {
		return nil, err
	}
	if err := checkByteRange("error code", errorCode); err != nil {
		return nil, err
	}
	if err := checkByteRange("request opcode", requestOpcode); err != nil {
		return nil, err
	}

	msg, err := protocol.BuildErrorResponse(
		uint8(txID), uint8(requestOpcode), uint8(errorCode), protocol.ErrorResponseOptions{})
	if err != nil {
		return nil, err
	}

	return &pumpx2.EncodedMessage{
		MessageType:    msg.MessageType,
		TxID:           int(msg.TxID),
		Opcode:         int(msg.Opcode),
		Characteristic: bluetooth.CharCurrentStatus.String(),
		Packets:        msg.PacketsHex(),
	}, nil
}

// checkByteRange rejects a value the single-byte wire fields cannot hold,
// rather than letting a uint8 conversion wrap it into a different message.
//
// The opcode range is signed-extended on purpose: pumpX2 spells response
// opcodes above 127 as negative bytes (HistoryLogStreamResponse is -127), and
// a fault armed against such a message reports the opcode the same way.
func checkByteRange(what string, value int) error {
	if value < -128 || value > 255 {
		return fmt.Errorf("%s %d does not fit in the one byte the wire format has for it", what, value)
	}
	return nil
}
