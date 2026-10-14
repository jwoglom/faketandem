package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BasalIQStatusResponse is what Basal-IQ is doing, as a BasalIQStatusState
// ID, and whether the pump is delivering insulin.
type BasalIQStatusResponse struct {
	messages.Base
	BasalIQStatusState int32 `pumpx2:"basalIQStatusState"`
	DeliveringTherapy  bool  `pumpx2:"deliveringTherapy"`
}

var basalIQStatusResponseProps = messages.Props{OpCode: 113, Size: 2, Type: messages.Response, Request: "BasalIQStatusRequest"}

func init() {
	messages.Register("BasalIQStatusResponse", &BasalIQStatusResponse{}, basalIQStatusResponseProps,
		messages.Constructor{Params: "int basalIQStatusState, boolean deliveringTherapy", Func: NewBasalIQStatusResponse},
	)
}

// NewBasalIQStatusResponse builds the response.
func NewBasalIQStatusResponse(basalIQStatusState int32, deliveringTherapy bool) *BasalIQStatusResponse {
	m := &BasalIQStatusResponse{BasalIQStatusState: basalIQStatusState, DeliveringTherapy: deliveringTherapy}
	m.SetCargo([]byte{byte(basalIQStatusState), helpers.Bool(deliveringTherapy)})
	return m
}

// Parse implements messages.Message.
func (m *BasalIQStatusResponse) Parse(raw []byte) error {
	if len(raw) != basalIQStatusResponseProps.Size {
		return messages.InvalidLength("BasalIQStatusResponse", len(raw), basalIQStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BasalIQStatusState = int32(int8(raw[0]))
	m.DeliveringTherapy = raw[1] != 0
	return nil
}

// State is the state BasalIQStatusState names, or nil.
func (m *BasalIQStatusResponse) State() *BasalIQStatusState {
	return BasalIQStatusStateFromID(m.BasalIQStatusState)
}

// BasalIQStatusState is what Basal-IQ is doing.
type BasalIQStatusState int32

// The Basal-IQ states, by their wire IDs.
const (
	BasalIQStatusStateIdle        BasalIQStatusState = 0
	BasalIQStatusStateSuspend     BasalIQStatusState = 1
	BasalIQStatusStateDisabled    BasalIQStatusState = 2
	BasalIQStatusStateUnavailable BasalIQStatusState = 3
)

var basalIQStatusStateNames = []string{"Idle", "Suspend", "Disabled", "Unavailable"}

// BasalIQStatusStateFromID is the state with wire ID id, or nil.
func BasalIQStatusStateFromID(id int32) *BasalIQStatusState {
	if id < 0 || int(id) >= len(basalIQStatusStateNames) {
		return nil
	}
	s := BasalIQStatusState(id)
	return &s
}

func (s BasalIQStatusState) String() string {
	if s >= 0 && int(s) < len(basalIQStatusStateNames) {
		return basalIQStatusStateNames[s]
	}
	return fmt.Sprintf("BasalIQStatusState(%d)", int32(s))
}
