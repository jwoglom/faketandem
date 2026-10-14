package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// FactoryResetResponse is the pump's answer to a FactoryResetRequest. It has
// no payload.
type FactoryResetResponse struct {
	messages.Base
}

var factoryResetResponseProps = messages.Props{OpCode: -23, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "FactoryResetRequest"}

func init() {
	messages.Register("FactoryResetResponse", &FactoryResetResponse{}, factoryResetResponseProps)
}

// Parse implements messages.Message. A payload of any length parses.
func (m *FactoryResetResponse) Parse(raw []byte) error {
	m.SetCargo(messages.RemoveSignedRequestHmacBytes(factoryResetResponseProps, raw))
	return nil
}
