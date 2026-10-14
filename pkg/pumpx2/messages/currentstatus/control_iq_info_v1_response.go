package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ControlIQInfoV1Response is Control-IQ's state and settings on a pump at API
// 2.1: whether closed loop is on, the user's weight (in WeightUnitID's unit)
// and total daily insulin (units), the activity mode (a UserModeType ID) and
// the control state. Byte6 to Byte8 are not understood.
type ControlIQInfoV1Response struct {
	messages.Base
	ClosedLoopEnabled     bool  `pumpx2:"closedLoopEnabled"`
	Weight                int32 `pumpx2:"weight"`
	WeightUnitID          int32 `pumpx2:"weightUnit"`
	TotalDailyInsulin     int32 `pumpx2:"totalDailyInsulin"`
	CurrentUserModeTypeID int32 `pumpx2:"currentUserModeType"`
	Byte6                 int32 `pumpx2:"byte6"`
	Byte7                 int32 `pumpx2:"byte7"`
	Byte8                 int32 `pumpx2:"byte8"`
	ControlStateType      int32 `pumpx2:"controlStateType"`
}

var controlIQInfoV1ResponseProps = messages.Props{OpCode: 105, Size: 10, Type: messages.Response, Request: "ControlIQInfoV1Request"}

func init() {
	messages.Register("ControlIQInfoV1Response", &ControlIQInfoV1Response{}, controlIQInfoV1ResponseProps,
		messages.Constructor{
			Params: "boolean closedLoopEnabled, int weight, int weightUnit, int totalDailyInsulin, int currentUserModeType, " +
				"int byte6, int byte7, int byte8, int controlStateType",
			Func: NewControlIQInfoV1Response,
		},
	)
}

// NewControlIQInfoV1Response builds the response.
func NewControlIQInfoV1Response(closedLoopEnabled bool, weight, weightUnit, totalDailyInsulin, currentUserModeType,
	byte6, byte7, byte8, controlStateType int32) *ControlIQInfoV1Response {
	m := &ControlIQInfoV1Response{
		ClosedLoopEnabled:     closedLoopEnabled,
		Weight:                weight,
		WeightUnitID:          weightUnit,
		TotalDailyInsulin:     totalDailyInsulin,
		CurrentUserModeTypeID: currentUserModeType,
		Byte6:                 byte6,
		Byte7:                 byte7,
		Byte8:                 byte8,
		ControlStateType:      controlStateType,
	}
	m.SetCargo(helpers.Combine(
		[]byte{helpers.Bool(closedLoopEnabled)},
		helpers.FirstTwoBytesLittleEndian(weight),
		[]byte{byte(weightUnit), byte(totalDailyInsulin), byte(currentUserModeType), byte(byte6), byte(byte7), byte(byte8), byte(controlStateType)},
	))
	return m
}

// Parse implements messages.Message. As pumpX2 does, it reads each byte
// signed, so a total daily insulin over 127 units reads negative.
func (m *ControlIQInfoV1Response) Parse(raw []byte) error {
	if len(raw) != controlIQInfoV1ResponseProps.Size {
		return messages.InvalidLength("ControlIQInfoV1Response", len(raw), controlIQInfoV1ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.ClosedLoopEnabled = raw[0] != 0
	m.Weight = helpers.ReadShort(raw, 1)
	m.WeightUnitID = int32(int8(raw[3]))
	m.TotalDailyInsulin = int32(int8(raw[4]))
	m.CurrentUserModeTypeID = int32(int8(raw[5]))
	m.Byte6 = int32(int8(raw[6]))
	m.Byte7 = int32(int8(raw[7]))
	m.Byte8 = int32(int8(raw[8]))
	m.ControlStateType = int32(int8(raw[9]))
	return nil
}

// CurrentUserModeType is the activity mode, or nil for an ID that is none.
func (m *ControlIQInfoV1Response) CurrentUserModeType() *UserModeType {
	return UserModeTypeFromID(m.CurrentUserModeTypeID)
}

// WeightUnit is the unit of Weight, or nil for an ID that is none.
func (m *ControlIQInfoV1Response) WeightUnit() *WeightUnit { return WeightUnitFromID(m.WeightUnitID) }

// UserModeType is Control-IQ's activity mode, from pumpX2's
// ControlIQInfoAbstractResponse.
type UserModeType int32

// The activity modes. Current firmware has no eating soon mode; Tandem
// Source's event schema names one.
const (
	UserModeTypeStandard UserModeType = iota
	UserModeTypeSleep
	UserModeTypeExercise
	UserModeTypeNotSupportedInCurrentFirmwareEatingSoon
)

var userModeTypeNames = []string{"STANDARD", "SLEEP", "EXERCISE", "NOT_SUPPORTED_IN_CURRENT_FIRMWARE__EATING_SOON"}

// UserModeTypeFromID is the mode with wire ID id, or nil for an ID that is
// none.
func UserModeTypeFromID(id int32) *UserModeType {
	if id < 0 || int(id) >= len(userModeTypeNames) {
		return nil
	}
	t := UserModeType(id)
	return &t
}

// ID is the mode's wire ID.
func (t UserModeType) ID() int32 { return int32(t) }

func (t UserModeType) String() string {
	if t >= 0 && int(t) < len(userModeTypeNames) {
		return userModeTypeNames[t]
	}
	return fmt.Sprintf("UserModeType(%d)", int32(t))
}

// WeightUnit is the unit Control-IQ keeps the user's weight in, from pumpX2's
// ControlIQInfoAbstractResponse.
type WeightUnit int32

// The weight units.
const (
	WeightUnitKilograms WeightUnit = iota
	WeightUnitPounds
)

var weightUnitNames = []string{"KILOGRAMS", "POUNDS"}

// WeightUnitFromID is the unit with wire ID id, or nil for an ID that is none.
func WeightUnitFromID(id int32) *WeightUnit {
	if id < 0 || int(id) >= len(weightUnitNames) {
		return nil
	}
	u := WeightUnit(id)
	return &u
}

// ID is the unit's wire ID.
func (u WeightUnit) ID() int32 { return int32(u) }

func (u WeightUnit) String() string {
	if u >= 0 && int(u) < len(weightUnitNames) {
		return weightUnitNames[u]
	}
	return fmt.Sprintf("WeightUnit(%d)", int32(u))
}
