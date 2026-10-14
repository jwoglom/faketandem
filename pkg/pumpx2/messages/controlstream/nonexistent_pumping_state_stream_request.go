package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentPumpingStateStreamRequest is pumpX2's placeholder request for
// PumpingStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentPumpingStateStreamRequest struct {
	messages.Base
}

var nonexistentPumpingStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "PumpingStateStreamResponse"}

func init() {
	messages.Register("NonexistentPumpingStateStreamRequest", &NonexistentPumpingStateStreamRequest{}, nonexistentPumpingStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentPumpingStateStreamRequest},
	)
}

// NewNonexistentPumpingStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentPumpingStateStreamRequest() *NonexistentPumpingStateStreamRequest {
	m := &NonexistentPumpingStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentPumpingStateStreamRequest) Parse([]byte) error { return nil }
