package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMRateAlertSettingsResponse is the CGM rise and fall rate alerts: each
// one's threshold in mg/dL per minute, whether it is on (1) or off (0), and
// an undocumented default bitmask.
type CGMRateAlertSettingsResponse struct {
	messages.Base
	RiseRateThreshold      int32 `pumpx2:"riseRateThreshold"`
	RiseRateEnabled        int32 `pumpx2:"riseRateEnabled"`
	RiseRateDefaultBitmask int32 `pumpx2:"riseRateDefaultBitmask"`
	FallRateThreshold      int32 `pumpx2:"fallRateThreshold"`
	FallRateEnabled        int32 `pumpx2:"fallRateEnabled"`
	FallRateDefaultBitmask int32 `pumpx2:"fallRateDefaultBitmask"`
}

var cgmRateAlertSettingsResponseProps = messages.Props{OpCode: 93, Size: 6, Type: messages.Response, Request: "CGMRateAlertSettingsRequest"}

func init() {
	messages.Register("CGMRateAlertSettingsResponse", &CGMRateAlertSettingsResponse{}, cgmRateAlertSettingsResponseProps,
		messages.Constructor{
			Params: "int riseRateThreshold, int riseRateEnabled, int riseRateDefaultBitmask, int fallRateThreshold, int fallRateEnabled, int fallRateDefaultBitmask",
			Func:   NewCGMRateAlertSettingsResponse,
		},
	)
}

// NewCGMRateAlertSettingsResponse builds the response.
func NewCGMRateAlertSettingsResponse(riseRateThreshold, riseRateEnabled, riseRateDefaultBitmask,
	fallRateThreshold, fallRateEnabled, fallRateDefaultBitmask int32) *CGMRateAlertSettingsResponse {
	m := &CGMRateAlertSettingsResponse{
		RiseRateThreshold:      riseRateThreshold,
		RiseRateEnabled:        riseRateEnabled,
		RiseRateDefaultBitmask: riseRateDefaultBitmask,
		FallRateThreshold:      fallRateThreshold,
		FallRateEnabled:        fallRateEnabled,
		FallRateDefaultBitmask: fallRateDefaultBitmask,
	}
	m.SetCargo([]byte{byte(riseRateThreshold), byte(riseRateEnabled), byte(riseRateDefaultBitmask),
		byte(fallRateThreshold), byte(fallRateEnabled), byte(fallRateDefaultBitmask)})
	return m
}

// Parse implements messages.Message.
func (m *CGMRateAlertSettingsResponse) Parse(raw []byte) error {
	if len(raw) != cgmRateAlertSettingsResponseProps.Size {
		return messages.InvalidLength("CGMRateAlertSettingsResponse", len(raw), cgmRateAlertSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.RiseRateThreshold = int32(int8(raw[0]))
	m.RiseRateEnabled = int32(int8(raw[1]))
	m.RiseRateDefaultBitmask = int32(int8(raw[2]))
	m.FallRateThreshold = int32(int8(raw[3]))
	m.FallRateEnabled = int32(int8(raw[4]))
	m.FallRateDefaultBitmask = int32(int8(raw[5]))
	return nil
}
