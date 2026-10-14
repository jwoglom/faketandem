package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// HistoryLogResponse acknowledges a HistoryLogRequest: Status 0 means the
// pump will stream the entries on the HISTORY_LOG characteristic under
// StreamID.
type HistoryLogResponse struct {
	messages.Base
	Status   int32 `pumpx2:"status"`
	StreamID int32 `pumpx2:"streamId"`
}

var historyLogResponseProps = messages.Props{OpCode: 61, Size: 2, Type: messages.Response, Request: "HistoryLogRequest"}

func init() {
	messages.Register("HistoryLogResponse", &HistoryLogResponse{}, historyLogResponseProps,
		messages.Constructor{Params: "int status, int streamId", Func: NewHistoryLogResponse},
	)
}

// NewHistoryLogResponse builds the response; each value goes on the wire as
// one byte.
func NewHistoryLogResponse(status, streamID int32) *HistoryLogResponse {
	m := &HistoryLogResponse{Status: status, StreamID: streamID}
	m.SetCargo([]byte{byte(status), byte(streamID)})
	return m
}

// Parse implements messages.Message.
func (m *HistoryLogResponse) Parse(raw []byte) error {
	if len(raw) != historyLogResponseProps.Size {
		return messages.InvalidLength("HistoryLogResponse", len(raw), historyLogResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.StreamID = int32(int8(raw[1]))
	return nil
}

// IsStatusOK reports whether the pump took the request.
func (m *HistoryLogResponse) IsStatusOK() bool { return m.Status == 0 }
