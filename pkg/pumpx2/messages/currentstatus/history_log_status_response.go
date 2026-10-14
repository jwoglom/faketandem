package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// HistoryLogStatusResponse is how many entries the pump's history log holds
// and the sequence numbers of its first and last.
type HistoryLogStatusResponse struct {
	messages.Base
	NumEntries       int64 `pumpx2:"numEntries"`
	FirstSequenceNum int64 `pumpx2:"firstSequenceNum"`
	LastSequenceNum  int64 `pumpx2:"lastSequenceNum"`
}

var historyLogStatusResponseProps = messages.Props{OpCode: 59, Size: 12, Type: messages.Response, Request: "HistoryLogStatusRequest"}

func init() {
	messages.Register("HistoryLogStatusResponse", &HistoryLogStatusResponse{}, historyLogStatusResponseProps,
		messages.Constructor{Params: "long numEntries, long firstSequenceNum, long lastSequenceNum", Func: NewHistoryLogStatusResponse},
	)
}

// NewHistoryLogStatusResponse builds the response; each value goes on the
// wire as a uint32.
func NewHistoryLogStatusResponse(numEntries, firstSequenceNum, lastSequenceNum int64) *HistoryLogStatusResponse {
	m := &HistoryLogStatusResponse{NumEntries: numEntries, FirstSequenceNum: firstSequenceNum, LastSequenceNum: lastSequenceNum}
	m.SetCargo(helpers.Combine(helpers.ToUint32(numEntries), helpers.ToUint32(firstSequenceNum), helpers.ToUint32(lastSequenceNum)))
	return m
}

// Parse implements messages.Message.
func (m *HistoryLogStatusResponse) Parse(raw []byte) error {
	if len(raw) != historyLogStatusResponseProps.Size {
		return messages.InvalidLength("HistoryLogStatusResponse", len(raw), historyLogStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.NumEntries = helpers.ReadUint32(raw, 0)
	m.FirstSequenceNum = helpers.ReadUint32(raw, 4)
	m.LastSequenceNum = helpers.ReadUint32(raw, 8)
	return nil
}
