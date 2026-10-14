package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetPumpSoundsResponse is the pump's answer to a SetPumpSoundsRequest:
// Status 0 is success.
type SetPumpSoundsResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setPumpSoundsResponseProps = messages.Props{OpCode: -27, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetPumpSoundsRequest"}

func init() {
	messages.Register("SetPumpSoundsResponse", &SetPumpSoundsResponse{}, setPumpSoundsResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetPumpSoundsResponse},
	)
}

// NewSetPumpSoundsResponse builds the response.
func NewSetPumpSoundsResponse(status int32) *SetPumpSoundsResponse {
	m := &SetPumpSoundsResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetPumpSoundsResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setPumpSoundsResponseProps, raw)
	if len(raw) != setPumpSoundsResponseProps.Size {
		return messages.InvalidLength("SetPumpSoundsResponse", len(raw), setPumpSoundsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
