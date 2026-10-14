package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentLoadCartridgeStateStreamRequest is pumpX2's placeholder request for
// LoadCartridgeStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentLoadCartridgeStateStreamRequest struct {
	messages.Base
}

var nonexistentLoadCartridgeStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "LoadCartridgeStateStreamResponse"}

func init() {
	messages.Register("NonexistentLoadCartridgeStateStreamRequest", &NonexistentLoadCartridgeStateStreamRequest{}, nonexistentLoadCartridgeStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentLoadCartridgeStateStreamRequest},
	)
}

// NewNonexistentLoadCartridgeStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentLoadCartridgeStateStreamRequest() *NonexistentLoadCartridgeStateStreamRequest {
	m := &NonexistentLoadCartridgeStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentLoadCartridgeStateStreamRequest) Parse([]byte) error { return nil }
