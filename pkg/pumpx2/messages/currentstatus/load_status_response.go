package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// LoadStatusResponse is where the pump is in changing the cartridge and
// filling the tubing and cannula. PrimeTubingStatus is set only in
// PRIME_TUBING, and PrimeNudgeStatus only in PRIME_NUDGE; both read
// PrimeStatusID.
type LoadStatusResponse struct {
	messages.Base
	IsLoadingActiveID int32              `pumpx2:"isLoadingActiveId"`
	IsLoadingActive   bool               `pumpx2:"isLoadingActive"`
	LoadStateID       int32              `pumpx2:"loadStateId"`
	LoadState         LoadState          `pumpx2:"loadState"`
	PrimeStatusID     int32              `pumpx2:"primeStatusId"`
	PrimeTubingStatus *PrimeTubingStatus `pumpx2:"primeTubingStatus"`
	PrimeNudgeStatus  *PrimeNudgeStatus  `pumpx2:"primeNudgeStatus"`
}

var loadStatusResponseProps = messages.Props{OpCode: 21, Size: 3, Type: messages.Response, Request: "LoadStatusRequest"}

func init() {
	messages.Register("LoadStatusResponse", &LoadStatusResponse{}, loadStatusResponseProps,
		messages.Constructor{Params: "int status, int unknown", Func: NewLegacyLoadStatusResponse},
		messages.Constructor{Params: "boolean isLoadingActive, LoadStatusResponse.LoadState loadState, int primeStatusId", Func: NewLoadStatusResponseWithState},
		messages.Constructor{Params: "boolean isLoadingActive, int loadStateId, int primeStatusId", Func: NewLoadStatusResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewLoadStatusResponseFromCargo},
	)
}

// NewLoadStatusResponse builds the response from the load state's and prime
// status's wire IDs, each one byte, and reads it back as Parse does.
func NewLoadStatusResponse(isLoadingActive bool, loadStateID, primeStatusID int32) *LoadStatusResponse {
	m := &LoadStatusResponse{}
	m.decode(loadStatusCargo(isLoadingActive, loadStateID, primeStatusID))
	return m
}

// NewLoadStatusResponseWithState builds the response.
func NewLoadStatusResponseWithState(isLoadingActive bool, loadState LoadState, primeStatusID int32) *LoadStatusResponse {
	return NewLoadStatusResponse(isLoadingActive, loadState.ID(), primeStatusID)
}

// NewLegacyLoadStatusResponse is pumpX2's deprecated constructor, from when
// it read byte 0 as a status and bytes 1-2 as an unknown little-endian
// short.
func NewLegacyLoadStatusResponse(status, unknown int32) *LoadStatusResponse {
	m := &LoadStatusResponse{}
	m.decode([]byte{byte(status), byte(unknown), byte(unknown >> 8)})
	return m
}

// NewLoadStatusResponseFromCargo parses a response's payload.
func NewLoadStatusResponseFromCargo(raw []byte) (*LoadStatusResponse, error) {
	m := &LoadStatusResponse{}
	return m, m.Parse(raw)
}

// LoadStatusResponseCargo is the payload with the prime status that goes
// with loadState, or 0 when there is none: pumpX2's
// buildCargo(boolean, LoadState, PrimeTubingStatus, PrimeNudgeStatus).
func LoadStatusResponseCargo(isLoadingActive bool, loadState LoadState, primeTubingStatus *PrimeTubingStatus, primeNudgeStatus *PrimeNudgeStatus) []byte {
	var primeStatusID int32
	switch {
	case loadState == LoadStatePrimeTubing && primeTubingStatus != nil:
		primeStatusID = primeTubingStatus.ID()
	case loadState == LoadStatePrimeNudge && primeNudgeStatus != nil:
		primeStatusID = primeNudgeStatus.ID()
	}
	return loadStatusCargo(isLoadingActive, loadState.ID(), primeStatusID)
}

func loadStatusCargo(isLoadingActive bool, loadStateID, primeStatusID int32) []byte {
	var active byte
	if isLoadingActive {
		active = 1
	}
	return []byte{active, byte(loadStateID), byte(primeStatusID)}
}

