package controlstream

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// FillTubingStateStreamResponse is streamed while the tubing fills, under the
// transaction ID of the EnterFillTubingModeRequest: ButtonState is 1 while
// the pump's button is held down to fill, and 0 once it is released.
type FillTubingStateStreamResponse struct {
	messages.Base
	ButtonState int32 `pumpx2:"buttonState"`
}

var fillTubingStateStreamResponseProps = messages.Props{OpCode: -27, Size: 1, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentFillTubingStateStreamRequest"}

func init() {
	messages.Register("FillTubingStateStreamResponse", &FillTubingStateStreamResponse{}, fillTubingStateStreamResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewFillTubingStateStreamResponseFromCargo},
	)
}

// NewFillTubingStateStreamResponseFromCargo parses a response's payload,
// which is the button state's one byte.
func NewFillTubingStateStreamResponseFromCargo(raw []byte) (*FillTubingStateStreamResponse, error) {
	m := &FillTubingStateStreamResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *FillTubingStateStreamResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(fillTubingStateStreamResponseProps, raw)
	if len(raw) != fillTubingStateStreamResponseProps.Size {
		return messages.InvalidLength("FillTubingStateStreamResponse", len(raw), fillTubingStateStreamResponseProps.Size)
	}
	m.SetCargo(raw)
	m.ButtonState = int32(int8(raw[0]))
	return nil
}

// ButtonDown reports whether the pump's button is held down.
func (m *FillTubingStateStreamResponse) ButtonDown() bool { return m.ButtonState == 1 }
