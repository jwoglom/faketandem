package controlstream

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ExitFillTubingModeStateStreamResponse is streamed while the pump leaves
// fill tubing mode, under the transaction ID of the
// ExitFillTubingModeRequest: State is nil for a StateID pumpX2 does not know.
//
// It is the class pumpX2 parses this opcode as, so it also parses the
// five-byte payload of a PumpingStateStreamResponse: whether the pumping
// state has been set since start-up, then the PumpingState bits. That first
// byte is StateID as well.
type ExitFillTubingModeStateStreamResponse struct {
	messages.Base
	StateID                       int32                    `pumpx2:"stateId"`
	State                         *ExitFillTubingModeState `pumpx2:"state"`
	IsPumpingStateSetAfterStartUp bool                     `pumpx2:"isPumpingStateSetAfterStartUp"`
	StateBitmask                  int64                    `pumpx2:"stateBitmask"`
}

var exitFillTubingModeStateStreamResponseProps = messages.Props{OpCode: -23, Size: 1, Stream: true, Signed: true, Type: messages.Response,
	Characteristic: messages.ControlStream, Request: "NonexistentExitFillTubingModeStateStreamRequest"}

func init() {
	messages.Register("ExitFillTubingModeStateStreamResponse", &ExitFillTubingModeStateStreamResponse{}, exitFillTubingModeStateStreamResponseProps,
		messages.Constructor{Params: "boolean isPumpingStateSetAfterStartUp, long stateBitmask", Func: NewExitFillTubingModeStateStreamResponseWithPumpingState},
		messages.Constructor{Params: "int stateId", Func: NewExitFillTubingModeStateStreamResponse},
	)
}

// NewExitFillTubingModeStateStreamResponse builds the response. StateID
// reads back as the low byte of stateID.
func NewExitFillTubingModeStateStreamResponse(stateID int32) *ExitFillTubingModeStateStreamResponse {
	m := &ExitFillTubingModeStateStreamResponse{}
	m.decodeValid([]byte{byte(stateID)})
	return m
}

// NewExitFillTubingModeStateStreamResponseWithPumpingState builds the
// five-byte form, a PumpingStateStreamResponse's payload. StateBitmask reads
// back as its low four bytes.
func NewExitFillTubingModeStateStreamResponseWithPumpingState(isPumpingStateSetAfterStartUp bool, stateBitmask int64) *ExitFillTubingModeStateStreamResponse {
	m := &ExitFillTubingModeStateStreamResponse{}
	m.decodeValid(pumpingStateCargo(isPumpingStateSetAfterStartUp, stateBitmask))
	return m
}

// Parse implements messages.Message.
func (m *ExitFillTubingModeStateStreamResponse) Parse(raw []byte) error {
	return m.decode(exitFillTubingModeStateStreamResponseProps, "ExitFillTubingModeStateStreamResponse", raw)
}

// decode is the parse PrimeNudgeStateStreamResponse and
// PumpingStateStreamResponse inherit, which drops a signed trailer by the
// size in props.
func (m *ExitFillTubingModeStateStreamResponse) decode(props messages.Props, name string, raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(props, raw)
	if len(raw) != 1 && len(raw) != 5 {
		return messages.Invalid("%s: payload of %d bytes, want 1 or 5", name, len(raw))
	}
	m.decodeValid(raw)
	return nil
}

func (m *ExitFillTubingModeStateStreamResponse) decodeValid(raw []byte) {
	m.SetCargo(raw)
	m.StateID = int32(raw[0])
	m.State = ExitFillTubingModeStateFromID(m.StateID)
	if len(raw) == 5 {
		m.IsPumpingStateSetAfterStartUp = raw[0] != 0
		m.StateBitmask = helpers.ReadUint32(raw, 1)
	} else {
		m.IsPumpingStateSetAfterStartUp = false
		m.StateBitmask = 0
	}
}

func pumpingStateCargo(isPumpingStateSetAfterStartUp bool, stateBitmask int64) []byte {
	return helpers.Combine([]byte{helpers.Bool(isPumpingStateSetAfterStartUp)}, helpers.ToUint32(stateBitmask))
}

// States is StateBitmask as PumpingStates.
func (m *ExitFillTubingModeStateStreamResponse) States() []PumpingState {
	return PumpingStateFromBitmask(m.StateBitmask)
}

// ExitFillTubingModeState is the progress of leaving fill tubing mode.
type ExitFillTubingModeState int32

// The exit fill tubing mode states, by their wire IDs.
const (
	ExitFillTubingModeStateNotComplete  ExitFillTubingModeState = 0
	ExitFillTubingModeStateTubingFilled ExitFillTubingModeState = 1
)

var exitFillTubingModeStateNames = []string{"NOT_COMPLETE", "TUBING_FILLED"}

