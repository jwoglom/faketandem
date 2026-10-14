package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// EnterChangeCartridgeModeRequest starts a cartridge change, which needs
// delivery suspended. It has no payload.
type EnterChangeCartridgeModeRequest struct {
	messages.Base
}

var enterChangeCartridgeModeRequestProps = messages.Props{OpCode: -112, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Response: "EnterChangeCartridgeModeResponse"}

func init() {
	messages.Register("EnterChangeCartridgeModeRequest", &EnterChangeCartridgeModeRequest{}, enterChangeCartridgeModeRequestProps,
		messages.Constructor{Func: NewEnterChangeCartridgeModeRequest},
	)
}

// NewEnterChangeCartridgeModeRequest builds the request.
func NewEnterChangeCartridgeModeRequest() *EnterChangeCartridgeModeRequest {
	m := &EnterChangeCartridgeModeRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *EnterChangeCartridgeModeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(enterChangeCartridgeModeRequestProps, raw)
	if len(raw) != enterChangeCartridgeModeRequestProps.Size {
		return messages.InvalidLength("EnterChangeCartridgeModeRequest", len(raw), enterChangeCartridgeModeRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
