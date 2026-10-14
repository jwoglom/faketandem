package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// HistoryLogStatusRequest asks how many entries the pump's history log holds
// and the range of their sequence numbers.
type HistoryLogStatusRequest struct {
	messages.Base
}

var historyLogStatusRequestProps = messages.Props{OpCode: 58, Type: messages.Request, Response: "HistoryLogStatusResponse"}

func init() {
	messages.Register("HistoryLogStatusRequest", &HistoryLogStatusRequest{}, historyLogStatusRequestProps,
		messages.Constructor{Func: NewHistoryLogStatusRequest},
	)
}

// NewHistoryLogStatusRequest is the request, which has no payload.
func NewHistoryLogStatusRequest() *HistoryLogStatusRequest {
	m := &HistoryLogStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *HistoryLogStatusRequest) Parse(raw []byte) error {
	if len(raw) != historyLogStatusRequestProps.Size {
		return messages.InvalidLength("HistoryLogStatusRequest", len(raw), historyLogStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
