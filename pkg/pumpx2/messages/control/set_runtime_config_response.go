package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetRuntimeConfigResponse is the pump's answer to a SetRuntimeConfigRequest:
// Status 0 is success.
type SetRuntimeConfigResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setRuntimeConfigResponseProps = messages.Props{OpCode: 123, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetRuntimeConfigRequest"}

func init() {
	messages.Register("SetRuntimeConfigResponse", &SetRuntimeConfigResponse{}, setRuntimeConfigResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetRuntimeConfigResponse},
	)
}

// NewSetRuntimeConfigResponse builds the response.
func NewSetRuntimeConfigResponse(status int32) *SetRuntimeConfigResponse {
	m := &SetRuntimeConfigResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetRuntimeConfigResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setRuntimeConfigResponseProps, raw)
	if len(raw) != setRuntimeConfigResponseProps.Size {
		return messages.InvalidLength("SetRuntimeConfigResponse", len(raw), setRuntimeConfigResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
