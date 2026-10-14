package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// FactoryResetBResponse is the pump's empty answer to a FactoryResetBRequest.
type FactoryResetBResponse struct {
	messages.Base
}

var factoryResetBResponseProps = messages.Props{OpCode: 125, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "FactoryResetBRequest"}

func init() {
	messages.Register("FactoryResetBResponse", &FactoryResetBResponse{}, factoryResetBResponseProps)
}

// Parse implements messages.Message. Any payload parses.
func (m *FactoryResetBResponse) Parse(raw []byte) error {
	m.SetCargo(messages.RemoveSignedRequestHmacBytes(factoryResetBResponseProps, raw))
	return nil
}
