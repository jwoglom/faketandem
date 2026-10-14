package controlstream

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// FillCannulaStateStreamResponse is streamed while a Mobi fills the cannula,
// under the transaction ID of the FillCannulaRequest. State is nil for a
// StateID pumpX2 does not know.
type FillCannulaStateStreamResponse struct {
	messages.Base
	StateID int32             `pumpx2:"stateId"`
	State   *FillCannulaState `pumpx2:"state"`
}

var fillCannulaStateStreamResponseProps = messages.Props{OpCode: -25, Size: 1, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentFillCannulaStateStreamRequest"}

func init() {
	messages.Register("FillCannulaStateStreamResponse", &FillCannulaStateStreamResponse{}, fillCannulaStateStreamResponseProps,
		messages.Constructor{Params: "int stateId", Func: NewFillCannulaStateStreamResponse},
	)
}

// NewFillCannulaStateStreamResponse builds the response. StateID reads back
// as the low byte of stateID, signed.
func NewFillCannulaStateStreamResponse(stateID int32) *FillCannulaStateStreamResponse {
	m := &FillCannulaStateStreamResponse{}
	m.decode([]byte{byte(stateID)})
	return m
}

// Parse implements messages.Message.
func (m *FillCannulaStateStreamResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(fillCannulaStateStreamResponseProps, raw)
	if len(raw) != fillCannulaStateStreamResponseProps.Size {
		return messages.InvalidLength("FillCannulaStateStreamResponse", len(raw), fillCannulaStateStreamResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *FillCannulaStateStreamResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.StateID = int32(int8(raw[0]))
	m.State = FillCannulaStateFromID(m.StateID)
}

// FillCannulaState is the progress of a cannula fill.
type FillCannulaState int32

// The fill cannula states pumpX2 knows, by their wire IDs.
const (
	FillCannulaStateCannulaFilled FillCannulaState = 2
)

// FillCannulaStateFromID is the state with wire ID id, or nil for an ID that
// is none.
func FillCannulaStateFromID(id int32) *FillCannulaState {
	if s := FillCannulaState(id); s == FillCannulaStateCannulaFilled {
		return &s
	}
	return nil
}

func (s FillCannulaState) String() string {
	if s == FillCannulaStateCannulaFilled {
		return "CANNULA_FILLED"
	}
	return fmt.Sprintf("FillCannulaState(%d)", int32(s))
}
