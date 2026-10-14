package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// EnterFillTubingModeRequest starts filling the tubing. It has no payload.
type EnterFillTubingModeRequest struct {
	messages.Base
}

var enterFillTubingModeRequestProps = messages.Props{OpCode: -108, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Response: "EnterFillTubingModeResponse"}

func init() {
	messages.Register("EnterFillTubingModeRequest", &EnterFillTubingModeRequest{}, enterFillTubingModeRequestProps,
		messages.Constructor{Func: NewEnterFillTubingModeRequest},
	)
}

// NewEnterFillTubingModeRequest builds the request.
func NewEnterFillTubingModeRequest() *EnterFillTubingModeRequest {
	m := &EnterFillTubingModeRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *EnterFillTubingModeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(enterFillTubingModeRequestProps, raw)
	if len(raw) != enterFillTubingModeRequestProps.Size {
		return messages.InvalidLength("EnterFillTubingModeRequest", len(raw), enterFillTubingModeRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
