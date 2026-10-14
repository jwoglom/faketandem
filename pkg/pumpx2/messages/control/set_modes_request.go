package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// SetModesRequest turns Control-IQ's sleep or exercise mode on or off.
// Control-IQ must be on. Command is nil for a Bitmap that is no ModeCommand.
type SetModesRequest struct {
	messages.Base
	Bitmap  int32        `pumpx2:"bitmap"`
	Command *ModeCommand `pumpx2:"command"`
}

var setModesRequestProps = messages.Props{OpCode: -52, Size: 1, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "SetModesResponse"}

func init() {
	messages.Register("SetModesRequest", &SetModesRequest{}, setModesRequestProps,
		messages.Constructor{Params: "byte[] raw", Func: NewSetModesRequestFromCargo},
		messages.Constructor{Params: "SetModesRequest.ModeCommand mode", Func: NewSetModesRequest},
		messages.Constructor{Params: "int bitmap", Func: NewSetModesRequestFromBitmap},
	)
}

// NewSetModesRequestFromCargo parses a request's payload.
func NewSetModesRequestFromCargo(raw []byte) (*SetModesRequest, error) {
	m := &SetModesRequest{}
	return m, m.Parse(raw)
}

// NewSetModesRequest is the request for a mode change.
func NewSetModesRequest(mode ModeCommand) *SetModesRequest {
	return NewSetModesRequestFromBitmap(mode.Bitmap())
}

// NewSetModesRequestFromBitmap is the request for a raw command byte.
func NewSetModesRequestFromBitmap(bitmap int32) *SetModesRequest {
	m := &SetModesRequest{}
	m.decode([]byte{byte(bitmap)})
	return m
}

// Parse implements messages.Message.
func (m *SetModesRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setModesRequestProps, raw)
	if len(raw) != setModesRequestProps.Size {
		return messages.InvalidLength("SetModesRequest", len(raw), setModesRequestProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *SetModesRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.Bitmap = int32(int8(raw[0]))
	m.Command = ModeCommandFromBitmap(m.Bitmap)
}

// ModeCommand is a Control-IQ mode change, by its command byte.
type ModeCommand int32

// The mode commands. The last three appear in Tandem Source's event schema
// but current firmware does not support them.
const (
	ModeCommandSleepModeOn                                  ModeCommand = 1
	ModeCommandSleepModeOff                                 ModeCommand = 2
	ModeCommandExerciseModeOn                               ModeCommand = 3
	ModeCommandExerciseModeOff                              ModeCommand = 4
	ModeCommandNotSupportedInCurrentFirmwareStopAll         ModeCommand = 5
	ModeCommandNotSupportedInCurrentFirmwareStartEatingSoon ModeCommand = 6
	ModeCommandNotSupportedInCurrentFirmwareStopEatingSoon  ModeCommand = 7
)

var modeCommandNames = map[ModeCommand]string{
	ModeCommandSleepModeOn:                                  "SLEEP_MODE_ON",
	ModeCommandSleepModeOff:                                 "SLEEP_MODE_OFF",
	ModeCommandExerciseModeOn:                               "EXERCISE_MODE_ON",
	ModeCommandExerciseModeOff:                              "EXERCISE_MODE_OFF",
	ModeCommandNotSupportedInCurrentFirmwareStopAll:         "NOT_SUPPORTED_IN_CURRENT_FIRMWARE__STOP_ALL",
	ModeCommandNotSupportedInCurrentFirmwareStartEatingSoon: "NOT_SUPPORTED_IN_CURRENT_FIRMWARE__START_EATING_SOON",
	ModeCommandNotSupportedInCurrentFirmwareStopEatingSoon:  "NOT_SUPPORTED_IN_CURRENT_FIRMWARE__STOP_EATING_SOON",
}

// ModeCommandFromBitmap is the command with that command byte, or nil for
// one that is none.
func ModeCommandFromBitmap(bitmap int32) *ModeCommand {
	c := ModeCommand(bitmap)
	if _, ok := modeCommandNames[c]; !ok {
		return nil
	}
	return &c
}

// Bitmap is the command byte.
func (c ModeCommand) Bitmap() int32 { return int32(c) }

func (c ModeCommand) String() string {
	if name, ok := modeCommandNames[c]; ok {
		return name
	}
	return fmt.Sprintf("ModeCommand(%d)", int32(c))
}
