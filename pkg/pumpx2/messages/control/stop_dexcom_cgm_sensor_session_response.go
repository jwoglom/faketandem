package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// StopDexcomCGMSensorSessionResponse is the pump's answer to a
// StopDexcomCGMSensorSessionRequest: Status 0 is success.
type StopDexcomCGMSensorSessionResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var stopDexcomCGMSensorSessionResponseProps = messages.Props{OpCode: -75, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "StopDexcomCGMSensorSessionRequest"}

func init() {
	messages.Register("StopDexcomCGMSensorSessionResponse", &StopDexcomCGMSensorSessionResponse{}, stopDexcomCGMSensorSessionResponseProps,
		messages.Constructor{Params: "int status", Func: NewStopDexcomCGMSensorSessionResponse},
	)
}

// NewStopDexcomCGMSensorSessionResponse builds the response.
func NewStopDexcomCGMSensorSessionResponse(status int32) *StopDexcomCGMSensorSessionResponse {
	m := &StopDexcomCGMSensorSessionResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *StopDexcomCGMSensorSessionResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(stopDexcomCGMSensorSessionResponseProps, raw)
	if len(raw) != stopDexcomCGMSensorSessionResponseProps.Size {
		return messages.InvalidLength("StopDexcomCGMSensorSessionResponse", len(raw), stopDexcomCGMSensorSessionResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
