package historylog

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentHistoryLogStreamRequest is pumpX2's placeholder request for
// HistoryLogStreamResponse, which the pump streams in answer to a
// HistoryLogRequest on another characteristic. Nothing sends it, and its
// opcode, 0, is no real one.
type NonexistentHistoryLogStreamRequest struct {
	messages.Base
}

var nonexistentHistoryLogStreamRequestProps = messages.Props{OpCode: 0, Type: messages.Request, Characteristic: messages.HistoryLog, Response: "HistoryLogStreamResponse"}

func init() {
	messages.Register("NonexistentHistoryLogStreamRequest", &NonexistentHistoryLogStreamRequest{}, nonexistentHistoryLogStreamRequestProps,
		messages.Constructor{Func: NewNonexistentHistoryLogStreamRequest},
	)
}

// NewNonexistentHistoryLogStreamRequest is the request, with an empty
// payload.
func NewNonexistentHistoryLogStreamRequest() *NonexistentHistoryLogStreamRequest {
	m := &NonexistentHistoryLogStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *NonexistentHistoryLogStreamRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
