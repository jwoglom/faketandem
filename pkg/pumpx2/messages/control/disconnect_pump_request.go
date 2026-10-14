package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// DisconnectPumpRequest asks the pump to drop the app's connection. It has no
// payload.
type DisconnectPumpRequest struct {
	messages.Base
}

var disconnectPumpRequestProps = messages.Props{OpCode: -66, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", Response: "DisconnectPumpResponse"}

func init() {
	messages.Register("DisconnectPumpRequest", &DisconnectPumpRequest{}, disconnectPumpRequestProps,
		messages.Constructor{Func: NewDisconnectPumpRequest},
	)
}

// NewDisconnectPumpRequest builds the request.
func NewDisconnectPumpRequest() *DisconnectPumpRequest {
	m := &DisconnectPumpRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *DisconnectPumpRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(disconnectPumpRequestProps, raw)
	if len(raw) != disconnectPumpRequestProps.Size {
		return messages.InvalidLength("DisconnectPumpRequest", len(raw), disconnectPumpRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
