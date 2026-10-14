package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentPrimeNudgeStateStreamRequest is pumpX2's placeholder request for
// PrimeNudgeStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentPrimeNudgeStateStreamRequest struct {
	messages.Base
}

var nonexistentPrimeNudgeStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "PrimeNudgeStateStreamResponse"}

func init() {
	messages.Register("NonexistentPrimeNudgeStateStreamRequest", &NonexistentPrimeNudgeStateStreamRequest{}, nonexistentPrimeNudgeStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentPrimeNudgeStateStreamRequest},
	)
}

// NewNonexistentPrimeNudgeStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentPrimeNudgeStateStreamRequest() *NonexistentPrimeNudgeStateStreamRequest {
	m := &NonexistentPrimeNudgeStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentPrimeNudgeStateStreamRequest) Parse([]byte) error { return nil }
