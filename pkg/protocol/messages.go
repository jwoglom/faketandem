package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// Opcodes for the messages this package builds natively. Each matches the
// pumpX2 @MessageProps(opCode=...) value for the message, cross-checked against
// TandemKit's MessageProps declarations.
const (
	// OpcodeAlarmStatusResponse is AlarmStatusResponse (CURRENT_STATUS).
	OpcodeAlarmStatusResponse uint8 = 71
	// OpcodeErrorResponse is ErrorResponse (CURRENT_STATUS). pumpX2 declares it
	// signed and variable-size.
	OpcodeErrorResponse uint8 = 77
	// OpcodeHistoryLogStreamResponse is HistoryLogStreamResponse
	// (HISTORY_LOG). pumpX2 spells it as the signed byte -127, which is 0x81.
	OpcodeHistoryLogStreamResponse uint8 = 0x81
)

// HistoryLogRecordLength is the fixed size of one history log record inside a
// HistoryLogStreamResponse.
const HistoryLogRecordLength = 26

// BuildAlarmStatusResponse builds an AlarmStatusResponse carrying the pump's
// active-alarm bitmask.
//
// The cargo is the 64-bit mask, little-endian; bit N corresponds to
// AlarmStatusResponse.AlarmResponseType with raw value N (bit 3 PUMP_RESET_ALARM,
// bit 18 RESUME_PUMP_ALARM, and so on).
//
// This has to be built here because the cliparser cannot construct it: the real
// "data" constructor takes an AlarmResponseType varargs array, which the JSON
// bridge has no way to express, so cliparser falls back to the no-arg
// constructor and emits an empty (zero-length) cargo -- which is not even a
// well-formed AlarmStatusResponse, since the driver reads 8 bytes from it.
func BuildAlarmStatusResponse(txID uint8, alarmBitmask uint64) (*NativeMessage, error) {
	cargo := make([]byte, 8)
	binary.LittleEndian.PutUint64(cargo, alarmBitmask)

	return NewNativeMessage("AlarmStatusResponse", bluetooth.CharCurrentStatus, MessageSpec{
		Opcode: OpcodeAlarmStatusResponse,
		TxID:   txID,
		Cargo:  cargo,
	})
}

// ErrorResponseOptions controls optional parts of an ErrorResponse.
type ErrorResponseOptions struct {
	// ExtraBytes are appended after the two-byte [requestCodeId][errorCodeId]
	// cargo; the driver exposes them as ErrorResponse.remainingBytes.
	ExtraBytes []byte
	// Signed appends the 24-byte trailer the real pump adds to this message
	// (pumpX2 marks ErrorResponse signed=true). It is off by default because
	// the emulator usually has no pump-side HMAC key to sign with, and because
	// an unsigned ErrorResponse fits in a single fragment -- which is what the
	// driver's ErrorResponse path reads (it parses the first fragment directly
	// rather than reassembling, see TandemPeripheralManager's
	// CURRENT_STATUS/opcode-77 branch).
	Signed         bool
	AuthKey        []byte
	TimeSinceReset uint32
}

// BuildErrorResponse builds the ErrorResponse a pump returns in place of the
// response to a request it rejects.
//
// requestOpcode is the opcode of the request being rejected and errorCode is a
// pumpX2 PumpErrorCode / PumpFaultCode ordinal. The cargo is simply those two
// bytes.
//
// cliparser cannot encode this message at all ("Unknown message name"), so it
// only exists in Go.
func BuildErrorResponse(txID, requestOpcode, errorCode uint8, opts ErrorResponseOptions) (*NativeMessage, error) {
	cargo := make([]byte, 0, 2+len(opts.ExtraBytes))
	cargo = append(cargo, requestOpcode, errorCode)
	cargo = append(cargo, opts.ExtraBytes...)

	return NewNativeMessage("ErrorResponse", bluetooth.CharCurrentStatus, MessageSpec{
		Opcode:         OpcodeErrorResponse,
		TxID:           txID,
		Cargo:          cargo,
		Signed:         opts.Signed,
		AuthKey:        opts.AuthKey,
		TimeSinceReset: opts.TimeSinceReset,
	})
}

// BuildHistoryLogStreamResponse builds one HistoryLogStreamResponse carrying
// the given 26-byte history log records.
//
// Cargo layout is [numberOfHistoryLogs][streamId] followed by the records back
// to back. numberOfHistoryLogs is the count in *this* message, not the total
// being streamed -- pumpX2 declares the message's nominal size as 28 (1 + 1 +
// one record), and every real capture carries exactly one record per message,
// so callers should normally pass a single record and let the driver
// reassemble the stream by sequence number.
//
// cliparser cannot encode this message: `encode HistoryLogStreamResponse`
// throws ClassCastException (JSONArray to List) on the records argument.
func BuildHistoryLogStreamResponse(txID, streamID uint8, records [][]byte) (*NativeMessage, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("history log stream response needs at least one record")
	}
	if len(records) > 255 {
		return nil, fmt.Errorf("history log stream response cannot carry %d records", len(records))
	}

	cargo := make([]byte, 0, 2+len(records)*HistoryLogRecordLength)
	cargo = append(cargo, byte(len(records)), streamID)
	for i, record := range records {
		if len(record) != HistoryLogRecordLength {
			return nil, fmt.Errorf("history log record %d is %d bytes, expected %d",
				i, len(record), HistoryLogRecordLength)
		}
		cargo = append(cargo, record...)
	}

	return NewNativeMessage("HistoryLogStreamResponse", bluetooth.CharHistoryLog, MessageSpec{
		Opcode: OpcodeHistoryLogStreamResponse,
		TxID:   txID,
		Cargo:  cargo,
	})
}
