package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentEnterChangeCartridgeModeStateStreamRequest is pumpX2's placeholder request for
// EnterChangeCartridgeModeStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentEnterChangeCartridgeModeStateStreamRequest struct {
	messages.Base
}

var nonexistentEnterChangeCartridgeModeStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "EnterChangeCartridgeModeStateStreamResponse"}

func init() {
	messages.Register("NonexistentEnterChangeCartridgeModeStateStreamRequest", &NonexistentEnterChangeCartridgeModeStateStreamRequest{}, nonexistentEnterChangeCartridgeModeStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentEnterChangeCartridgeModeStateStreamRequest},
	)
}

// NewNonexistentEnterChangeCartridgeModeStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentEnterChangeCartridgeModeStateStreamRequest() *NonexistentEnterChangeCartridgeModeStateStreamRequest {
	m := &NonexistentEnterChangeCartridgeModeStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentEnterChangeCartridgeModeStateStreamRequest) Parse([]byte) error { return nil }
