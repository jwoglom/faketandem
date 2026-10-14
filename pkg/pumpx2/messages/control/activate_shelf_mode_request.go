package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ActivateShelfModeRequest puts the pump into shelf mode, its low-power
// storage state. It has no payload.
type ActivateShelfModeRequest struct {
	messages.Base
}

var activateShelfModeRequestProps = messages.Props{OpCode: -70, Signed: true, Type: messages.Request, Characteristic: messages.Control, Response: "ActivateShelfModeResponse"}

func init() {
	messages.Register("ActivateShelfModeRequest", &ActivateShelfModeRequest{}, activateShelfModeRequestProps,
		messages.Constructor{Func: NewActivateShelfModeRequest},
	)
}

// NewActivateShelfModeRequest builds the request.
func NewActivateShelfModeRequest() *ActivateShelfModeRequest {
	m := &ActivateShelfModeRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. Unlike most control requests, a payload
// of any length parses.
func (m *ActivateShelfModeRequest) Parse(raw []byte) error {
	m.SetCargo(messages.RemoveSignedRequestHmacBytes(activateShelfModeRequestProps, raw))
	return nil
}
