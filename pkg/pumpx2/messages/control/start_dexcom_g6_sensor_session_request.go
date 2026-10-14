package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// G6NoSensorCode is the sensor code of a G6 session started without one.
const G6NoSensorCode = 0

// StartDexcomG6SensorSessionRequest starts a Dexcom G6 sensor session with
// the sensor's calibration code, or G6NoSensorCode.
type StartDexcomG6SensorSessionRequest struct {
	messages.Base
	SensorCode int32 `pumpx2:"sensorCode"`
}

var startDexcomG6SensorSessionRequestProps = messages.Props{OpCode: -78, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "StartDexcomG6SensorSessionResponse"}

func init() {
	messages.Register("StartDexcomG6SensorSessionRequest", &StartDexcomG6SensorSessionRequest{}, startDexcomG6SensorSessionRequestProps,
		messages.Constructor{Params: "int sensorCode", Func: NewStartDexcomG6SensorSessionRequest},
	)
}

// NewStartDexcomG6SensorSessionRequest builds the request.
func NewStartDexcomG6SensorSessionRequest(sensorCode int32) *StartDexcomG6SensorSessionRequest {
	m := &StartDexcomG6SensorSessionRequest{SensorCode: sensorCode}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(sensorCode))
	return m
}

// Parse implements messages.Message.
func (m *StartDexcomG6SensorSessionRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(startDexcomG6SensorSessionRequestProps, raw)
	if len(raw) != startDexcomG6SensorSessionRequestProps.Size {
		return messages.InvalidLength("StartDexcomG6SensorSessionRequest", len(raw), startDexcomG6SensorSessionRequestProps.Size)
	}
	m.SetCargo(raw)
	m.SensorCode = helpers.ReadShort(raw, 0)
	return nil
}
