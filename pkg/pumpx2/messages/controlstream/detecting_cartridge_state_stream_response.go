package controlstream

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// DetectingCartridgeStateStreamResponse is streamed while the pump checks a
// newly inserted cartridge, under the transaction ID of the
// ExitChangeCartridgeModeRequest: PercentComplete climbs to 100. It is the
// class pumpX2 parses this opcode as, so a one-byte payload, a
// LoadCartridgeStateStreamResponse's, parses here too: LoadCartridgeState is
// set and StateID is the state, signed. Either way PercentComplete and
// StateID read the same.
type DetectingCartridgeStateStreamResponse struct {
	messages.Base
	PercentComplete    int32 `pumpx2:"percentComplete"`
	StateID            int32 `pumpx2:"stateId"`
	LoadCartridgeState bool  `pumpx2:"loadCartridgeState"`
}

var detectingCartridgeStateStreamResponseProps = messages.Props{OpCode: -29, Size: 2, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentDetectingCartridgeStateStreamRequest"}

func init() {
	messages.Register("DetectingCartridgeStateStreamResponse", &DetectingCartridgeStateStreamResponse{}, detectingCartridgeStateStreamResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewDetectingCartridgeStateStreamResponseFromCargo},
		messages.Constructor{Params: "int percentComplete", Func: NewDetectingCartridgeStateStreamResponse},
	)
}

// NewDetectingCartridgeStateStreamResponseFromCargo parses a response's
// payload.
func NewDetectingCartridgeStateStreamResponseFromCargo(raw []byte) (*DetectingCartridgeStateStreamResponse, error) {
	m := &DetectingCartridgeStateStreamResponse{}
	return m, m.Parse(raw)
}

// NewDetectingCartridgeStateStreamResponse builds the response. As pumpX2's
// does, it leaves StateID 0, where a parse of the same payload reads it as
// percentComplete.
func NewDetectingCartridgeStateStreamResponse(percentComplete int32) *DetectingCartridgeStateStreamResponse {
	m := &DetectingCartridgeStateStreamResponse{PercentComplete: percentComplete}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(percentComplete))
	return m
}

// Parse implements messages.Message.
func (m *DetectingCartridgeStateStreamResponse) Parse(raw []byte) error {
	return m.decode(detectingCartridgeStateStreamResponseProps, "DetectingCartridgeStateStreamResponse", raw)
}

// decode is the parse LoadCartridgeStateStreamResponse inherits, which drops
// a signed trailer by the size in props.
func (m *DetectingCartridgeStateStreamResponse) decode(props messages.Props, name string, raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(props, raw)
	if len(raw) != 1 && len(raw) != 2 {
		return messages.Invalid("%s: payload of %d bytes, want 1 or 2", name, len(raw))
	}
	m.decodeValid(raw)
	return nil
}

func (m *DetectingCartridgeStateStreamResponse) decodeValid(raw []byte) {
	m.SetCargo(raw)
	if len(raw) == 2 {
		m.PercentComplete = helpers.ReadShort(raw, 0)
		m.StateID = m.PercentComplete
		m.LoadCartridgeState = false
	} else {
		m.StateID = int32(int8(raw[0]))
		m.PercentComplete = m.StateID
		m.LoadCartridgeState = true
	}
}

// IsComplete reports whether the check has finished.
func (m *DetectingCartridgeStateStreamResponse) IsComplete() bool { return m.PercentComplete == 100 }
