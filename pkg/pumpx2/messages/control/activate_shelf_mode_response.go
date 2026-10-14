package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ActivateShelfModeResponse is the pump's answer to an
// ActivateShelfModeRequest. It has no payload.
type ActivateShelfModeResponse struct {
	messages.Base
}

var activateShelfModeResponseProps = messages.Props{OpCode: -69, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "ActivateShelfModeRequest"}

func init() {
	messages.Register("ActivateShelfModeResponse", &ActivateShelfModeResponse{}, activateShelfModeResponseProps)
}

// Parse implements messages.Message. A payload of any length parses.
func (m *ActivateShelfModeResponse) Parse(raw []byte) error {
	m.SetCargo(messages.RemoveSignedRequestHmacBytes(activateShelfModeResponseProps, raw))
	return nil
}
