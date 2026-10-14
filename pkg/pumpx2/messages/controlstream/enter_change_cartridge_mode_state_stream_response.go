package controlstream

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// EnterChangeCartridgeModeStateStreamResponse is streamed while the pump
// enters change cartridge mode, under the transaction ID of the
// EnterChangeCartridgeModeRequest. State is nil for a StateID pumpX2 does not
// know.
type EnterChangeCartridgeModeStateStreamResponse struct {
	messages.Base
	StateID int32                 `pumpx2:"stateId"`
	State   *ChangeCartridgeState `pumpx2:"state"`
}

var enterChangeCartridgeModeStateStreamResponseProps = messages.Props{OpCode: -31, Size: 1, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentEnterChangeCartridgeModeStateStreamRequest"}

func init() {
	messages.Register("EnterChangeCartridgeModeStateStreamResponse", &EnterChangeCartridgeModeStateStreamResponse{}, enterChangeCartridgeModeStateStreamResponseProps,
		messages.Constructor{Params: "int stateId", Func: NewEnterChangeCartridgeModeStateStreamResponse},
	)
}

// NewEnterChangeCartridgeModeStateStreamResponse builds the response.
// StateID reads back as the low byte of stateID, signed.
func NewEnterChangeCartridgeModeStateStreamResponse(stateID int32) *EnterChangeCartridgeModeStateStreamResponse {
	m := &EnterChangeCartridgeModeStateStreamResponse{}
	m.decode([]byte{byte(stateID)})
	return m
}

// Parse implements messages.Message.
func (m *EnterChangeCartridgeModeStateStreamResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(enterChangeCartridgeModeStateStreamResponseProps, raw)
	if len(raw) != enterChangeCartridgeModeStateStreamResponseProps.Size {
		return messages.InvalidLength("EnterChangeCartridgeModeStateStreamResponse", len(raw), enterChangeCartridgeModeStateStreamResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *EnterChangeCartridgeModeStateStreamResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.StateID = int32(int8(raw[0]))
	m.State = ChangeCartridgeStateFromID(m.StateID)
}

// ChangeCartridgeState is how far the pump is into change cartridge mode.
type ChangeCartridgeState int32

// The change cartridge states pumpX2 knows, by their wire IDs.
const (
	ChangeCartridgeStateReadyToChange ChangeCartridgeState = 2
)

// ChangeCartridgeStateFromID is the state with wire ID id, or nil for an ID
// that is none.
func ChangeCartridgeStateFromID(id int32) *ChangeCartridgeState {
	if s := ChangeCartridgeState(id); s == ChangeCartridgeStateReadyToChange {
		return &s
	}
	return nil
}

func (s ChangeCartridgeState) String() string {
	if s == ChangeCartridgeStateReadyToChange {
		return "READY_TO_CHANGE"
	}
	return fmt.Sprintf("ChangeCartridgeState(%d)", int32(s))
}
