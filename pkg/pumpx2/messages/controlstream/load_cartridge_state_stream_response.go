package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LoadCartridgeStateStreamResponse is the one-byte form of
// DetectingCartridgeStateStreamResponse, on the same opcode: a cartridge
// loading state in StateID. pumpX2 parses the opcode as
// DetectingCartridgeStateStreamResponse, so this type is only built.
type LoadCartridgeStateStreamResponse struct {
	DetectingCartridgeStateStreamResponse
}

var loadCartridgeStateStreamResponseProps = messages.Props{OpCode: -29, Size: 1, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentLoadCartridgeStateStreamRequest"}

func init() {
	messages.Register("LoadCartridgeStateStreamResponse", &LoadCartridgeStateStreamResponse{}, loadCartridgeStateStreamResponseProps,
		messages.Constructor{Params: "int stateId", Func: NewLoadCartridgeStateStreamResponse},
	)
}

// NewLoadCartridgeStateStreamResponse builds the response. StateID reads
// back as the low byte of stateID, signed.
func NewLoadCartridgeStateStreamResponse(stateID int32) *LoadCartridgeStateStreamResponse {
	m := &LoadCartridgeStateStreamResponse{}
	m.decodeValid([]byte{byte(stateID)})
	return m
}

// Parse implements messages.Message.
func (m *LoadCartridgeStateStreamResponse) Parse(raw []byte) error {
	return m.decode(loadCartridgeStateStreamResponseProps, "LoadCartridgeStateStreamResponse", raw)
}
