package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMOORAlertSettingsResponse is the CGM out-of-range alert, which sounds
// when the pump has had no CGM signal for SensorTimeoutAlertThreshold
// minutes: the threshold, whether it is on (1) or off (0), and an
// undocumented default bitmask.
type CGMOORAlertSettingsResponse struct {
	messages.Base
	SensorTimeoutAlertThreshold int32 `pumpx2:"sensorTimeoutAlertThreshold"`
	SensorTimeoutAlertEnabled   int32 `pumpx2:"sensorTimeoutAlertEnabled"`
	SensorTimeoutDefaultBitmask int32 `pumpx2:"sensorTimeoutDefaultBitmask"`
}

var cgmoorAlertSettingsResponseProps = messages.Props{OpCode: 95, Size: 3, Type: messages.Response, Request: "CGMOORAlertSettingsRequest"}

func init() {
	messages.Register("CGMOORAlertSettingsResponse", &CGMOORAlertSettingsResponse{}, cgmoorAlertSettingsResponseProps,
		messages.Constructor{Params: "int sensorTimeoutAlertThreshold, int sensorTimeoutAlertEnabled, int sensorTimeoutDefaultBitmask", Func: NewCGMOORAlertSettingsResponse},
	)
}

// NewCGMOORAlertSettingsResponse builds the response.
func NewCGMOORAlertSettingsResponse(sensorTimeoutAlertThreshold, sensorTimeoutAlertEnabled, sensorTimeoutDefaultBitmask int32) *CGMOORAlertSettingsResponse {
	m := &CGMOORAlertSettingsResponse{
		SensorTimeoutAlertThreshold: sensorTimeoutAlertThreshold,
		SensorTimeoutAlertEnabled:   sensorTimeoutAlertEnabled,
		SensorTimeoutDefaultBitmask: sensorTimeoutDefaultBitmask,
	}
	m.SetCargo([]byte{byte(sensorTimeoutAlertThreshold), byte(sensorTimeoutAlertEnabled), byte(sensorTimeoutDefaultBitmask)})
	return m
}

// Parse implements messages.Message.
func (m *CGMOORAlertSettingsResponse) Parse(raw []byte) error {
	if len(raw) != cgmoorAlertSettingsResponseProps.Size {
		return messages.InvalidLength("CGMOORAlertSettingsResponse", len(raw), cgmoorAlertSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.SensorTimeoutAlertThreshold = int32(int8(raw[0]))
	m.SensorTimeoutAlertEnabled = int32(int8(raw[1]))
	m.SensorTimeoutDefaultBitmask = int32(int8(raw[2]))
	return nil
}
