package currentstatus

import (
	"fmt"
	"strconv"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// NotificationMessage is pumpX2's NotificationMessage: a response that
// reports some of the pump's active notifications.
type NotificationMessage interface {
	messages.Message
	// Size is how many notifications the message reports.
	Size() int
	// NotificationIDs are the IDs of the reported notifications that pumpX2
	// describes.
	NotificationIDs() []int32
}

var (
	_ NotificationMessage = (*ActiveAamBitsResponse)(nil)
	_ NotificationMessage = (*AlarmStatusResponse)(nil)
	_ NotificationMessage = (*AlertStatusResponse)(nil)
	_ NotificationMessage = (*ReminderStatusResponse)(nil)
	_ NotificationMessage = (*CGMAlertStatusResponse)(nil)
	_ NotificationMessage = (*HighestAamResponse)(nil)
	_ NotificationMessage = (*MalfunctionBitmaskStatusResponse)(nil)
)

// HighestAamResponse is the pump's highest-priority active alert, alarm or
// malfunction (AAM). For a malfunction, ErrorString is the code to report to
// Tandem: AamID and FaultID as "12-0x2071". Remaining is the payload's last
// two or three bytes, which pumpX2 does not decode.
type HighestAamResponse struct {
	messages.Base
	// IgnorableCodes are the {AamID, FaultID} pairs that are not
	// malfunctions; pumpX2 prints its constant on every instance.
	IgnorableCodes [][]int64 `pumpx2:"IGNORABLE_CODES"`
	AamID          int64     `pumpx2:"aamId"`
	FaultID        int64     `pumpx2:"faultId"`
	Remaining      []byte    `pumpx2:"remaining"`
	ErrorString    string    `pumpx2:"errorString"`
}

var highestAamResponseProps = messages.Props{OpCode: 121, Size: 11, VariableSize: true, Type: messages.Response, Request: "HighestAamRequest"}

func init() {
	messages.Register("HighestAamResponse", &HighestAamResponse{}, highestAamResponseProps,
		messages.Constructor{Func: NewEmptyHighestAamResponse},
		messages.Constructor{Params: "long aamId, long faultId, byte[] remaining", Func: NewHighestAamResponse},
	)
}

func ignorableAamCodes() [][]int64 {
	return [][]int64{
		{0, 0},
		{3, 8230},  // 3-0x2026, alongside PUMP_RESET_ALARM
		{18, 8311}, // 18-0x2077, alongside RESUME_PUMP_ALARM on a new pump
		{26, 8322}, // 26-0x2082
	}
}

// NewEmptyHighestAamResponse is the response pumpX2 parses into.
func NewEmptyHighestAamResponse() *HighestAamResponse {
	return &HighestAamResponse{IgnorableCodes: ignorableAamCodes()}
}

// NewHighestAamResponse builds the response. It fails unless remaining is
// two or three bytes.
func NewHighestAamResponse(aamID, faultID int64, remaining []byte) (*HighestAamResponse, error) {
	m := NewEmptyHighestAamResponse()
	if err := m.Parse(helpers.Combine(helpers.ToUint32(aamID), helpers.ToUint32(faultID), remaining)); err != nil {
		return nil, err
	}
	return m, nil
}

// Parse implements messages.Message. The payload is 10 or 11 bytes.
func (m *HighestAamResponse) Parse(raw []byte) error {
	if len(raw) != highestAamResponseProps.Size && len(raw) != highestAamResponseProps.Size-1 {
		return messages.Invalid("Unexpected HighestAamResponse payload length: %d", len(raw))
	}
	if m.IgnorableCodes == nil {
		m.IgnorableCodes = ignorableAamCodes()
	}
	m.SetCargo(raw)
	m.AamID = helpers.ReadUint32(raw, 0)
	m.FaultID = helpers.ReadUint32(raw, 4)
	m.Remaining = helpers.DropFirstN(raw, 8)
	m.ErrorString = m.errorString()
	return nil
}

// FaultIDAsHex is FaultID in lower-case hex, without a prefix.
func (m *HighestAamResponse) FaultIDAsHex() string { return strconv.FormatInt(m.FaultID, 16) }

func (m *HighestAamResponse) errorString() string {
	if !m.HasMalfunction() {
		return ""
	}
	return fmt.Sprintf("%d-0x%s", m.AamID, m.FaultIDAsHex())
}

// HasMalfunction reports whether the response is a real malfunction: not
// one of IgnorableCodes, and not, by AamID, a notification one of others
// reports (an AlarmStatusResponse or AlertStatusResponse read alongside it).
func (m *HighestAamResponse) HasMalfunction(others ...NotificationMessage) bool {
	for _, code := range m.IgnorableCodes {
		if code[0] == m.AamID && code[1] == m.FaultID {
			return false
		}
	}
	return !m.MatchesKnownIDs(others...)
}

// MatchesKnownIDs reports whether one of others reports a described
// notification whose ID is AamID.
func (m *HighestAamResponse) MatchesKnownIDs(others ...NotificationMessage) bool {
	for _, other := range others {
		for _, id := range other.NotificationIDs() {
			if id == int32(m.AamID) {
				return true
			}
		}
	}
	return false
}

// MatchesActiveMalfunction reports whether AamID is an active malfunction:
// its bit in active's mask when active reports malfunctions, else in
// bitmask's. With neither to go on it is true.
func (m *HighestAamResponse) MatchesActiveMalfunction(active *ActiveAamBitsResponse, bitmask *MalfunctionBitmaskStatusResponse) bool {
	if active != nil && active.AamType == AamTypeMalfunction {
		return active.HasActiveBit(int32(m.AamID))
	}
	if bitmask != nil {
		return bitmask.HasMalfunctionBit(int32(m.AamID))
	}
	return true
}

// Size is 1 for a malfunction and 0 otherwise.
func (m *HighestAamResponse) Size() int {
	if m.HasMalfunction() {
		return 1
	}
	return 0
}

// NotificationIDs is empty, as pumpX2 leaves it.
func (m *HighestAamResponse) NotificationIDs() []int32 { return []int32{} }
