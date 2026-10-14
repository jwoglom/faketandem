package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetPumpAlertSnoozeResponse is the pump's answer to a
// SetPumpAlertSnoozeRequest: Status 0 is success.
type SetPumpAlertSnoozeResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setPumpAlertSnoozeResponseProps = messages.Props{OpCode: -43, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetPumpAlertSnoozeRequest"}

func init() {
	messages.Register("SetPumpAlertSnoozeResponse", &SetPumpAlertSnoozeResponse{}, setPumpAlertSnoozeResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetPumpAlertSnoozeResponse},
	)
}

// NewSetPumpAlertSnoozeResponse builds the response.
func NewSetPumpAlertSnoozeResponse(status int32) *SetPumpAlertSnoozeResponse {
	m := &SetPumpAlertSnoozeResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetPumpAlertSnoozeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setPumpAlertSnoozeResponseProps, raw)
	if len(raw) != setPumpAlertSnoozeResponseProps.Size {
		return messages.InvalidLength("SetPumpAlertSnoozeResponse", len(raw), setPumpAlertSnoozeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
