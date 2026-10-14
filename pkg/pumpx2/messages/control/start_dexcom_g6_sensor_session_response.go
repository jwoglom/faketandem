package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// StartDexcomG6SensorSessionResponse is the pump's answer to a
// StartDexcomG6SensorSessionRequest: Status 0 is success.
type StartDexcomG6SensorSessionResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var startDexcomG6SensorSessionResponseProps = messages.Props{OpCode: -77, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "StartDexcomG6SensorSessionRequest"}

func init() {
	messages.Register("StartDexcomG6SensorSessionResponse", &StartDexcomG6SensorSessionResponse{}, startDexcomG6SensorSessionResponseProps,
		messages.Constructor{Params: "int status", Func: NewStartDexcomG6SensorSessionResponse},
	)
}

// NewStartDexcomG6SensorSessionResponse builds the response.
func NewStartDexcomG6SensorSessionResponse(status int32) *StartDexcomG6SensorSessionResponse {
	m := &StartDexcomG6SensorSessionResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *StartDexcomG6SensorSessionResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(startDexcomG6SensorSessionResponseProps, raw)
	if len(raw) != startDexcomG6SensorSessionResponseProps.Size {
		return messages.InvalidLength("StartDexcomG6SensorSessionResponse", len(raw), startDexcomG6SensorSessionResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
