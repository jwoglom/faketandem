package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ControlIQInfoV2Response is ControlIQInfoV1Response for a pump past API 2.1,
// with the exercise activity chosen, its duration and the time it has left.
type ControlIQInfoV2Response struct {
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
	ExerciseChoice        int32 `pumpx2:"exerciseChoice"`
	ExerciseDuration      int64 `pumpx2:"exerciseDuration"`
	ExerciseTimeRemaining int64 `pumpx2:"exerciseTimeRemaining"`
}

var controlIQInfoV2ResponseProps = messages.Props{OpCode: -77, Size: 19, Type: messages.Response, Request: "ControlIQInfoV2Request"}

func init() {
	messages.Register("ControlIQInfoV2Response", &ControlIQInfoV2Response{}, controlIQInfoV2ResponseProps,
		messages.Constructor{
			Params: "boolean closedLoopEnabled, int weight, int weightUnit, int totalDailyInsulin, int currentUserModeType, " +
				"int byte6, int byte7, int byte8, int controlStateType, int exerciseChoice, long exerciseDuration, long exerciseTimeRemaining",
			Func: NewControlIQInfoV2Response,
		},
	)
}

// NewControlIQInfoV2Response builds the response.
func NewControlIQInfoV2Response(closedLoopEnabled bool, weight, weightUnit, totalDailyInsulin, currentUserModeType,
	byte6, byte7, byte8, controlStateType, exerciseChoice int32, exerciseDuration, exerciseTimeRemaining int64) *ControlIQInfoV2Response {
	m := &ControlIQInfoV2Response{
		ClosedLoopEnabled:     closedLoopEnabled,
		Weight:                weight,
		WeightUnitID:          weightUnit,
		TotalDailyInsulin:     totalDailyInsulin,
		CurrentUserModeTypeID: currentUserModeType,
		Byte6:                 byte6,
		Byte7:                 byte7,
		Byte8:                 byte8,
		ControlStateType:      controlStateType,
		ExerciseChoice:        exerciseChoice,
		ExerciseDuration:      exerciseDuration,
		ExerciseTimeRemaining: exerciseTimeRemaining,
	}
	m.SetCargo(helpers.Combine(
		[]byte{helpers.Bool(closedLoopEnabled)},
		helpers.FirstTwoBytesLittleEndian(weight),
		[]byte{byte(weightUnit), byte(totalDailyInsulin), byte(currentUserModeType), byte(byte6), byte(byte7), byte(byte8),
			byte(controlStateType), byte(exerciseChoice)},
		helpers.ToUint32(exerciseDuration),
		helpers.ToUint32(exerciseTimeRemaining),
	))
	return m
}

// Parse implements messages.Message. As pumpX2 does, it reads each byte
// signed, so a total daily insulin over 127 units reads negative.
func (m *ControlIQInfoV2Response) Parse(raw []byte) error {
	if len(raw) != controlIQInfoV2ResponseProps.Size {
		return messages.InvalidLength("ControlIQInfoV2Response", len(raw), controlIQInfoV2ResponseProps.Size)
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
	m.ExerciseChoice = int32(int8(raw[10]))
	m.ExerciseDuration = helpers.ReadUint32(raw, 11)
	m.ExerciseTimeRemaining = helpers.ReadUint32(raw, 15)
	return nil
}

// CurrentUserModeType is the activity mode, or nil for an ID that is none.
func (m *ControlIQInfoV2Response) CurrentUserModeType() *UserModeType {
	return UserModeTypeFromID(m.CurrentUserModeTypeID)
}

// WeightUnit is the unit of Weight, or nil for an ID that is none.
func (m *ControlIQInfoV2Response) WeightUnit() *WeightUnit { return WeightUnitFromID(m.WeightUnitID) }
