package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CGMGlucoseAlertSettingsResponse is the CGM high and low glucose alerts:
// each one's threshold in mg/dL, whether it is on (1) or off (0), how long
// before it repeats in minutes (0 for never), and an undocumented default
// bitmask.
type CGMGlucoseAlertSettingsResponse struct {
	messages.Base
	HighGlucoseAlertThreshold      int32 `pumpx2:"highGlucoseAlertThreshold"`
	HighGlucoseAlertEnabled        int32 `pumpx2:"highGlucoseAlertEnabled"`
	HighGlucoseRepeatDuration      int32 `pumpx2:"highGlucoseRepeatDuration"`
	HighGlucoseAlertDefaultBitmask int32 `pumpx2:"highGlucoseAlertDefaultBitmask"`
	LowGlucoseAlertThreshold       int32 `pumpx2:"lowGlucoseAlertThreshold"`
	LowGlucoseAlertEnabled         int32 `pumpx2:"lowGlucoseAlertEnabled"`
	LowGlucoseRepeatDuration       int32 `pumpx2:"lowGlucoseRepeatDuration"`
	LowGlucoseAlertDefaultBitmask  int32 `pumpx2:"lowGlucoseAlertDefaultBitmask"`
}

var cgmGlucoseAlertSettingsResponseProps = messages.Props{OpCode: 91, Size: 12, Type: messages.Response, Request: "CGMGlucoseAlertSettingsRequest"}

func init() {
	messages.Register("CGMGlucoseAlertSettingsResponse", &CGMGlucoseAlertSettingsResponse{}, cgmGlucoseAlertSettingsResponseProps,
		messages.Constructor{
			Params: "int highGlucoseAlertThreshold, int highGlucoseAlertEnabled, int highGlucoseRepeatDuration, int highGlucoseAlertDefaultBitmask, " +
				"int lowGlucoseAlertThreshold, int lowGlucoseAlertEnabled, int lowGlucoseRepeatDuration, int lowGlucoseAlertDefaultBitmask",
			Func: NewCGMGlucoseAlertSettingsResponse,
		},
	)
}

// NewCGMGlucoseAlertSettingsResponse builds the response.
func NewCGMGlucoseAlertSettingsResponse(highGlucoseAlertThreshold, highGlucoseAlertEnabled, highGlucoseRepeatDuration, highGlucoseAlertDefaultBitmask,
	lowGlucoseAlertThreshold, lowGlucoseAlertEnabled, lowGlucoseRepeatDuration, lowGlucoseAlertDefaultBitmask int32) *CGMGlucoseAlertSettingsResponse {
	m := &CGMGlucoseAlertSettingsResponse{
		HighGlucoseAlertThreshold:      highGlucoseAlertThreshold,
		HighGlucoseAlertEnabled:        highGlucoseAlertEnabled,
		HighGlucoseRepeatDuration:      highGlucoseRepeatDuration,
		HighGlucoseAlertDefaultBitmask: highGlucoseAlertDefaultBitmask,
		LowGlucoseAlertThreshold:       lowGlucoseAlertThreshold,
		LowGlucoseAlertEnabled:         lowGlucoseAlertEnabled,
		LowGlucoseRepeatDuration:       lowGlucoseRepeatDuration,
		LowGlucoseAlertDefaultBitmask:  lowGlucoseAlertDefaultBitmask,
	}
	m.SetCargo(helpers.Combine(
		helpers.FirstTwoBytesLittleEndian(highGlucoseAlertThreshold),
		[]byte{byte(highGlucoseAlertEnabled)},
		helpers.FirstTwoBytesLittleEndian(highGlucoseRepeatDuration),
		[]byte{byte(highGlucoseAlertDefaultBitmask)},
		helpers.FirstTwoBytesLittleEndian(lowGlucoseAlertThreshold),
		[]byte{byte(lowGlucoseAlertEnabled)},
		helpers.FirstTwoBytesLittleEndian(lowGlucoseRepeatDuration),
		[]byte{byte(lowGlucoseAlertDefaultBitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *CGMGlucoseAlertSettingsResponse) Parse(raw []byte) error {
	if len(raw) != cgmGlucoseAlertSettingsResponseProps.Size {
		return messages.InvalidLength("CGMGlucoseAlertSettingsResponse", len(raw), cgmGlucoseAlertSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.HighGlucoseAlertThreshold = helpers.ReadShort(raw, 0)
	m.HighGlucoseAlertEnabled = int32(int8(raw[2]))
	m.HighGlucoseRepeatDuration = helpers.ReadShort(raw, 3)
	m.HighGlucoseAlertDefaultBitmask = int32(int8(raw[5]))
	m.LowGlucoseAlertThreshold = helpers.ReadShort(raw, 6)
	m.LowGlucoseAlertEnabled = int32(int8(raw[8]))
	m.LowGlucoseRepeatDuration = helpers.ReadShort(raw, 9)
	m.LowGlucoseAlertDefaultBitmask = int32(int8(raw[11]))
	return nil
}