// ExitFillTubingModeStateFromID is the state with wire ID id, or nil for an
// ID that is none.
func ExitFillTubingModeStateFromID(id int32) *ExitFillTubingModeState {
	if id < 0 || int(id) >= len(exitFillTubingModeStateNames) {
		return nil
	}
	s := ExitFillTubingModeState(id)
	return &s
}

func (s ExitFillTubingModeState) String() string {
	if s >= 0 && int(s) < len(exitFillTubingModeStateNames) {
		return exitFillTubingModeStateNames[s]
	}
	return fmt.Sprintf("ExitFillTubingModeState(%d)", int32(s))
}

// PumpingState is one bit of a pumping state stream's StateBitmask.
type PumpingState int64

// The pumping states, by their bits.
const (
	PumpingStateIsDeliveringTherapy   PumpingState = 1
	PumpingStateCanResumeTherapy      PumpingState = 2
	PumpingStateCanAutoResume         PumpingState = 4
	PumpingStateBolusAllowed          PumpingState = 8
	PumpingStateTubingFilled          PumpingState = 16
	PumpingStateDeliveringBasal       PumpingState = 32
	PumpingStateDeliveringBolus       PumpingState = 64
	PumpingStateDeliveringNormalBolus PumpingState = 128
	PumpingStateIsFillTubingAllowed   PumpingState = 256
	PumpingStateIsBasalState          PumpingState = 512
	PumpingStateIsPrepCartridgeState  PumpingState = 1024
	PumpingStateIsLoadCartridgeState  PumpingState = 2048
	PumpingStateIsFillState           PumpingState = 4096
	PumpingStateIsEstimateState       PumpingState = 8192
	PumpingStateCartridgeIsInstalled  PumpingState = 16384
	PumpingStateCanSnooze             PumpingState = 32768
)

// PumpingStates are the pumping states in pumpX2's declaration order, which
// is their bits' order.
var PumpingStates = []PumpingState{
	PumpingStateIsDeliveringTherapy, PumpingStateCanResumeTherapy, PumpingStateCanAutoResume, PumpingStateBolusAllowed,
	PumpingStateTubingFilled, PumpingStateDeliveringBasal, PumpingStateDeliveringBolus, PumpingStateDeliveringNormalBolus,
	PumpingStateIsFillTubingAllowed, PumpingStateIsBasalState, PumpingStateIsPrepCartridgeState, PumpingStateIsLoadCartridgeState,
	PumpingStateIsFillState, PumpingStateIsEstimateState, PumpingStateCartridgeIsInstalled, PumpingStateCanSnooze,
}

var pumpingStateNames = map[PumpingState]string{
	PumpingStateIsDeliveringTherapy:   "IS_DELIVERING_THERAPY",
	PumpingStateCanResumeTherapy:      "CAN_RESUME_THERAPY",
	PumpingStateCanAutoResume:         "CAN_AUTO_RESUME",
	PumpingStateBolusAllowed:          "BOLUS_ALLOWED",
	PumpingStateTubingFilled:          "TUBING_FILLED",
	PumpingStateDeliveringBasal:       "DELIVERING_BASAL",
	PumpingStateDeliveringBolus:       "DELIVERING_BOLUS",
	PumpingStateDeliveringNormalBolus: "DELIVERING_NORMAL_BOLUS",
	PumpingStateIsFillTubingAllowed:   "IS_FILL_TUBING_ALLOWED",
	PumpingStateIsBasalState:          "IS_BASAL_STATE",
	PumpingStateIsPrepCartridgeState:  "IS_PREP_CARTRIDGE_STATE",
	PumpingStateIsLoadCartridgeState:  "IS_LOAD_CARTRIDGE_STATE",
	PumpingStateIsFillState:           "IS_FILL_STATE",
	PumpingStateIsEstimateState:       "IS_ESTIMATE_STATE",
	PumpingStateCartridgeIsInstalled:  "CARTRIDGE_IS_INSTALLED",
	PumpingStateCanSnooze:             "CAN_SNOOZE",
}

// PumpingStateFromBitmask is the states whose bits mask sets, in
// declaration order; never nil.
func PumpingStateFromBitmask(mask int64) []PumpingState {
	out := []PumpingState{}
	for _, s := range PumpingStates {
		if mask&int64(s) != 0 {
			out = append(out, s)
		}
	}
	return out
}

// PumpingStateToBitmask ORs the states' bits.
func PumpingStateToBitmask(states ...PumpingState) int64 {
	var mask int64
	for _, s := range states {
		mask |= int64(s)
	}
	return mask
}

func (s PumpingState) String() string {
	if name, ok := pumpingStateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("PumpingState(%d)", int64(s))
}
