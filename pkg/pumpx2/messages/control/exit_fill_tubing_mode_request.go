package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ExitFillTubingModeRequest stops filling the tubing. It has no payload.
type ExitFillTubingModeRequest struct {
	messages.Base
}

var exitFillTubingModeRequestProps = messages.Props{OpCode: -106, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "ExitFillTubingModeResponse"}

func init() {
	messages.Register("ExitFillTubingModeRequest", &ExitFillTubingModeRequest{}, exitFillTubingModeRequestProps,
		messages.Constructor{Func: NewExitFillTubingModeRequest},
	)
}

// NewExitFillTubingModeRequest builds the request.
func NewExitFillTubingModeRequest() *ExitFillTubingModeRequest {
	m := &ExitFillTubingModeRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ExitFillTubingModeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(exitFillTubingModeRequestProps, raw)
	if len(raw) != exitFillTubingModeRequestProps.Size {
		return messages.InvalidLength("ExitFillTubingModeRequest", len(raw), exitFillTubingModeRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
