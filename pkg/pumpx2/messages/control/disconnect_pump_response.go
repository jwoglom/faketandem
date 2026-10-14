package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// DisconnectPumpResponse is the pump's answer to a DisconnectPumpRequest:
// Status 0 is success.
type DisconnectPumpResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var disconnectPumpResponseProps = messages.Props{OpCode: -65, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, MinAPI: "MOBI_API_V3_5", Request: "DisconnectPumpRequest"}

func init() {
	messages.Register("DisconnectPumpResponse", &DisconnectPumpResponse{}, disconnectPumpResponseProps,
		messages.Constructor{Params: "int status", Func: NewDisconnectPumpResponse},
	)
}

// NewDisconnectPumpResponse builds the response.
func NewDisconnectPumpResponse(status int32) *DisconnectPumpResponse {
	m := &DisconnectPumpResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *DisconnectPumpResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(disconnectPumpResponseProps, raw)
	if len(raw) != disconnectPumpResponseProps.Size {
		return messages.InvalidLength("DisconnectPumpResponse", len(raw), disconnectPumpResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
