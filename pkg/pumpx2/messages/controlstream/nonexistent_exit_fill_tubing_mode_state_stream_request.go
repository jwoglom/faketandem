package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentExitFillTubingModeStateStreamRequest is pumpX2's placeholder request for
// ExitFillTubingModeStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentExitFillTubingModeStateStreamRequest struct {
	messages.Base
}

var nonexistentExitFillTubingModeStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "ExitFillTubingModeStateStreamResponse"}

func init() {
	messages.Register("NonexistentExitFillTubingModeStateStreamRequest", &NonexistentExitFillTubingModeStateStreamRequest{}, nonexistentExitFillTubingModeStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentExitFillTubingModeStateStreamRequest},
	)
}

// NewNonexistentExitFillTubingModeStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentExitFillTubingModeStateStreamRequest() *NonexistentExitFillTubingModeStateStreamRequest {
	m := &NonexistentExitFillTubingModeStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentExitFillTubingModeStateStreamRequest) Parse([]byte) error { return nil }
