package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PrimeNudgeStateStreamResponse is an ExitFillTubingModeStateStreamResponse
// streamed during a prime nudge, on the same opcode. pumpX2 parses the opcode
// as ExitFillTubingModeStateStreamResponse, so this type is only built.
type PrimeNudgeStateStreamResponse struct {
	ExitFillTubingModeStateStreamResponse
}

var primeNudgeStateStreamResponseProps = messages.Props{OpCode: -23, Size: 1, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentPrimeNudgeStateStreamRequest"}

func init() {
	messages.Register("PrimeNudgeStateStreamResponse", &PrimeNudgeStateStreamResponse{}, primeNudgeStateStreamResponseProps,
		messages.Constructor{Params: "int stateId", Func: NewPrimeNudgeStateStreamResponse},
	)
}

// NewPrimeNudgeStateStreamResponse builds the response. StateID reads back as
// the low byte of stateID.
func NewPrimeNudgeStateStreamResponse(stateID int32) *PrimeNudgeStateStreamResponse {
	m := &PrimeNudgeStateStreamResponse{}
	m.decodeValid([]byte{byte(stateID)})
	return m
}

// Parse implements messages.Message.
func (m *PrimeNudgeStateStreamResponse) Parse(raw []byte) error {
	return m.decode(primeNudgeStateStreamResponseProps, "PrimeNudgeStateStreamResponse", raw)
}
