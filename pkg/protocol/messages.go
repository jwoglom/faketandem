package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
)

// Opcodes for the messages this package builds natively. Each matches the
// pumpX2 @MessageProps(opCode=...) value for the message, cross-checked against
// TandemKit's MessageProps declarations.
const (
	// OpcodeAlarmStatusResponse is AlarmStatusResponse (CURRENT_STATUS).
	OpcodeAlarmStatusResponse uint8 = 71
	// OpcodeReminderStatusResponse is ReminderStatusResponse (CURRENT_STATUS).
	OpcodeReminderStatusResponse uint8 = 73
	// OpcodeErrorResponse is ErrorResponse (CURRENT_STATUS). pumpX2 and
	// TandemKit declare it unsigned and variable-size: 2 bytes, or 26 with a
	// signature trailer.
	OpcodeErrorResponse uint8 = 77
	// OpcodeHistoryLogStreamResponse is HistoryLogStreamResponse
	// (HISTORY_LOG). pumpX2 spells it as the signed byte -127, which is 0x81.
	OpcodeHistoryLogStreamResponse uint8 = 0x81
	// OpcodeFillTubingStateStreamResponse is FillTubingStateStreamResponse
	// (CONTROL_STREAM), the signed byte -27.
	OpcodeFillTubingStateStreamResponse uint8 = 0xE5
	// OpcodeEnterChangeCartridgeModeStateStreamResponse is
	// EnterChangeCartridgeModeStateStreamResponse (CONTROL_STREAM), the signed byte -31.
	OpcodeEnterChangeCartridgeModeStateStreamResponse uint8 = 0xE1
	// OpcodeDetectingCartridgeStateStreamResponse is DetectingCartridgeStateStreamResponse
	// (CONTROL_STREAM), the signed byte -29.
	OpcodeDetectingCartridgeStateStreamResponse uint8 = 0xE3
	// OpcodeFillCannulaStateStreamResponse is FillCannulaStateStreamResponse (CONTROL_STREAM),
	// the signed byte -25.
	OpcodeFillCannulaStateStreamResponse uint8 = 0xE7
	// OpcodeExitFillTubingModeStateStreamResponse is ExitFillTubingModeStateStreamResponse
	// (CONTROL_STREAM), the signed byte -23.
	OpcodeExitFillTubingModeStateStreamResponse uint8 = 0xE9
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

// BuildReminderStatusResponse builds a ReminderStatusResponse carrying the
// pump's active-reminder bitmask, little-endian, bit N being ReminderType N.
// Its only data constructor takes a BigInteger, which neither cliparser nor
// the Go codec builds from a JSON number below 2^63.
func BuildReminderStatusResponse(txID uint8, reminderBitmask uint64) (*NativeMessage, error) {
	cargo := make([]byte, 8)
	binary.LittleEndian.PutUint64(cargo, reminderBitmask)

	return NewNativeMessage("ReminderStatusResponse", bluetooth.CharCurrentStatus, MessageSpec{
		Opcode: OpcodeReminderStatusResponse,
		TxID:   txID,
		Cargo:  cargo,
	})
}

// ErrorResponseOptions controls optional parts of an ErrorResponse.
type ErrorResponseOptions struct {
	// ExtraBytes are appended after the two-byte [requestCodeId][errorCodeId]
	// cargo; the driver exposes them as ErrorResponse.remainingBytes.
	ExtraBytes []byte
	// Signed appends a 24-byte signature trailer, the 26-byte form pumpX2's
	// PacketArrayList and TandemKit's ErrorResponse.containsSignature accept,
	// though both declare the message unsigned. It is off by default because
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

// BuildFillTubingStateStreamResponse builds the signed message the pump
// streams on CONTROL_STREAM while fill tubing mode is open, each time its
// button goes down or comes back up. txID is the EnterFillTubingModeRequest's,
// which the pump streams every one under; the cargo is the button state, 1
// down and 0 up.
func BuildFillTubingStateStreamResponse(txID uint8, buttonDown bool, authKey []byte, timeSinceReset uint32) (*NativeMessage, error) {
	var buttonState byte
	if buttonDown {
		buttonState = 1
	}
	return NewNativeMessage("FillTubingStateStreamResponse", bluetooth.CharControlStream, MessageSpec{
		Opcode:         OpcodeFillTubingStateStreamResponse,
		TxID:           txID,
		Cargo:          []byte{buttonState},
		Signed:         true,
		AuthKey:        authKey,
		TimeSinceReset: timeSinceReset,
	})
}

func buildControlStream(name string, opcode, txID uint8, cargo, authKey []byte, timeSinceReset uint32) (*NativeMessage, error) {
	return NewNativeMessage(name, bluetooth.CharControlStream, MessageSpec{
		Opcode:         opcode,
		TxID:           txID,
		Cargo:          cargo,
		Signed:         true,
		AuthKey:        authKey,
		TimeSinceReset: timeSinceReset,
	})
}

// BuildEnterChangeCartridgeModeStateStreamResponse builds the signed message the pump streams
// under the EnterChangeCartridgeModeRequest's txID as it gets ready for the cartridge to come out.
// pumpX2 names only stateID 2, READY_TO_CHANGE.
func BuildEnterChangeCartridgeModeStateStreamResponse(txID uint8, stateID int32, authKey []byte, timeSinceReset uint32) (*NativeMessage, error) {
	cargo := controlstream.NewEnterChangeCartridgeModeStateStreamResponse(stateID).Cargo()
	return buildControlStream("EnterChangeCartridgeModeStateStreamResponse", OpcodeEnterChangeCartridgeModeStateStreamResponse,
		txID, cargo, authKey, timeSinceReset)
}

// BuildDetectingCartridgeStateStreamResponse builds the signed message the pump streams while it
// checks a new cartridge, its two-byte progress climbing to 100.
func BuildDetectingCartridgeStateStreamResponse(txID uint8, percentComplete int32, authKey []byte, timeSinceReset uint32) (*NativeMessage, error) {
	cargo := controlstream.NewDetectingCartridgeStateStreamResponse(percentComplete).Cargo()
	return buildControlStream("DetectingCartridgeStateStreamResponse", OpcodeDetectingCartridgeStateStreamResponse,
		txID, cargo, authKey, timeSinceReset)
}

// BuildFillCannulaStateStreamResponse builds the signed message the pump streams under the
// FillCannulaRequest's txID. pumpX2 names only stateID 2, CANNULA_FILLED.
func BuildFillCannulaStateStreamResponse(txID uint8, stateID int32, authKey []byte, timeSinceReset uint32) (*NativeMessage, error) {
	cargo := controlstream.NewFillCannulaStateStreamResponse(stateID).Cargo()
	return buildControlStream("FillCannulaStateStreamResponse", OpcodeFillCannulaStateStreamResponse,
		txID, cargo, authKey, timeSinceReset)
}

// BuildExitFillTubingModeStateStreamResponse builds the one-byte form of the signed message the
// pump streams under the ExitFillTubingModeRequest's txID: 0 NOT_COMPLETE, 1 TUBING_FILLED.
func BuildExitFillTubingModeStateStreamResponse(txID uint8, stateID int32, authKey []byte, timeSinceReset uint32) (*NativeMessage, error) {
	cargo := controlstream.NewExitFillTubingModeStateStreamResponse(stateID).Cargo()
	return buildControlStream("ExitFillTubingModeStateStreamResponse", OpcodeExitFillTubingModeStateStreamResponse,
		txID, cargo, authKey, timeSinceReset)
}
