package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonexistentFillCannulaStateStreamRequest is pumpX2's placeholder request for
// FillCannulaStateStreamResponse, which the pump streams unasked. Nothing sends
// it, and its opcode, 0, is no real one.
type NonexistentFillCannulaStateStreamRequest struct {
	messages.Base
}

var nonexistentFillCannulaStateStreamRequestProps = messages.Props{OpCode: 0, Stream: true, Signed: true, Type: messages.Request,
	Characteristic: messages.ControlStream, Response: "FillCannulaStateStreamResponse"}

func init() {
	messages.Register("NonexistentFillCannulaStateStreamRequest", &NonexistentFillCannulaStateStreamRequest{}, nonexistentFillCannulaStateStreamRequestProps,
		messages.Constructor{Func: NewNonexistentFillCannulaStateStreamRequest},
	)
}

// NewNonexistentFillCannulaStateStreamRequest is the request, with an empty
// payload.
func NewNonexistentFillCannulaStateStreamRequest() *NonexistentFillCannulaStateStreamRequest {
	m := &NonexistentFillCannulaStateStreamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. As pumpX2's does, it reads nothing and
// keeps the payload the request was made with.
func (m *NonexistentFillCannulaStateStreamRequest) Parse([]byte) error { return nil }
