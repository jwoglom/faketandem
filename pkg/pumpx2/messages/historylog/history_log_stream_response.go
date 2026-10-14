package historylog

import (
	"encoding/hex"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

const historyLogRecordSize = 26

// HistoryLogStreamResponse is one batch of the history log records the pump
// streams on HISTORY_LOG after a HistoryLogRequest: NumberOfHistoryLogs
// records of 26 bytes, under the stream ID StreamID.
//
// pumpX2 decodes each record into one of its history log classes, which are
// not ported, so HistoryLogs keeps the records raw. That makes its toString
// differ from pumpX2's for a stream that carries any record, and lets the
// constructor and Parse accept a record pumpX2 would fail to decode.
type HistoryLogStreamResponse struct {
	messages.Base
	NumberOfHistoryLogs int32              `pumpx2:"numberOfHistoryLogs"`
	StreamID            int32              `pumpx2:"streamId"`
	HistoryLogs         []HistoryLogRecord `pumpx2:"historyLogs"`
}

var historyLogStreamResponseProps = messages.Props{OpCode: -127, Size: 28, VariableSize: true, Stream: true, Type: messages.Response,
	Characteristic: messages.HistoryLog, Request: "NonexistentHistoryLogStreamRequest"}

func init() {
	messages.Register("HistoryLogStreamResponse", &HistoryLogStreamResponse{}, historyLogStreamResponseProps,
		messages.Constructor{Params: "int numberOfHistoryLogs, int streamId, List<byte[]> historyLogStreamBytes", Func: NewHistoryLogStreamResponse},
	)
}

// NewHistoryLogStreamResponse builds the response from its records. The
// count and stream ID are written as their low bytes, and the count is not
// checked against the records.
func NewHistoryLogStreamResponse(numberOfHistoryLogs, streamID int32, historyLogStreamBytes [][]byte) *HistoryLogStreamResponse {
	m := &HistoryLogStreamResponse{NumberOfHistoryLogs: numberOfHistoryLogs, StreamID: streamID, HistoryLogs: make([]HistoryLogRecord, len(historyLogStreamBytes))}
	cargo := []byte{byte(numberOfHistoryLogs), byte(streamID)}
	for i, record := range historyLogStreamBytes {
		m.HistoryLogs[i] = record
		cargo = append(cargo, record...)
	}
	m.SetCargo(cargo)
	return m
}

// Parse implements messages.Message. The count is signed, so a stream
// claiming more than 127 records never parses.
func (m *HistoryLogStreamResponse) Parse(raw []byte) error {
	if len(raw) < 2 {
		return messages.Invalid("HistoryLogStreamResponse: payload of %d bytes has no record count and stream ID", len(raw))
	}
	numberOfHistoryLogs := int32(int8(raw[0]))
	if want := int(numberOfHistoryLogs)*historyLogRecordSize + 2; len(raw) != want {
		return messages.Invalid("HistoryLogStreamResponse: %d records need %d bytes, not %d", numberOfHistoryLogs, want, len(raw))
	}
	m.SetCargo(raw)
	m.NumberOfHistoryLogs = numberOfHistoryLogs
	m.StreamID = int32(int8(raw[1]))
	m.HistoryLogs = make([]HistoryLogRecord, 0, numberOfHistoryLogs)
	for i := 2; i < len(raw); i += historyLogRecordSize {
		m.HistoryLogs = append(m.HistoryLogs, helpers.CopyOfRange(raw, i, i+historyLogRecordSize))
	}
	return nil
}

// HistoryLogRecord is one history log record, undecoded: its first two bytes,
// little-endian, hold its type ID in their low 12 bits.
type HistoryLogRecord []byte

// TypeID is the record's history log type ID.
func (r HistoryLogRecord) TypeID() int32 { return helpers.ReadShort(r, 0) & 4095 }

func (r HistoryLogRecord) String() string { return hex.EncodeToString(r) }
