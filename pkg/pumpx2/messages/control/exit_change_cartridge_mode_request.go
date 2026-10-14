package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ExitChangeCartridgeModeRequest ends change cartridge mode once the new
// cartridge is in; delivery must still be suspended. It has no payload.
type ExitChangeCartridgeModeRequest struct {
	messages.Base
}

var exitChangeCartridgeModeRequestProps = messages.Props{OpCode: -110, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "ExitChangeCartridgeModeResponse"}

func init() {
	messages.Register("ExitChangeCartridgeModeRequest", &ExitChangeCartridgeModeRequest{}, exitChangeCartridgeModeRequestProps,
		messages.Constructor{Func: NewExitChangeCartridgeModeRequest},
	)
}

// NewExitChangeCartridgeModeRequest builds the request.
func NewExitChangeCartridgeModeRequest() *ExitChangeCartridgeModeRequest {
	m := &ExitChangeCartridgeModeRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ExitChangeCartridgeModeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(exitChangeCartridgeModeRequestProps, raw)
	if len(raw) != exitChangeCartridgeModeRequestProps.Size {
		return messages.InvalidLength("ExitChangeCartridgeModeRequest", len(raw), exitChangeCartridgeModeRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
