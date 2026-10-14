package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// StopDexcomCGMSensorSessionRequest stops the active Dexcom G6 or G7 sensor
// session.
type StopDexcomCGMSensorSessionRequest struct {
	messages.Base
}

var stopDexcomCGMSensorSessionRequestProps = messages.Props{OpCode: -76, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "StopDexcomCGMSensorSessionResponse"}

func init() {
	messages.Register("StopDexcomCGMSensorSessionRequest", &StopDexcomCGMSensorSessionRequest{}, stopDexcomCGMSensorSessionRequestProps,
		messages.Constructor{Func: NewStopDexcomCGMSensorSessionRequest},
	)
}

// NewStopDexcomCGMSensorSessionRequest is the request, with no payload.
func NewStopDexcomCGMSensorSessionRequest() *StopDexcomCGMSensorSessionRequest {
	m := &StopDexcomCGMSensorSessionRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *StopDexcomCGMSensorSessionRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(stopDexcomCGMSensorSessionRequestProps, raw)
	if len(raw) != stopDexcomCGMSensorSessionRequestProps.Size {
		return messages.InvalidLength("StopDexcomCGMSensorSessionRequest", len(raw), stopDexcomCGMSensorSessionRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
