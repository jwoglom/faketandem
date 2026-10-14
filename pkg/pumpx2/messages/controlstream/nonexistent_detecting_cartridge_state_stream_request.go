package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentDetectingCartridgeStateStreamRequest is pumpX2's placeholder request for
// DetectingCartridgeStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentDetectingCartridgeStateStreamRequest struct {
	messages.Base
}

var nonexistentDetectingCartridgeStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "DetectingCartridgeStateStreamResponse"}

func init() {
	messages.Register("NonexistentDetectingCartridgeStateStreamRequest", &NonexistentDetectingCartridgeStateStreamRequest{}, nonexistentDetectingCartridgeStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentDetectingCartridgeStateStreamRequest},
	)
}

// NewNonexistentDetectingCartridgeStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentDetectingCartridgeStateStreamRequest() *NonexistentDetectingCartridgeStateStreamRequest {
	m := &NonexistentDetectingCartridgeStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentDetectingCartridgeStateStreamRequest) Parse([]byte) error { return nil }
