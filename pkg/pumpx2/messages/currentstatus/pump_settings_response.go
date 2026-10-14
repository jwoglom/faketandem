package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpSettingsResponse is the pump's low insulin alert threshold, cannula
// fill amount, auto-off alarm (whether it is on and after how long), feature
// lock and screen timeout. Status is 0 on success.
type PumpSettingsResponse struct {
	messages.Base
	LowInsulinThreshold  int32 `pumpx2:"lowInsulinThreshold"`
	CannulaPrimeSize     int32 `pumpx2:"cannulaPrimeSize"`
	AutoShutdownEnabled  int32 `pumpx2:"autoShutdownEnabled"`
	AutoShutdownDuration int32 `pumpx2:"autoShutdownDuration"`
	FeatureLock          int32 `pumpx2:"featureLock"`
	OLEDTimeout          int32 `pumpx2:"oledTimeout"`
	Status               int32 `pumpx2:"status"`
}

var pumpSettingsResponseProps = messages.Props{OpCode: 83, Size: 9, Type: messages.Response, Request: "PumpSettingsRequest"}

func init() {
	messages.Register("PumpSettingsResponse", &PumpSettingsResponse{}, pumpSettingsResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewPumpSettingsResponseFromCargo},
		messages.Constructor{
			Params: "int lowInsulinThreshold, int cannulaPrimeSize, int autoShutdownEnabled, int autoShutdownDuration, int featureLock, int oledTimeout, int status",
			Func:   NewPumpSettingsResponse,
		},
	)
}

// NewPumpSettingsResponseFromCargo parses a response's payload.
func NewPumpSettingsResponseFromCargo(raw []byte) (*PumpSettingsResponse, error) {
	m := &PumpSettingsResponse{}
	return m, m.Parse(raw)
}

// NewPumpSettingsResponse builds the response.
func NewPumpSettingsResponse(lowInsulinThreshold, cannulaPrimeSize, autoShutdownEnabled, autoShutdownDuration, featureLock, oledTimeout, status int32) *PumpSettingsResponse {
	m := &PumpSettingsResponse{
		LowInsulinThreshold:  lowInsulinThreshold,
		CannulaPrimeSize:     cannulaPrimeSize,
		AutoShutdownEnabled:  autoShutdownEnabled,
		AutoShutdownDuration: autoShutdownDuration,
		FeatureLock:          featureLock,
		OLEDTimeout:          oledTimeout,
		Status:               status,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(lowInsulinThreshold), byte(cannulaPrimeSize), byte(autoShutdownEnabled)},
		helpers.FirstTwoBytesLittleEndian(autoShutdownDuration),
		[]byte{byte(featureLock), byte(oledTimeout)},
		helpers.FirstTwoBytesLittleEndian(status),
	))
	return m
}

// Parse implements messages.Message.
func (m *PumpSettingsResponse) Parse(raw []byte) error {
	if len(raw) != pumpSettingsResponseProps.Size {
		return messages.InvalidLength("PumpSettingsResponse", len(raw), pumpSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.LowInsulinThreshold = int32(int8(raw[0]))
	m.CannulaPrimeSize = int32(int8(raw[1]))
	m.AutoShutdownEnabled = int32(int8(raw[2]))
	m.AutoShutdownDuration = helpers.ReadShort(raw, 3)
	m.FeatureLock = int32(int8(raw[5]))
	m.OLEDTimeout = int32(int8(raw[6]))
	m.Status = helpers.ReadShort(raw, 7)
	return nil
}
