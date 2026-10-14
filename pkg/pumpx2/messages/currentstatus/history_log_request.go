package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// HistoryLogRequest asks the pump to stream NumberOfLogs history log
// entries from sequence number StartLog on the HISTORY_LOG characteristic.
// HistoryLogResponse acknowledges it with the stream's ID.
type HistoryLogRequest struct {
	messages.Base
	StartLog int64 `pumpx2:"startLog"`
	// NumberOfLogs is the count byte read signed, as pumpX2 reads it: a
	// request for 255 entries reads -1. LogCount reads it as the pump does.
	NumberOfLogs int32 `pumpx2:"numberOfLogs"`
}

var historyLogRequestProps = messages.Props{OpCode: 60, Size: 5, Type: messages.Request, Response: "HistoryLogResponse"}

func init() {
	messages.Register("HistoryLogRequest", &HistoryLogRequest{}, historyLogRequestProps,
		messages.Constructor{Params: "long startLog, int numberOfLogs", Func: NewHistoryLogRequest},
	)
}

// NewHistoryLogRequest builds the request; numberOfLogs goes on the wire as
// one byte.
func NewHistoryLogRequest(startLog int64, numberOfLogs int32) *HistoryLogRequest {
	m := &HistoryLogRequest{StartLog: startLog, NumberOfLogs: numberOfLogs}
	m.SetCargo(helpers.Combine(helpers.ToUint32(startLog), []byte{byte(numberOfLogs)}))
	return m
}

// Parse implements messages.Message.
func (m *HistoryLogRequest) Parse(raw []byte) error {
	if len(raw) != historyLogRequestProps.Size {
		return messages.InvalidLength("HistoryLogRequest", len(raw), historyLogRequestProps.Size)
	}
	m.SetCargo(raw)
	m.StartLog = helpers.ReadUint32(raw, 0)
	m.NumberOfLogs = int32(int8(raw[4]))
	return nil
}

// LogCount is how many entries the request asks for, 0 to 255: the count
// byte read unsigned.
func (m *HistoryLogRequest) LogCount() int32 { return int32(uint8(m.NumberOfLogs)) }