// Parse implements messages.Message.
func (m *LoadStatusResponse) Parse(raw []byte) error {
	if len(raw) != loadStatusResponseProps.Size {
		return messages.InvalidLength("LoadStatusResponse", len(raw), loadStatusResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *LoadStatusResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.IsLoadingActiveID = int32(raw[0])
	m.IsLoadingActive = m.IsLoadingActiveID != 0
	m.LoadStateID = int32(raw[1])
	m.LoadState = LoadStateFromID(m.LoadStateID)
	m.PrimeStatusID = int32(raw[2])
	m.PrimeTubingStatus, m.PrimeNudgeStatus = nil, nil
	switch m.LoadState {
	case LoadStatePrimeTubing:
		s := PrimeTubingStatusFromID(m.PrimeStatusID)
		m.PrimeTubingStatus = &s
	case LoadStatePrimeNudge:
		s := PrimeNudgeStatusFromID(m.PrimeStatusID)
		m.PrimeNudgeStatus = &s
	}
}

// IsInLoadingState reports whether the pump is changing or loading the
// cartridge or filling the tubing, as the Tandem app tells.
func (m *LoadStatusResponse) IsInLoadingState() bool {
	switch m.LoadState {
	case LoadStateChangeCartridge, LoadStateLoadCartridge, LoadStatePrimeTubing:
		return true
	}
	return false
}

// LoadState is the step of a cartridge change the pump is at.
type LoadState int32

// The load states.
const (
	LoadStateChangeCartridge LoadState = iota
	LoadStateLoadCartridge
	LoadStatePrimeTubing
	LoadStatePrimeCannula
	LoadStatePrimeNudge
	LoadStateInvalid
	LoadStateUnknown
)

var loadStateNames = []string{"CHANGE_CARTRIDGE", "LOAD_CARTRIDGE", "PRIME_TUBING", "PRIME_CANNULA", "PRIME_NUDGE", "INVALID", "UNKNOWN"}

// LoadStateFromID is the state with wire ID id, or Unknown.
func LoadStateFromID(id int32) LoadState {
	if id >= 0 && int(id) < len(loadStateNames) {
		return LoadState(id)
	}
	return LoadStateUnknown
}

// ID is the state's wire ID.
func (s LoadState) ID() int32 { return int32(s) }

func (s LoadState) String() string {
	if s >= 0 && int(s) < len(loadStateNames) {
		return loadStateNames[s]
	}
	return fmt.Sprintf("LoadState(%d)", int32(s))
}

// PrimeTubingStatus is how a tubing fill stands.
type PrimeTubingStatus int32

// The tubing fill statuses.
const (
	PrimeTubingStatusStop PrimeTubingStatus = iota
	PrimeTubingStatusStart
	PrimeTubingStatusEnteredCanExit
	PrimeTubingStatusEnteredCannotExit
	PrimeTubingStatusSuspended
	PrimeTubingStatusUnknown
)

var primeTubingStatusNames = []string{"STOP", "START", "ENTERED_CAN_EXIT", "ENTERED_CANNOT_EXIT", "SUSPENDED", "UNKNOWN"}

// PrimeTubingStatusFromID is the status with wire ID id, or Unknown.
func PrimeTubingStatusFromID(id int32) PrimeTubingStatus {
	if id >= 0 && int(id) < len(primeTubingStatusNames) {
		return PrimeTubingStatus(id)
	}
	return PrimeTubingStatusUnknown
}

// ID is the status's wire ID.
func (s PrimeTubingStatus) ID() int32 { return int32(s) }

func (s PrimeTubingStatus) String() string {
	if s >= 0 && int(s) < len(primeTubingStatusNames) {
		return primeTubingStatusNames[s]
	}
	return fmt.Sprintf("PrimeTubingStatus(%d)", int32(s))
}

// PrimeNudgeStatus is how the PRIME_NUDGE step stands.
type PrimeNudgeStatus int32

// The PRIME_NUDGE statuses.
const (
	PrimeNudgeStatusStart PrimeNudgeStatus = iota
	PrimeNudgeStatusComplete
	PrimeNudgeStatusUnknown
)

var primeNudgeStatusNames = []string{"START", "COMPLETE", "UNKNOWN"}

// PrimeNudgeStatusFromID is the status with wire ID id, or Unknown.
func PrimeNudgeStatusFromID(id int32) PrimeNudgeStatus {
	if id >= 0 && int(id) < len(primeNudgeStatusNames) {
		return PrimeNudgeStatus(id)
	}
	return PrimeNudgeStatusUnknown
}

// ID is the status's wire ID.
func (s PrimeNudgeStatus) ID() int32 { return int32(s) }

func (s PrimeNudgeStatus) String() string {
	if s >= 0 && int(s) < len(primeNudgeStatusNames) {
		return primeNudgeStatusNames[s]
	}
	return fmt.Sprintf("PrimeNudgeStatus(%d)", int32(s))
}
