package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PumpingStateStreamResponse is the five-byte form of
// ExitFillTubingModeStateStreamResponse, on the same opcode: whether the
// pumping state has been set since start-up, and the PumpingState bits in
// StateBitmask. pumpX2 parses the opcode as
// ExitFillTubingModeStateStreamResponse, so this type is only built.
type PumpingStateStreamResponse struct {
	ExitFillTubingModeStateStreamResponse
}

var pumpingStateStreamResponseProps = messages.Props{OpCode: -23, Size: 5, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentPumpingStateStreamRequest"}

func init() {
	messages.Register("PumpingStateStreamResponse", &PumpingStateStreamResponse{}, pumpingStateStreamResponseProps,
		messages.Constructor{Params: "boolean isPumpingStateSetAfterStartUp, long stateBitmask", Func: NewPumpingStateStreamResponse},
	)
}

// NewPumpingStateStreamResponse builds the response. StateBitmask reads back
// as its low four bytes.
func NewPumpingStateStreamResponse(isPumpingStateSetAfterStartUp bool, stateBitmask int64) *PumpingStateStreamResponse {
	m := &PumpingStateStreamResponse{}
	m.decodeValid(pumpingStateCargo(isPumpingStateSetAfterStartUp, stateBitmask))
	return m
}

// Parse implements messages.Message.
func (m *PumpingStateStreamResponse) Parse(raw []byte) error {
	return m.decode(pumpingStateStreamResponseProps, "PumpingStateStreamResponse", raw)
}
