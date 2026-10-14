package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentFillTubingStateStreamRequest is pumpX2's placeholder request for
// FillTubingStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentFillTubingStateStreamRequest struct {
	messages.Base
}

var nonexistentFillTubingStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "FillTubingStateStreamResponse"}

func init() {
	messages.Register("NonexistentFillTubingStateStreamRequest", &NonexistentFillTubingStateStreamRequest{}, nonexistentFillTubingStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentFillTubingStateStreamRequest},
	)
}

// NewNonexistentFillTubingStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentFillTubingStateStreamRequest() *NonexistentFillTubingStateStreamRequest {
	m := &NonexistentFillTubingStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentFillTubingStateStreamRequest) Parse([]byte) error { return nil }
